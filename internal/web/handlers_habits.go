package web

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"rhythms/internal/domain"
	"rhythms/internal/store"
)

type habitListVM struct {
	ShowArchived bool
	Habits       []habitVM
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	showArchived := r.URL.Query().Get("archived") == "1"

	vm, err := s.buildHabitListVM(r, showArchived)
	if err != nil {
		s.serverError(w, err)
		return
	}

	s.render(w, "page_index", vm)
}

// handleHabitListPartial re-renders just the list (used after archive/delete
// so the whole row set can change without a full page reload).
func (s *Server) handleHabitListPartial(w http.ResponseWriter, r *http.Request, showArchived bool) {
	vm, err := s.buildHabitListVM(r, showArchived)
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderPartial(w, "habit_list", vm)
}

func (s *Server) buildHabitListVM(r *http.Request, showArchived bool) (habitListVM, error) {
	habits, err := s.habits.List(r.Context(), showArchived)
	if err != nil {
		return habitListVM{}, err
	}

	filterQuery := ""
	if showArchived {
		filterQuery = "?archived=1"
	}

	vms := make([]habitVM, 0, len(habits))
	for _, h := range habits {
		vm, err := s.buildHabitVM(r.Context(), h)
		if err != nil {
			return habitListVM{}, err
		}
		vm.ListFilterQuery = filterQuery
		vms = append(vms, vm)
	}

	return habitListVM{ShowArchived: showArchived, Habits: vms}, nil
}

type habitFormVM struct {
	IsEdit  bool
	Habit   domain.Habit
	Palette []string
	Error   string
}

func (s *Server) handleHabitNewForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, "page_habit_form", habitFormVM{Palette: domain.Palette[:]})
}

func (s *Server) handleHabitEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	s.render(w, "page_habit_form", habitFormVM{IsEdit: true, Habit: h, Palette: domain.Palette[:]})
}

func (s *Server) handleHabitCreate(w http.ResponseWriter, r *http.Request) {
	h, formErr := parseHabitForm(r)
	if formErr != "" {
		// This response replaces the <form> itself via hx-swap="outerHTML",
		// so it must be the bare form partial, not a full layout-wrapped page.
		s.renderPartial(w, "page_habit_form", habitFormVM{Habit: h, Palette: domain.Palette[:], Error: formErr})
		return
	}

	if _, err := s.habits.Create(r.Context(), h); err != nil {
		s.serverError(w, err)
		return
	}
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleHabitUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Load the existing habit first: the form only exposes
	// name/question/description/color, so type/target/frequency must carry
	// over unchanged rather than being zeroed out (Update writes every
	// column, and an empty habit_type/target_type would violate the
	// table's CHECK constraint).
	existing, err := s.habits.Get(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.serverError(w, err)
		return
	}

	formHabit, formErr := parseHabitForm(r)
	if formErr != "" {
		existing.Name, existing.Question, existing.Description = formHabit.Name, formHabit.Question, formHabit.Description
		s.renderPartial(w, "page_habit_form", habitFormVM{IsEdit: true, Habit: existing, Palette: domain.Palette[:], Error: formErr})
		return
	}

	existing.Name = formHabit.Name
	existing.Question = formHabit.Question
	existing.Description = formHabit.Description
	existing.Color = formHabit.Color

	if err := s.habits.Update(r.Context(), existing); err != nil {
		s.serverError(w, err)
		return
	}
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleHabitArchiveToggle(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	if err := s.habits.SetArchived(r.Context(), id, !h.Archived); err != nil {
		s.serverError(w, err)
		return
	}

	// The row just left (or entered) the current view's filter, so
	// re-render the whole list rather than just this row.
	s.handleHabitListPartial(w, r, r.URL.Query().Get("archived") == "1")
}

func (s *Server) handleHabitDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.habits.Delete(r.Context(), id); err != nil && !errors.Is(err, store.ErrNotFound) {
		s.serverError(w, err)
		return
	}
	s.handleHabitListPartial(w, r, r.URL.Query().Get("archived") == "1")
}

func parseIDParam(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

// parseHabitForm reads and validates the add/edit habit form. On success it
// returns a Habit with formErr == "". Type/target fields are left at their
// zero values here — HabitRepo.Create fills in the YES_NO/daily defaults;
// Update preserves whatever the stored habit already had by only touching
// the fields this form actually exposes (name/question/description/color).
func parseHabitForm(r *http.Request) (domain.Habit, string) {
	if err := r.ParseForm(); err != nil {
		return domain.Habit{}, "could not parse form"
	}

	name := strings.TrimSpace(r.PostForm.Get("name"))
	if name == "" {
		return domain.Habit{Question: r.PostForm.Get("question"), Description: r.PostForm.Get("description")},
			"Name is required."
	}

	color, err := strconv.Atoi(r.PostForm.Get("color"))
	if err != nil || color < 0 || color >= len(domain.Palette) {
		color = 0
	}

	return domain.Habit{
		Name:        name,
		Question:    strings.TrimSpace(r.PostForm.Get("question")),
		Description: strings.TrimSpace(r.PostForm.Get("description")),
		Color:       color,
	}, ""
}
