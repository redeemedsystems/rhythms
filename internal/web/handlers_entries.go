package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"rhythms/internal/domain"
	"rhythms/internal/store"
)

type numericEntryFormVM struct {
	HabitID        int64
	Date           string
	Unit           string
	Value          float64
	PostQuery      string // appended to the form's hx-post URL, e.g. "?from=today"
	TargetSelector string // hx-target for the form's own submit
	SwapMode       string // hx-swap for the form's own submit
}

// buildNumericEntryFormVM computes where a numeric habit's inline edit form
// should submit its result, based on which view opened it — the habit list
// (from=""), /today (from="today"), or a habit's calendar
// (from="calendar", carrying which month to re-render).
func buildNumericEntryFormVM(habitID int64, date, unit string, value float64, from, month string) numericEntryFormVM {
	vm := numericEntryFormVM{HabitID: habitID, Date: date, Unit: unit, Value: value}
	switch from {
	case "calendar":
		vm.PostQuery = "?from=calendar&month=" + month
		vm.TargetSelector = "#habit-calendar"
		vm.SwapMode = "innerHTML"
	case "today":
		vm.PostQuery = "?from=today"
		vm.TargetSelector = fmt.Sprintf("#habit-%d", habitID)
		vm.SwapMode = "outerHTML"
	default:
		vm.TargetSelector = fmt.Sprintf("#habit-%d", habitID)
		vm.SwapMode = "outerHTML"
	}
	return vm
}

// handleEntryEditForm renders the inline numeric-entry mini-form (a number
// input swapped in over the habit row's value cell). Boolean habits don't
// use this route — their checkmark posts directly.
func (s *Server) handleEntryEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	date, err := domain.ParseDate(r.PathValue("date"))
	if err != nil {
		http.Error(w, "invalid date", http.StatusBadRequest)
		return
	}

	h, err := s.habits.Get(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.serverError(w, err)
		return
	}
	if h.Type != domain.Numerical {
		http.Error(w, "not a numeric habit", http.StatusBadRequest)
		return
	}

	entry, _, err := s.entries.Get(r.Context(), id, date)
	if err != nil {
		s.serverError(w, err)
		return
	}
	vm := buildNumericEntryFormVM(id, date.String(), h.Unit, entry.NumericValue,
		r.URL.Query().Get("from"), r.URL.Query().Get("month"))
	s.renderPartial(w, "numeric_entry_form", vm)
}

// handleEntryToggle advances a boolean habit's entry through its click-cycle,
// or (for numeric habits) records the submitted value — then re-renders
// whichever view triggered it (a habit list row, a /today item, or a
// habit's calendar), per the "from" query param.
func (s *Server) handleEntryToggle(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	date, err := domain.ParseDate(r.PathValue("date"))
	if err != nil {
		http.Error(w, "invalid date", http.StatusBadRequest)
		return
	}

	h, err := s.habits.Get(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.serverError(w, err)
		return
	}

	if h.Type == domain.Numerical {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "could not parse form", http.StatusBadRequest)
			return
		}
		value, err := strconv.ParseFloat(r.PostForm.Get("value"), 64)
		if err != nil {
			http.Error(w, "invalid value", http.StatusBadRequest)
			return
		}
		if err := s.entries.Upsert(r.Context(), h.Type, domain.Entry{HabitID: id, Date: date, NumericValue: value}); err != nil {
			s.serverError(w, err)
			return
		}
	} else {
		current, _, err := s.entries.Get(r.Context(), id, date)
		if err != nil {
			s.serverError(w, err)
			return
		}
		next := domain.NextToggleValue(current.Value, s.cfg.SkipEnabled)
		if err := s.entries.Upsert(r.Context(), h.Type, domain.Entry{HabitID: id, Date: date, Value: next}); err != nil {
			s.serverError(w, err)
			return
		}
	}

	from := r.URL.Query().Get("from")

	// A calendar day-click re-renders that month's whole grid — the target
	// (#habit-calendar) isn't a per-habit-row element, so this can't reuse
	// buildHabitVM below.
	if from == "calendar" {
		month := parseMonthParam(r.URL.Query().Get("month"))
		calVM, err := s.buildCalendarVM(r.Context(), h, month)
		if err != nil {
			s.serverError(w, err)
			return
		}
		s.renderPartial(w, "habit_calendar", calVM)
		return
	}

	vm, err := s.buildHabitVM(r.Context(), h)
	if err != nil {
		s.serverError(w, err)
		return
	}

	// The /today checklist posts with ?from=today so a now-completed item
	// disappears (empty response removes it via hx-swap="outerHTML")
	// instead of re-rendering the full list row, which /today doesn't show.
	if from == "today" {
		if vm.TodayCompleted {
			return
		}
		s.renderPartial(w, "today_item", vm)
		return
	}
	s.renderPartial(w, "habit_row", vm)
}
