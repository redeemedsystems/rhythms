package web

import (
	"net/http"
	"strings"
	"testing"

	"rhythms/internal/domain"
)

func TestHandleDashboardNoHabits(t *testing.T) {
	s, _, _ := newTestServer(t)

	rec := doRequest(t, s, http.MethodGet, "/dashboard", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Add a") {
		t.Errorf("dashboard body missing empty-state message: %s", rec.Body.String())
	}
}

func TestHandleDashboardShowsStatsAndHabits(t *testing.T) {
	s, habits, entries := newTestServer(t)
	id, _ := habits.Create(t.Context(), testUserID, domain.Habit{Name: "Meditate", Freq: domain.DailyFrequency(), Type: domain.YesNo})
	today := domain.Today()
	entries.Upsert(t.Context(), domain.YesNo, domain.Entry{HabitID: id, Date: today, Value: domain.YesManual})

	rec := doRequest(t, s, http.MethodGet, "/dashboard", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Meditate", "chart-line", "1/1", "streak-badge", "score-badge", "today-done-badge"} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard body missing %q", want)
		}
	}
}

func TestHandleDashboardEmptyHistoryDoesNotPanic(t *testing.T) {
	s, habits, _ := newTestServer(t)
	habits.Create(t.Context(), testUserID, domain.Habit{Name: "Fresh", Freq: domain.DailyFrequency(), Type: domain.YesNo})

	rec := doRequest(t, s, http.MethodGet, "/dashboard", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (a habit with no entries yet must still render)", rec.Code)
	}
}
