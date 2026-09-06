package web

import (
	"net/http"
	"strings"
	"testing"

	"rhythms/internal/domain"
)

func TestHandleTodayHidesCompletedHabits(t *testing.T) {
	s, habits, entries := newTestServer(t)
	doneID, _ := habits.Create(t.Context(), domain.Habit{Name: "Done today"})
	pendingID, _ := habits.Create(t.Context(), domain.Habit{Name: "Still pending"})
	today := domain.Today()
	entries.Upsert(t.Context(), domain.YesNo, domain.Entry{HabitID: doneID, Date: today, Value: domain.YesManual})

	rec := doRequest(t, s, http.MethodGet, "/today", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "Done today") {
		t.Error("completed habit should not appear on /today")
	}
	if !strings.Contains(body, "Still pending") {
		t.Error("incomplete habit should appear on /today")
	}
	if !strings.Contains(body, domain.Today().Format("Monday, January 2")) {
		t.Errorf("expected today's date in the page, got: %s", body)
	}
	_ = pendingID
}

func TestHandleTodayEmptyState(t *testing.T) {
	s, _, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/today", "")
	if !strings.Contains(rec.Body.String(), "Nothing left for today") {
		t.Errorf("expected empty-state message, got: %s", rec.Body.String())
	}
}

func TestHandleEntryToggleFromTodayRemovesCompletedRow(t *testing.T) {
	s, habits, _ := newTestServer(t)
	id, _ := habits.Create(t.Context(), domain.Habit{Name: "Read"})
	today := domain.Today().String()

	rec := doRequest(t, s, http.MethodPost, "/habits/"+itoa(id)+"/entries/"+today+"?from=today", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); strings.TrimSpace(body) != "" {
		t.Errorf("expected empty body once completed from /today, got: %q", body)
	}
}

func TestHandleHabitExportCSV(t *testing.T) {
	s, habits, entries := newTestServer(t)
	id, _ := habits.Create(t.Context(), domain.Habit{Name: "Meditate"})
	today := domain.Today()
	entries.Upsert(t.Context(), domain.YesNo, domain.Entry{HabitID: id, Date: today, Value: domain.YesManual})

	rec := doRequest(t, s, http.MethodGet, "/habits/1/export.csv", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/csv" {
		t.Errorf("Content-Type = %q, want text/csv", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Date,Value,Notes") {
		t.Errorf("missing CSV header: %s", body)
	}
	if !strings.Contains(body, today.String()+",YES_MANUAL,") {
		t.Errorf("missing expected row: %s", body)
	}
}

func TestHandleHabitExportCSVNotFound(t *testing.T) {
	s, _, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/habits/999/export.csv", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestSaveAndLoadReminder(t *testing.T) {
	s, habits, _ := newTestServer(t)
	id, _ := habits.Create(t.Context(), domain.Habit{Name: "Stretch"})

	form := "name=Stretch&color=0&reminder_enabled=on&reminder_time=07:15&reminder_weekday=0&reminder_weekday=2"
	rec := doRequest(t, s, http.MethodPost, "/habits/"+itoa(id), form)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200", rec.Code)
	}

	rec = doRequest(t, s, http.MethodGet, "/habits/"+itoa(id)+"/edit", "")
	body := rec.Body.String()
	if !strings.Contains(body, `value="07:15"`) {
		t.Errorf("edit form missing saved reminder time: %s", body)
	}
	if !strings.Contains(body, `name="reminder_enabled" checked`) {
		t.Errorf("edit form should show reminder enabled: %s", body)
	}
}
