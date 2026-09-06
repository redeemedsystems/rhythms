package server

import (
	"errors"
	"net/http"
	"strings"

	"rhythms/internal/auth"
	"rhythms/internal/store"
)

type AuthPageData struct {
	Base
	Email string
}

func (s *Server) handleRegisterForm(w http.ResponseWriter, r *http.Request) {
	s.render.Page(w, "register.html", AuthPageData{})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	timezone := strings.TrimSpace(r.FormValue("timezone"))
	if timezone == "" {
		timezone = "UTC"
	}

	if email == "" || len(password) < 8 {
		s.render.Page(w, "register.html", AuthPageData{
			Base:  Base{Flash: "Email is required and password must be at least 8 characters."},
			Email: email,
		})
		return
	}

	if _, err := store.GetUserByEmail(s.db, email); err == nil {
		s.render.Page(w, "register.html", AuthPageData{
			Base:  Base{Flash: "An account with that email already exists."},
			Email: email,
		})
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		s.internalError(w, err)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		s.internalError(w, err)
		return
	}

	user, err := store.CreateUser(s.db, email, hash, timezone)
	if err != nil {
		s.internalError(w, err)
		return
	}

	if err := s.startSession(w, r, user); err != nil {
		s.internalError(w, err)
		return
	}
	http.Redirect(w, r, "/today", http.StatusSeeOther)
}

func (s *Server) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	s.render.Page(w, "login.html", AuthPageData{})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	user, err := store.GetUserByEmail(s.db, email)
	if err != nil || !auth.VerifyPassword(user.PasswordHash, password) {
		s.render.Page(w, "login.html", AuthPageData{
			Base:  Base{Flash: "Invalid email or password."},
			Email: email,
		})
		return
	}

	if err := s.startSession(w, r, user); err != nil {
		s.internalError(w, err)
		return
	}
	http.Redirect(w, r, "/today", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if sess := auth.SessionFromContext(r.Context()); sess != nil {
		_ = store.DeleteSession(s.db, sess.ID)
	}
	auth.ClearSessionCookie(w, s.cfg.SecureCookies)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, user *store.User) error {
	sessionID, err := auth.GenerateToken()
	if err != nil {
		return err
	}
	csrfToken, err := auth.GenerateToken()
	if err != nil {
		return err
	}
	if err := store.CreateSession(s.db, sessionID, user.ID, csrfToken, auth.SessionTTL, r.UserAgent()); err != nil {
		return err
	}
	auth.SetSessionCookie(w, sessionID, s.cfg.SecureCookies)
	return nil
}
