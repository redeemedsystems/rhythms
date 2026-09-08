package web

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"rhythms/internal/domain"
)

// fakeGoogleAuth lets handlers_auth tests drive the ID-token callback
// without ever making a real network call to Google.
type fakeGoogleAuth struct {
	identity googleIdentity
	err      error
}

func (f *fakeGoogleAuth) VerifyIDToken(ctx context.Context, idToken string) (googleIdentity, error) {
	if f.err != nil {
		return googleIdentity{}, f.err
	}
	return f.identity, nil
}

// callbackRequest builds the POST /auth/google/callback request Google
// Identity Services' button would submit: a form body with "credential"
// (the ID token — opaque to our fake, which ignores it) and the
// g_csrf_token double-submit field, plus a matching g_csrf_token cookie.
func callbackRequest(t *testing.T, s *Server, csrfToken string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"credential": {"fake-id-token"}, "g_csrf_token": {csrfToken}}
	req := httptest.NewRequest(http.MethodPost, "/auth/google/callback", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
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

	rec := callbackRequest(t, s, "matching-csrf-token")

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

	rec := callbackRequest(t, s, "matching-csrf-token")

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

	rec := callbackRequest(t, s, "matching-csrf-token")

	if !strings.Contains(rec.Header().Get("Location"), "error=email_unverified") {
		t.Errorf("Location = %q, want error=email_unverified", rec.Header().Get("Location"))
	}
}

func TestGoogleCallbackCSRFMismatchRejected(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.googleAuth = &fakeGoogleAuth{identity: googleIdentity{Email: "invited@example.com", EmailVerified: true, Sub: "google-sub-4"}}

	form := url.Values{"credential": {"fake-id-token"}, "g_csrf_token": {"attacker-supplied"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/google/callback", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "what-google-actually-set"})
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if !strings.Contains(rec.Header().Get("Location"), "error=state_mismatch") {
		t.Errorf("Location = %q, want error=state_mismatch", rec.Header().Get("Location"))
	}
}

func TestGoogleCallbackNoCSRFCookieRejected(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.googleAuth = &fakeGoogleAuth{identity: googleIdentity{Email: "invited@example.com", EmailVerified: true, Sub: "google-sub-5"}}

	form := url.Values{"credential": {"fake-id-token"}, "g_csrf_token": {"some-token"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/google/callback", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if !strings.Contains(rec.Header().Get("Location"), "error=state_mismatch") {
		t.Errorf("Location = %q, want error=state_mismatch", rec.Header().Get("Location"))
	}
}

func TestGoogleCallbackVerifyFailureRejected(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.googleAuth = &fakeGoogleAuth{err: fmt.Errorf("network error")}

	rec := callbackRequest(t, s, "matching-csrf-token")

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
