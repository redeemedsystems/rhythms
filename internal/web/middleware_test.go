package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequireAuthRedirectsUnauthenticated(t *testing.T) {
	s, _, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/today", nil) // no session cookie
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want /login", loc)
	}
}

func TestRequireAuthHXRedirectsUnauthenticatedHTMXRequest(t *testing.T) {
	s, _, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/today", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (htmx redirects via header, not a real 302)", rec.Code)
	}
	if hx := rec.Header().Get("HX-Redirect"); hx != "/login" {
		t.Errorf("HX-Redirect = %q, want /login", hx)
	}
}

func TestRequireAuthAllowsPublicPathsWithNoSession(t *testing.T) {
	s, _, _ := newTestServer(t)
	for _, path := range []string{"/login", "/healthz", "/manifest.webmanifest"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		s.Routes().ServeHTTP(rec, req)
		if rec.Code == http.StatusFound {
			t.Errorf("%s: got redirected to login, want reachable with no session", path)
		}
	}
}

func TestRequireAuthLoginPageRedirectsAlreadySignedInVisitor(t *testing.T) {
	s, _, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/login", "") // doRequest attaches a valid session cookie
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/" {
		t.Errorf("GET /login while signed in: status=%d Location=%q, want 302 to /", rec.Code, rec.Header().Get("Location"))
	}
}

func TestRequireAdminForbidsNonAdmin(t *testing.T) {
	s, _, _ := newTestServer(t) // testUserID seeded as a non-admin
	rec := doRequest(t, s, http.MethodGet, "/admin/users", "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestRequireAdminAllowsAdmin(t *testing.T) {
	s, _, _ := newTestServer(t)
	users := s.users.(*fakeUserRepo)
	u := users.users[testUserID]
	u.IsAdmin = true
	users.users[testUserID] = u

	rec := doRequest(t, s, http.MethodGet, "/admin/users", "")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for an admin", rec.Code)
	}
}

func TestRequireAuthRejectsExpiredSession(t *testing.T) {
	s, _, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/today", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: signSession(s.sessionSecret, testUserID, time.Now().Add(-time.Hour)),
	})
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, want 302 (expired session must not authenticate)", rec.Code)
	}
}

func TestRequireAuthRejectsSessionForDeletedUser(t *testing.T) {
	s, _, _ := newTestServer(t)
	users := s.users.(*fakeUserRepo)
	delete(users.users, testUserID)

	rec := doRequest(t, s, http.MethodGet, "/today", "")
	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, want 302 (a valid cookie for a deleted user must not authenticate)", rec.Code)
	}
}
