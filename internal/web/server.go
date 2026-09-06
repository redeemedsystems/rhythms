package web

import (
	"database/sql"
	"html/template"
	"io/fs"
	"net/http"

	"rhythms/internal/config"
	webassets "rhythms/web"
)

type Server struct {
	cfg  config.Config
	db   *sql.DB
	tmpl *template.Template
}

func NewServer(cfg config.Config, db *sql.DB) (*Server, error) {
	tmpl, err := template.ParseFS(webassets.TemplatesFS, "templates/*.html", "templates/pages/*.html", "templates/partials/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{cfg: cfg, db: db, tmpl: tmpl}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /{$}", s.handleIndex)

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

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if err := s.tmpl.ExecuteTemplate(w, "layout", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
