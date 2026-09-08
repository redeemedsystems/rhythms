// Package web is the HTTP layer: routing, handlers, and html/template
// rendering. Handlers only orchestrate — they load data through the
// internal/domain repo interfaces, call domain functions for any actual
// logic (streaks, scores, the checkmark click-cycle), and render a
// viewmodel. No algorithmic logic lives here.
package web

import (
	"bytes"
	"database/sql"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"

	"rhythms/internal/config"
	"rhythms/internal/domain"
	"rhythms/internal/store"
	webassets "rhythms/web"
)

// Server holds every dependency an HTTP handler might need: the repo
// interfaces for each domain concept, the parsed template set, and
// deployment config.
type Server struct {
	cfg            config.Config
	db             *sql.DB
	habits         domain.HabitRepo
	entries        domain.EntryRepo
	reminders      domain.ReminderRepo
	pushSubs       domain.PushSubscriptionRepo
	users          domain.UserRepo
	sessionSecret  []byte
	googleAuth     googleAuth
	vapidPublicKey string
	tmpl           *template.Template
}

var templateFuncs = template.FuncMap{
	// hasWeekdayBit reports whether bit is set in a Reminder.WeekdayMask —
	// used to pre-check the right weekday checkboxes when editing.
	"hasWeekdayBit": func(mask, bit int) bool { return mask&(1<<bit) != 0 },
}

func NewServer(cfg config.Config, db *sql.DB, vapidPublicKey string, sessionSecret []byte) (*Server, error) {
	tmpl, err := template.New("").Funcs(templateFuncs).ParseFS(webassets.TemplatesFS, "templates/*.html", "templates/pages/*.html", "templates/partials/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:            cfg,
		db:             db,
		habits:         store.NewHabitRepo(db),
		entries:        store.NewEntryRepo(db),
		reminders:      store.NewReminderRepo(db),
		pushSubs:       store.NewPushSubscriptionRepo(db),
		users:          store.NewUserRepo(db),
		sessionSecret:  sessionSecret,
		googleAuth:     newGoogleAuth(cfg),
		vapidPublicKey: vapidPublicKey,
		tmpl:           tmpl,
	}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /today", s.handleToday)
	mux.HandleFunc("GET /dashboard", s.handleDashboard)
	mux.HandleFunc("GET /manifest.webmanifest", s.handleManifest)
	mux.HandleFunc("GET /sw.js", s.handleServiceWorker)
	mux.HandleFunc("POST /push/subscribe", s.handlePushSubscribe)
	mux.HandleFunc("POST /push/unsubscribe", s.handlePushUnsubscribe)

	mux.HandleFunc("GET /habits/new", s.handleHabitNewForm)
	mux.HandleFunc("GET /habits/{id}/export.csv", s.handleHabitExportCSV)
	mux.HandleFunc("POST /habits", s.handleHabitCreate)
	mux.HandleFunc("GET /habits/{id}", s.handleHabitDetail)
	mux.HandleFunc("GET /habits/{id}/calendar", s.handleHabitCalendar)
	mux.HandleFunc("GET /habits/{id}/edit", s.handleHabitEditForm)
	mux.HandleFunc("POST /habits/{id}", s.handleHabitUpdate)
	mux.HandleFunc("POST /habits/{id}/archive", s.handleHabitArchiveToggle)
	mux.HandleFunc("POST /habits/{id}/delete", s.handleHabitDelete)
	mux.HandleFunc("POST /habits/reorder", s.handleHabitReorder)
	mux.HandleFunc("GET /habits/{id}/entries/{date}", s.handleEntryEditForm)
	mux.HandleFunc("POST /habits/{id}/entries/{date}", s.handleEntryToggle)

	// Auth: /login, /auth/google/callback, and /logout are all in
	// isPublicPath's allowlist (middleware.go) — requireAuth lets them
	// through unauthenticated by design, since they're how a session gets
	// established in the first place. There's no server-initiated
	// "/auth/google/login" redirect route (unlike the Authorization Code
	// flow) — Google Identity Services' button on login.html posts
	// straight to the callback once someone picks an account.
	mux.HandleFunc("GET /login", s.handleLoginPage)
	mux.HandleFunc("POST /auth/google/callback", s.handleGoogleCallback)
	mux.HandleFunc("POST /logout", s.handleLogout)

	// Admin-only. /backup lives here too, not with the other GETs above —
	// it's a full raw-database dump (every user's data, no per-habit
	// scoping is even possible), so under multi-tenancy it must never be
	// reachable by an ordinary signed-in user.
	mux.Handle("GET /admin/users", requireAdmin(http.HandlerFunc(s.handleAdminUsers)))
	mux.Handle("POST /admin/users", requireAdmin(http.HandlerFunc(s.handleAdminUserCreate)))
	mux.Handle("POST /admin/users/{id}/delete", requireAdmin(http.HandlerFunc(s.handleAdminUserDelete)))
	mux.Handle("GET /backup", requireAdmin(http.HandlerFunc(s.handleBackup)))

	staticFS, err := fs.Sub(webassets.StaticFS, "static")
	if err != nil {
		panic(err) // programmer error: embed FS is malformed
	}
	mux.Handle("GET /static/", cacheStatic(http.StripPrefix("/static/", http.FileServerFS(staticFS))))

	var handler http.Handler = mux
	handler = s.requireAuth(handler)
	handler = logging(handler)
	handler = recoverer(handler)
	return handler
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("ok"))
}

type layoutData struct {
	Body           template.HTML
	VapidPublicKey string
	User           *domain.User
}

// render executes the named page-body template into a buffer, then wraps it
// in the shared layout. Keeping pages out of the layout's own template
// namespace avoids the {{define "content"}} collision described in layout.html.
// Takes r (rather than just a *domain.User) so callers can't forget to
// thread the current user through — it's always exactly whatever requireAuth
// already resolved for this request.
func (s *Server) render(w http.ResponseWriter, r *http.Request, pageTemplate string, data any) {
	var body bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&body, pageTemplate, data); err != nil {
		s.serverError(w, err)
		return
	}
	if err := s.tmpl.ExecuteTemplate(w, "layout", layoutData{
		Body:           template.HTML(body.String()),
		VapidPublicKey: s.vapidPublicKey,
		User:           userFromContext(r.Context()),
	}); err != nil {
		s.serverError(w, err)
	}
}

// renderPartial executes a fragment template directly, with no layout — used
// for HTMX swap responses.
func (s *Server) renderPartial(w http.ResponseWriter, tmplName string, data any) {
	if err := s.tmpl.ExecuteTemplate(w, tmplName, data); err != nil {
		s.serverError(w, err)
	}
}

func (s *Server) serverError(w http.ResponseWriter, err error) {
	slog.Error("handler error", "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
