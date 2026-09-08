package web

import (
	"net/http"
	"strings"
	"testing"

	"rhythms/internal/domain"
)

func TestBuildCalendarVMLayout(t *testing.T) {
	s, habits, entries := newTestServer(t)
	id, _ := habits.Create(t.Context(), testUserID, domain.Habit{Name: "Read", Type: domain.YesNo, Freq: domain.DailyFrequency()})
	entries.Upsert(t.Context(), domain.YesNo, domain.Entry{HabitID: id, Date: domain.NewDate(2026, 9, 15), Value: domain.YesManual})

	h, _ := habits.Get(t.Context(), testUserID, id)
	vm, err := s.buildCalendarVM(t.Context(), h, domain.NewDate(2026, 9, 1))
	if err != nil {
		t.Fatalf("buildCalendarVM: %v", err)
	}

	if vm.MonthLabel != "September 2026" {
		t.Errorf("MonthLabel = %q, want September 2026", vm.MonthLabel)
	}
	if vm.MonthParam != "2026-09" || vm.PrevMonth != "2026-08" || vm.NextMonth != "2026-10" {
		t.Errorf("month params = %+v", vm)
	}

	// September 1, 2026 is a Tuesday, so the first week should have 2 leading
	// blanks (Sun, Mon) before day 1 lands in the Tue slot.
	first := vm.Weeks[0]
	if first[0].Day != 0 || first[1].Day != 0 || first[2].Day != 1 {
		t.Errorf("first week = %+v, want blanks then day 1 on Tuesday", first)
	}

	// Total non-blank days must equal the month's day count (30 for September).
	count := 0
	var day15Completed bool
	for _, week := range vm.Weeks {
		for _, d := range week {
			if d.Day != 0 {
				count++
			}
			if d.Day == 15 {
				day15Completed = d.Completed
			}
		}
	}
	if count != 30 {
		t.Errorf("total days rendered = %d, want 30", count)
	}
	if !day15Completed {
		t.Error("Sep 15 should show as completed (it was checked)")
	}
}

func TestBuildCalendarVMFutureDaysMarked(t *testing.T) {
	s, habits, _ := newTestServer(t)
	id, _ := habits.Create(t.Context(), testUserID, domain.Habit{Name: "Read", Type: domain.YesNo, Freq: domain.DailyFrequency()})
	h, _ := habits.Get(t.Context(), testUserID, id)

	today := domain.Today()
	vm, err := s.buildCalendarVM(t.Context(), h, today.StartOfMonth())
	if err != nil {
		t.Fatalf("buildCalendarVM: %v", err)
	}

	var sawFuture, sawToday bool
	for _, week := range vm.Weeks {
		for _, d := range week {
			if d.Day == 0 {
				continue
			}
			if d.IsToday {
				sawToday = true
				if d.IsFuture {
					t.Error("today should not be marked as future")
				}
			}
			date, _ := domain.ParseDate(d.Date)
			if date.After(today) {
				sawFuture = true
				if !d.IsFuture {
					t.Errorf("date %s is after today but IsFuture=false", d.Date)
				}
			}
		}
	}
	if !sawToday {
		t.Error("expected today's date to appear in its own month")
	}
	_ = sawFuture // not all months necessarily have future days from "today", so no hard assertion
}

func TestHandleHabitCalendarDefaultsToCurrentMonth(t *testing.T) {
	s, habits, _ := newTestServer(t)
	id, _ := habits.Create(t.Context(), testUserID, domain.Habit{Name: "Read"})

	rec := doRequest(t, s, http.MethodGet, "/habits/"+itoa(id)+"/calendar", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), domain.Today().Format("January 2006")) {
		t.Errorf("expected current month label, got: %s", rec.Body.String())
	}
}

func TestHandleHabitCalendarExplicitMonth(t *testing.T) {
	s, habits, _ := newTestServer(t)
	id, _ := habits.Create(t.Context(), testUserID, domain.Habit{Name: "Read"})

	rec := doRequest(t, s, http.MethodGet, "/habits/"+itoa(id)+"/calendar?month=2026-03", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "March 2026") {
		t.Errorf("expected March 2026, got: %s", rec.Body.String())
	}
}

func TestHandleHabitCalendarNotFound(t *testing.T) {
	s, _, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/habits/999/calendar", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleEntryToggleFromCalendarRerendersGrid(t *testing.T) {
	s, habits, _ := newTestServer(t)
	id, _ := habits.Create(t.Context(), testUserID, domain.Habit{Name: "Read", Type: domain.YesNo, Freq: domain.DailyFrequency()})
	today := domain.Today()

	target := "/habits/" + itoa(id) + "/entries/" + today.String() + "?from=calendar&month=" + today.Format("2006-01")
	rec := doRequest(t, s, http.MethodPost, target, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="habit-calendar"`) {
		t.Errorf("expected the calendar grid re-rendered, got: %s", body)
	}
	if !strings.Contains(body, "calendar-day-complete") {
		t.Errorf("expected today's cell to show completed, got: %s", body)
	}
}

func TestHandleEntryEditFormCalendarContext(t *testing.T) {
	s, habits, _ := newTestServer(t)
	id, _ := habits.Create(t.Context(), testUserID, domain.Habit{Name: "Water", Type: domain.Numerical, Unit: "glasses"})
	today := domain.Today()

	target := "/habits/" + itoa(id) + "/entries/" + today.String() + "?from=calendar&month=" + today.Format("2006-01")
	rec := doRequest(t, s, http.MethodGet, target, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `hx-target="#habit-calendar"`) || !strings.Contains(body, `hx-swap="innerHTML"`) {
		t.Errorf("expected form to target the calendar grid, got: %s", body)
	}
	if !strings.Contains(body, "from=calendar") {
		t.Errorf("expected the form's own post to carry the calendar context forward, got: %s", body)
	}
}
