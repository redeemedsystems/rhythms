package web

import (
	"net/http"
	"strings"
	"testing"

	"rhythms/internal/domain"
)

func TestHandleHabitDetail(t *testing.T) {
	s, habits, entries := newTestServer(t)
	id, _ := habits.Create(t.Context(), testUserID, domain.Habit{Name: "Meditate", Freq: domain.DailyFrequency(), Type: domain.YesNo})
	today := domain.Today()
	entries.Upsert(t.Context(), domain.YesNo, domain.Entry{HabitID: id, Date: today, Value: domain.YesManual})

	rec := doRequest(t, s, http.MethodGet, "/habits/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Meditate", "chart-bars", "chart-heatmap"} {
		if !strings.Contains(body, want) {
			t.Errorf("detail page missing %q", want)
		}
	}
}

func TestHandleHabitDetailNotFound(t *testing.T) {
	s, _, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/habits/999", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleHabitDetailEmptyHistoryDoesNotPanic(t *testing.T) {
	s, habits, _ := newTestServer(t)
	habits.Create(t.Context(), testUserID, domain.Habit{Name: "Fresh", Freq: domain.DailyFrequency(), Type: domain.YesNo})

	rec := doRequest(t, s, http.MethodGet, "/habits/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (a habit with no entries yet must still render)", rec.Code)
	}
}
