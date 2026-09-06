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
	cfg     config.Config
	habits  domain.HabitRepo
	entries domain.EntryRepo
	tmpl    *template.Template
}

func NewServer(cfg config.Config, db *sql.DB) (*Server, error) {
	tmpl, err := template.ParseFS(webassets.TemplatesFS, "templates/*.html", "templates/pages/*.html", "templates/partials/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:     cfg,
		habits:  store.NewHabitRepo(db),
		entries: store.NewEntryRepo(db),
		tmpl:    tmpl,
	}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /{$}", s.handleIndex)

	mux.HandleFunc("GET /habits/new", s.handleHabitNewForm)
	mux.HandleFunc("POST /habits", s.handleHabitCreate)
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

// render executes the named page-body template into a buffer, then wraps it
// in the shared layout. Keeping pages out of the layout's own template
// namespace avoids the {{define "content"}} collision described in layout.html.
func (s *Server) render(w http.ResponseWriter, pageTemplate string, data any) {
	var body bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&body, pageTemplate, data); err != nil {
		s.serverError(w, err)
		return
	}
	if err := s.tmpl.ExecuteTemplate(w, "layout", struct{ Body template.HTML }{template.HTML(body.String())}); err != nil {
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
