package web

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"rhythms/internal/domain"
)

// fakeGoogleAuth lets handlers_auth tests drive the OAuth callback without
// ever making a real network call to Google.
type fakeGoogleAuth struct {
	identity googleIdentity
	err      error
}

func (f *fakeGoogleAuth) AuthCodeURL(state string) string {
	return "https://accounts.google.com/fake-auth?state=" + state
}

func (f *fakeGoogleAuth) Exchange(ctx context.Context, code string) (googleIdentity, error) {
	if f.err != nil {
		return googleIdentity{}, f.err
	}
	return f.identity, nil
}

// callbackRequest builds a GET /auth/google/callback request carrying a
// matching state cookie+param (as handleGoogleLogin would have set up),
// through the full middleware chain.
func callbackRequest(t *testing.T, s *Server, state string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state="+state+"&code=fake-code", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookieName, Value: state})
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)
	return rec
}

func TestGoogleCallbackInvitedEmailSignsIn(t *testing.T) {
	s, _, _ := newTestServer(t)
	users := s.users.(*fakeUserRepo)
	invitedID, err := users.Create(t.Context(), domain.User{Email: "invited@example.com"})
	if err != nil {
		t.Fatalf("seed invited user: %v", err)
	}
	s.googleAuth = &fakeGoogleAuth{identity: googleIdentity{Email: "invited@example.com", EmailVerified: true, Sub: "google-sub-1"}}

	rec := callbackRequest(t, s, "valid-state")

	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/" {
		t.Fatalf("status=%d Location=%q, want 302 to /", rec.Code, rec.Header().Get("Location"))
	}
	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected a session cookie to be set")
	}
	gotUserID, ok := verifySession(s.sessionSecret, sessionCookie.Value)
	if !ok || gotUserID != invitedID {
		t.Errorf("session cookie userID = %d ok=%v, want %d", gotUserID, ok, invitedID)
	}

	activated, err := users.Get(t.Context(), invitedID)
	if err != nil || activated.GoogleSub != "google-sub-1" {
		t.Errorf("user after first sign-in = %+v err=%v, want GoogleSub=google-sub-1", activated, err)
	}
}

func TestGoogleCallbackUninvitedEmailRejected(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.googleAuth = &fakeGoogleAuth{identity: googleIdentity{Email: "stranger@example.com", EmailVerified: true, Sub: "google-sub-2"}}

	rec := callbackRequest(t, s, "valid-state")

	if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "error=not_invited") {
		t.Errorf("status=%d Location=%q, want 302 with error=not_invited", rec.Code, rec.Header().Get("Location"))
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName && c.Value != "" {
			t.Error("expected no session cookie to be set for an uninvited email")
		}
	}
}

func TestGoogleCallbackUnverifiedEmailRejected(t *testing.T) {
	s, _, _ := newTestServer(t)
	users := s.users.(*fakeUserRepo)
	users.Create(t.Context(), domain.User{Email: "unverified@example.com"})
	s.googleAuth = &fakeGoogleAuth{identity: googleIdentity{Email: "unverified@example.com", EmailVerified: false, Sub: "google-sub-3"}}

	rec := callbackRequest(t, s, "valid-state")

	if !strings.Contains(rec.Header().Get("Location"), "error=email_unverified") {
		t.Errorf("Location = %q, want error=email_unverified", rec.Header().Get("Location"))
	}
}

func TestGoogleCallbackStateMismatchRejected(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.googleAuth = &fakeGoogleAuth{identity: googleIdentity{Email: "invited@example.com", EmailVerified: true, Sub: "google-sub-4"}}

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=attacker-supplied&code=fake-code", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookieName, Value: "what-this-server-actually-set"})
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if !strings.Contains(rec.Header().Get("Location"), "error=state_mismatch") {
		t.Errorf("Location = %q, want error=state_mismatch", rec.Header().Get("Location"))
	}
}

func TestGoogleCallbackNoStateCookieRejected(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.googleAuth = &fakeGoogleAuth{identity: googleIdentity{Email: "invited@example.com", EmailVerified: true, Sub: "google-sub-5"}}

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=some-state&code=fake-code", nil)
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if !strings.Contains(rec.Header().Get("Location"), "error=state_mismatch") {
		t.Errorf("Location = %q, want error=state_mismatch", rec.Header().Get("Location"))
	}
}

func TestGoogleCallbackExchangeFailureRejected(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.googleAuth = &fakeGoogleAuth{err: fmt.Errorf("network error")}

	rec := callbackRequest(t, s, "valid-state")

	if !strings.Contains(rec.Header().Get("Location"), "error=exchange_failed") {
		t.Errorf("Location = %q, want error=exchange_failed", rec.Header().Get("Location"))
	}
}

func TestLogoutClearsSession(t *testing.T) {
	s, _, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodPost, "/logout", "")

	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/login" {
		t.Errorf("status=%d Location=%q, want 302 to /login", rec.Code, rec.Header().Get("Location"))
	}
	var cleared bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("expected logout to clear the session cookie (MaxAge < 0)")
	}

	// The cleared cookie really does deauthenticate a follow-up request.
	req := httptest.NewRequest(http.MethodGet, "/today", nil)
	rec2 := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec2, req)
	if rec2.Code != http.StatusFound || rec2.Header().Get("Location") != "/login" {
		t.Errorf("request with no cookie after logout: status=%d Location=%q, want 302 to /login", rec2.Code, rec2.Header().Get("Location"))
	}
}
