package web

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"rhythms/internal/config"
	"rhythms/internal/domain"
	webassets "rhythms/web"
)

func newTestServer(t *testing.T) (*Server, *fakeHabitRepo, *fakeEntryRepo) {
	t.Helper()
	tmpl, err := template.ParseFS(webassets.TemplatesFS, "templates/*.html", "templates/pages/*.html", "templates/partials/*.html")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	habits := newFakeHabitRepo()
	entries := newFakeEntryRepo()
	return &Server{cfg: config.Config{}, habits: habits, entries: entries, tmpl: tmpl}, habits, entries
}

func doRequest(t *testing.T, s *Server, method, target string, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody *strings.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	} else {
		reqBody = strings.NewReader("")
	}
	req := httptest.NewRequest(method, target, reqBody)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)
	return rec
}

func TestHandleIndexEmpty(t *testing.T) {
	s, _, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "No habits yet") {
		t.Errorf("body missing empty-state message: %s", rec.Body.String())
	}
}

func TestHandleHabitCreateAndList(t *testing.T) {
	s, habits, _ := newTestServer(t)

	rec := doRequest(t, s, http.MethodPost, "/habits", "name=Meditate&question=Did+you%3F&color=5")
	if rec.Code != http.StatusOK {
		t.Fatalf("create status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("HX-Redirect"); got != "/" {
		t.Errorf("HX-Redirect = %q, want /", got)
	}
	if len(habits.habits) != 1 {
		t.Fatalf("expected 1 habit stored, got %d", len(habits.habits))
	}

	rec = doRequest(t, s, http.MethodGet, "/", "")
	if !strings.Contains(rec.Body.String(), "Meditate") {
		t.Errorf("index body missing created habit: %s", rec.Body.String())
	}
}

func TestHandleHabitCreateValidationError(t *testing.T) {
	s, habits, _ := newTestServer(t)

	rec := doRequest(t, s, http.MethodPost, "/habits", "name=&question=x")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (re-rendered form)", rec.Code)
	}
	if got := rec.Header().Get("HX-Redirect"); got != "" {
		t.Errorf("HX-Redirect = %q, want empty on validation error", got)
	}
	if !strings.Contains(rec.Body.String(), "Name is required") {
		t.Errorf("body missing validation error: %s", rec.Body.String())
	}
	// The error response must be the bare form partial, not a full HTML
	// document — it's swapped in via hx-swap="outerHTML" on the <form> itself.
	if strings.Contains(rec.Body.String(), "<html") {
		t.Errorf("validation-error response should not include a full <html> document: %s", rec.Body.String())
	}
	if len(habits.habits) != 0 {
		t.Errorf("expected no habit stored after validation error, got %d", len(habits.habits))
	}
}

func TestHandleEntryToggleCycle(t *testing.T) {
	s, habits, _ := newTestServer(t)
	id, _ := habits.Create(t.Context(), domain.Habit{Name: "Read"})
	today := domain.Today().String()

	rec := doRequest(t, s, http.MethodPost, "/habits/"+itoa(id)+"/entries/"+today, "")
	if !strings.Contains(rec.Body.String(), "checkmark-2") {
		t.Errorf("first toggle: expected checkmark-2 (YesManual), got: %s", rec.Body.String())
	}

	rec = doRequest(t, s, http.MethodPost, "/habits/"+itoa(id)+"/entries/"+today, "")
	if !strings.Contains(rec.Body.String(), "checkmark-0") {
		t.Errorf("second toggle: expected checkmark-0 (No), got: %s", rec.Body.String())
	}
}

func TestHandleHabitArchiveExcludesFromActiveList(t *testing.T) {
	s, habits, _ := newTestServer(t)
	id, _ := habits.Create(t.Context(), domain.Habit{Name: "Stretch"})

	rec := doRequest(t, s, http.MethodPost, "/habits/"+itoa(id)+"/archive", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("archive status = %d, want 200", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "Stretch") {
		t.Errorf("archived habit should not appear in the re-rendered active list: %s", rec.Body.String())
	}

	rec = doRequest(t, s, http.MethodGet, "/?archived=1", "")
	if !strings.Contains(rec.Body.String(), "Stretch") {
		t.Errorf("archived habit should appear in the archived view: %s", rec.Body.String())
	}
}

func TestHandleHabitEditNotFound(t *testing.T) {
	s, _, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/habits/999/edit", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
