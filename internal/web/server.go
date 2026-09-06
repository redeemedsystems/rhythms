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

type Server struct {
	cfg            config.Config
	db             *sql.DB
	habits         domain.HabitRepo
	entries        domain.EntryRepo
	reminders      domain.ReminderRepo
	pushSubs       domain.PushSubscriptionRepo
	vapidPublicKey string
	tmpl           *template.Template
}

var templateFuncs = template.FuncMap{
	// hasWeekdayBit reports whether bit is set in a Reminder.WeekdayMask —
	// used to pre-check the right weekday checkboxes when editing.
	"hasWeekdayBit": func(mask, bit int) bool { return mask&(1<<bit) != 0 },
}

func NewServer(cfg config.Config, db *sql.DB, vapidPublicKey string) (*Server, error) {
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
		vapidPublicKey: vapidPublicKey,
		tmpl:           tmpl,
	}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /today", s.handleToday)
	mux.HandleFunc("GET /backup", s.handleBackup)
	mux.HandleFunc("GET /manifest.webmanifest", s.handleManifest)
	mux.HandleFunc("GET /sw.js", s.handleServiceWorker)
	mux.HandleFunc("POST /push/subscribe", s.handlePushSubscribe)
	mux.HandleFunc("POST /push/unsubscribe", s.handlePushUnsubscribe)

	mux.HandleFunc("GET /habits/new", s.handleHabitNewForm)
	mux.HandleFunc("GET /habits/{id}/export.csv", s.handleHabitExportCSV)
	mux.HandleFunc("POST /habits", s.handleHabitCreate)
	mux.HandleFunc("GET /habits/{id}", s.handleHabitDetail)
	mux.HandleFunc("GET /habits/{id}/edit", s.handleHabitEditForm)
	mux.HandleFunc("POST /habits/{id}", s.handleHabitUpdate)
	mux.HandleFunc("POST /habits/{id}/archive", s.handleHabitArchiveToggle)
	mux.HandleFunc("POST /habits/{id}/delete", s.handleHabitDelete)
	mux.HandleFunc("POST /habits/reorder", s.handleHabitReorder)
	mux.HandleFunc("GET /habits/{id}/entries/{date}", s.handleEntryEditForm)
	mux.HandleFunc("POST /habits/{id}/entries/{date}", s.handleEntryToggle)

	staticFS, err := fs.Sub(webassets.StaticFS, "static")
	if err != nil {
		panic(err) // programmer error: embed FS is malformed
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))

	var handler http.Handler = mux
	handler = basicAuth(s.cfg.BasicAuthUser, s.cfg.BasicAuthPass, handler)
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
}

// render executes the named page-body template into a buffer, then wraps it
// in the shared layout. Keeping pages out of the layout's own template
// namespace avoids the {{define "content"}} collision described in layout.html.
func (s *Server) render(w http.ResponseWriter, pageTemplate string, data any) {
	var body bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&body, pageTemplate, data); err != nil {
		s.serverError(w, err)
		return
	}
	if err := s.tmpl.ExecuteTemplate(w, "layout", layoutData{Body: template.HTML(body.String()), VapidPublicKey: s.vapidPublicKey}); err != nil {
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
