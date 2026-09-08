package web

import (
	"net/http"
	"strings"
	"testing"

	"rhythms/internal/domain"
)

func makeAdmin(t *testing.T, s *Server) {
	t.Helper()
	users := s.users.(*fakeUserRepo)
	u := users.users[testUserID]
	u.IsAdmin = true
	users.users[testUserID] = u
}

func TestAdminInviteCreatesUser(t *testing.T) {
	s, _, _ := newTestServer(t)
	makeAdmin(t, s)

	rec := doRequest(t, s, http.MethodPost, "/admin/users", "email=new%40example.com")
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/admin/users" {
		t.Fatalf("status=%d Location=%q, want 302 to /admin/users", rec.Code, rec.Header().Get("Location"))
	}

	users := s.users.(*fakeUserRepo)
	got, err := users.GetByEmail(t.Context(), "new@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got.GoogleSub != "" {
		t.Errorf("newly invited user GoogleSub = %q, want empty (not yet signed in)", got.GoogleSub)
	}
}

func TestAdminInviteDuplicateEmailFriendlyError(t *testing.T) {
	s, _, _ := newTestServer(t)
	makeAdmin(t, s)
	users := s.users.(*fakeUserRepo)
	users.Create(t.Context(), domain.User{Email: "dup@example.com"})

	rec := doRequest(t, s, http.MethodPost, "/admin/users", "email=dup%40example.com")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (re-rendered with an error, not a 500)", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "already been invited") {
		t.Errorf("body missing duplicate-invite error: %s", rec.Body.String())
	}
}

func TestAdminInviteRequiresAdmin(t *testing.T) {
	s, _, _ := newTestServer(t) // testUserID is not an admin
	rec := doRequest(t, s, http.MethodPost, "/admin/users", "email=new%40example.com")
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestAdminDeleteUser(t *testing.T) {
	s, _, _ := newTestServer(t)
	makeAdmin(t, s)
	users := s.users.(*fakeUserRepo)
	id, _ := users.Create(t.Context(), domain.User{Email: "temp@example.com"})

	rec := doRequest(t, s, http.MethodPost, "/admin/users/"+itoa(id)+"/delete", "")
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if _, err := users.Get(t.Context(), id); err == nil {
		t.Error("expected user to be deleted")
	}
}

func TestAdminCannotDeleteOwnAccount(t *testing.T) {
	s, _, _ := newTestServer(t)
	makeAdmin(t, s)

	rec := doRequest(t, s, http.MethodPost, "/admin/users/"+itoa(testUserID)+"/delete", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (re-rendered with an error, not a redirect)", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "remove your own account") {
		t.Errorf("body missing self-delete guard message: %s", rec.Body.String())
	}
	users := s.users.(*fakeUserRepo)
	if _, err := users.Get(t.Context(), testUserID); err != nil {
		t.Error("expected the admin's own account to still exist")
	}
}
