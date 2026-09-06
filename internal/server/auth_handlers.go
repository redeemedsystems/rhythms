package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"rhythms/internal/auth"
	"rhythms/internal/store"
)

const passwordResetTTL = 1 * time.Hour

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
	data := AuthPageData{}
	if r.URL.Query().Get("reset") == "success" {
		data.Flash = "Your password has been reset. Please log in."
	}
	s.render.Page(w, "login.html", data)
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

type ForgotPasswordPageData struct {
	Base
	Email string
	Sent  bool
}

func (s *Server) handleForgotPasswordForm(w http.ResponseWriter, r *http.Request) {
	s.render.Page(w, "forgot_password.html", ForgotPasswordPageData{})
}

// handleForgotPassword always renders the same "check your email" response,
// whether or not the address is registered, so the form can't be used to
// enumerate accounts.
func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))

	user, err := store.GetUserByEmail(s.db, email)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		s.internalError(w, err)
		return
	}

	if err == nil {
		token, genErr := auth.GenerateToken()
		if genErr != nil {
			s.internalError(w, genErr)
			return
		}
		if createErr := store.CreatePasswordReset(s.db, auth.HashToken(token), user.ID, passwordResetTTL); createErr != nil {
			s.internalError(w, createErr)
			return
		}

		resetURL := s.cfg.BaseURL + "/reset-password?token=" + token
		body := fmt.Sprintf(
			"Someone requested a password reset for your Rhythms account.\n\nReset your password: %s\n\nThis link expires in an hour. If you didn't request this, you can ignore this email.",
			resetURL,
		)
		if sendErr := s.mailer.Send(user.Email, "Reset your Rhythms password", body); sendErr != nil {
			slog.Error("send password reset email", "err", sendErr)
		}
	}

	s.render.Page(w, "forgot_password.html", ForgotPasswordPageData{Email: email, Sent: true})
}

type ResetPasswordPageData struct {
	Base
	Token string
}

func (s *Server) handleResetPasswordForm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		s.render.Page(w, "reset_password.html", ResetPasswordPageData{})
		return
	}
	if _, err := store.GetPasswordReset(s.db, auth.HashToken(token)); err != nil {
		s.render.Page(w, "reset_password.html", ResetPasswordPageData{
			Base: Base{Flash: "That reset link is invalid or has expired."},
		})
		return
	}
	s.render.Page(w, "reset_password.html", ResetPasswordPageData{Token: token})
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	token := r.FormValue("token")
	password := r.FormValue("password")

	reset, err := store.GetPasswordReset(s.db, auth.HashToken(token))
	if err != nil {
		s.render.Page(w, "reset_password.html", ResetPasswordPageData{
			Base: Base{Flash: "That reset link is invalid or has expired."},
		})
		return
	}

	if len(password) < 8 {
		s.render.Page(w, "reset_password.html", ResetPasswordPageData{
			Base:  Base{Flash: "Password must be at least 8 characters."},
			Token: token,
		})
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		s.internalError(w, err)
		return
	}
	if err := store.UpdatePassword(s.db, reset.UserID, hash); err != nil {
		s.internalError(w, err)
		return
	}
	if err := store.MarkPasswordResetUsed(s.db, reset.TokenHash); err != nil {
		s.internalError(w, err)
		return
	}
	if err := store.DeleteSessionsForUser(s.db, reset.UserID); err != nil {
		s.internalError(w, err)
		return
	}

	http.Redirect(w, r, "/login?reset=success", http.StatusSeeOther)
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
