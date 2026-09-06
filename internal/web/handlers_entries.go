package web

import (
	"errors"
	"net/http"

	"rhythms/internal/domain"
	"rhythms/internal/store"
)

// handleEntryToggle advances a boolean habit's entry for one day through its
// click-cycle and re-renders that single row. skipEnabled is hardcoded off
// for M1 — SKIP/UNKNOWN states land in M2 behind a real preference.
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

	current, _, err := s.entries.Get(r.Context(), id, date)
	if err != nil {
		s.serverError(w, err)
		return
	}

	next := domain.NextToggleValue(current.Value, false /* skipEnabled: M2 */)
	if err := s.entries.Upsert(r.Context(), domain.Entry{HabitID: id, Date: date, Value: next}); err != nil {
		s.serverError(w, err)
		return
	}

	vm, err := s.buildHabitVM(r.Context(), h)
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderPartial(w, "habit_row", vm)
}
