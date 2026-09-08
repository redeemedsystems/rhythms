package web

import (
	"errors"
	"log/slog"
	"net/http"

	"rhythms/internal/store"
)

// g_csrf_token is Google Identity Services' own convention, not something
// this app invents: its client-side script sets this cookie on the login
// page, and includes the same value as a form field when it posts the
// signed-in credential to login_uri — the server comparing the two is
// exactly the double-submit CSRF check Google's own docs specify for this
// integration.
const csrfCookieName = "g_csrf_token"

type loginPageVM struct {
	Error          string
	GoogleClientID string
	CallbackURL    string
}

// handleLoginPage renders the sign-in page. An already-signed-in visitor
// (requireAuth still resolves the session on public paths) is bounced
// straight to "/" rather than shown a login page they don't need.
func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if userFromContext(r.Context()) != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	s.render(w, r, "page_login", loginPageVM{
		Error:          r.URL.Query().Get("error"),
		GoogleClientID: s.cfg.GoogleClientID,
		CallbackURL:    s.cfg.BaseURL + "/auth/google/callback",
	})
}

// handleGoogleCallback receives the POST Google Identity Services' "Sign in
// with Google" button submits directly from the browser (login.html) once
// someone picks an account — no server-initiated redirect to Google at
// all, unlike the Authorization Code flow. It verifies the CSRF
// double-submit cookie, verifies the ID token, and either activates an
// invited user (first sign-in) or logs in an already-activated one. An
// email with no matching User row at all — never invited — is rejected;
// there is no open signup.
func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "could not parse form", http.StatusBadRequest)
		return
	}

	csrfCookie, err := r.Cookie(csrfCookieName)
	if err != nil || csrfCookie.Value == "" || csrfCookie.Value != r.PostForm.Get(csrfCookieName) {
		http.Redirect(w, r, "/login?error=state_mismatch", http.StatusFound)
		return
	}

	identity, err := s.googleAuth.VerifyIDToken(r.Context(), r.PostForm.Get("credential"))
	if err != nil {
		slog.Error("google id token verification failed", "error", err)
		http.Redirect(w, r, "/login?error=exchange_failed", http.StatusFound)
		return
	}
	if !identity.EmailVerified {
		http.Redirect(w, r, "/login?error=email_unverified", http.StatusFound)
		return
	}

	u, err := s.users.GetByEmail(r.Context(), identity.Email)
	if errors.Is(err, store.ErrNotFound) {
		http.Redirect(w, r, "/login?error=not_invited", http.StatusFound)
		return
	}
	if err != nil {
		s.serverError(w, err)
		return
	}

	switch u.GoogleSub {
	case "":
		if err := s.users.SetGoogleSub(r.Context(), u.ID, identity.Sub); err != nil {
			s.serverError(w, err)
			return
		}
	case identity.Sub:
		// Already activated, signing in again — nothing to do.
	default:
		// Shouldn't happen (an email uniquely maps to one Google account),
		// but never silently accept a sub that doesn't match what this
		// user activated with.
		slog.Error("google sub mismatch on sign-in", "email", identity.Email)
		http.Redirect(w, r, "/login?error=account_mismatch", http.StatusFound)
		return
	}

	s.setSessionCookie(w, u.ID)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.clearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusFound)
}
