package server

import (
	"log/slog"
	"net/http"

	"rhythms/internal/auth"
)

// baseFor builds the Base fields for an authenticated page render.
func (s *Server) baseFor(r *http.Request, nav string) Base {
	return Base{LoggedIn: true, Nav: nav, CSRFToken: s.csrfFor(r)}
}

func (s *Server) csrfFor(r *http.Request) string {
	if sess := auth.SessionFromContext(r.Context()); sess != nil {
		return sess.CSRFToken
	}
	return ""
}

func (s *Server) internalError(w http.ResponseWriter, err error) {
	slog.Error("internal error", "err", err)
	http.Error(w, "Something went wrong. Please try again.", http.StatusInternalServerError)
}
