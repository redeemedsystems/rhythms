package web

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"rhythms/internal/store"
)

const (
	oauthStateCookieName = "rhythms_oauth_state"
	oauthStateDuration   = 10 * time.Minute
)

type loginPageVM struct {
	Error string
}

// handleLoginPage renders the sign-in page. An already-signed-in visitor
// (requireAuth still resolves the session on public paths) is bounced
// straight to "/" rather than shown a login page they don't need.
func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if userFromContext(r.Context()) != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	s.render(w, r, "page_login", loginPageVM{Error: r.URL.Query().Get("error")})
}

// handleGoogleLogin starts the OAuth flow: stash a random state value in a
// short-lived cookie, then redirect to Google with the same value, so the
// callback can confirm the response actually answers a request this
// server made (rather than a forged redirect to the callback URL).
func (s *Server) handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := randomState()
	if err != nil {
		s.serverError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(oauthStateDuration.Seconds()),
	})
	http.Redirect(w, r, s.googleAuth.AuthCodeURL(state), http.StatusFound)
}

// handleGoogleCallback completes the OAuth flow: validate the state,
// exchange the code for the signed-in Google account's identity, and
// either activate an invited user (first sign-in) or just log in an
// already-activated one. An email with no matching User row at all — never
// invited — is rejected; there is no open signup.
func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(oauthStateCookieName)
	http.SetCookie(w, &http.Cookie{Name: oauthStateCookieName, Value: "", Path: "/", MaxAge: -1})
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		http.Redirect(w, r, "/login?error=state_mismatch", http.StatusFound)
		return
	}

	identity, err := s.googleAuth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		slog.Error("google oauth exchange failed", "error", err)
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

func randomState() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
