package web

import (
	"errors"
	"net/http"
	"strconv"

	"rhythms/internal/domain"
	"rhythms/internal/store"
)

type numericEntryFormVM struct {
	HabitID int64
	Date    string
	Unit    string
	Value   float64
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
	s.renderPartial(w, "numeric_entry_form", numericEntryFormVM{
		HabitID: id, Date: date.String(), Unit: h.Unit, Value: entry.NumericValue,
	})
}

// handleEntryToggle advances a boolean habit's entry through its click-cycle,
// or (for numeric habits) records the submitted value — either way,
// re-rendering the single affected row.
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

	vm, err := s.buildHabitVM(r.Context(), h)
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderPartial(w, "habit_row", vm)
}
