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
	IsEdit          bool
	Habit           domain.Habit
	Palette         []string
	Error           string
	ReminderEnabled bool
	Reminder        domain.Reminder
}

func (s *Server) handleHabitNewForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, "page_habit_form", habitFormVM{
		Palette:  domain.Palette[:],
		Reminder: domain.Reminder{Hour: 8, WeekdayMask: domain.AllWeekdaysMask},
	})
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

	rem, ok, err := s.reminders.Get(r.Context(), id)
	if err != nil {
		s.serverError(w, err)
		return
	}
	if !ok {
		rem = domain.Reminder{Hour: 8, WeekdayMask: domain.AllWeekdaysMask}
	}

	s.render(w, "page_habit_form", habitFormVM{IsEdit: true, Habit: h, Palette: domain.Palette[:], ReminderEnabled: ok, Reminder: rem})
}

func (s *Server) handleHabitCreate(w http.ResponseWriter, r *http.Request) {
	h, formErr := parseHabitForm(r)
	if formErr != "" {
		// This response replaces the <form> itself via hx-swap="outerHTML",
		// so it must be the bare form partial, not a full layout-wrapped page.
		s.renderPartial(w, "page_habit_form", habitFormVM{Habit: h, Palette: domain.Palette[:], Error: formErr})
		return
	}

	id, err := s.habits.Create(r.Context(), h)
	if err != nil {
		s.serverError(w, err)
		return
	}
	if err := s.saveReminderForm(r, id); err != nil {
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

	if _, err := s.habits.Get(r.Context(), id); errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		s.serverError(w, err)
		return
	}

	h, formErr := parseHabitForm(r)
	h.ID = id
	if formErr != "" {
		// This response replaces the <form> itself via hx-swap="outerHTML",
		// so it must be the bare form partial, not a full layout-wrapped page.
		s.renderPartial(w, "page_habit_form", habitFormVM{IsEdit: true, Habit: h, Palette: domain.Palette[:], Error: formErr})
		return
	}

	if err := s.habits.Update(r.Context(), h); err != nil {
		s.serverError(w, err)
		return
	}
	if err := s.saveReminderForm(r, id); err != nil {
		s.serverError(w, err)
		return
	}
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

// handleHabitReorder persists a new drag-and-drop order: the client posts
// the full list of habit ids as repeated "id" fields, in their new order.
func (s *Server) handleHabitReorder(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "could not parse form", http.StatusBadRequest)
		return
	}
	idStrs := r.PostForm["id"]
	ids := make([]int64, 0, len(idStrs))
	for _, s := range idStrs {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		ids = append(ids, id)
	}
	if err := s.habits.Reorder(r.Context(), ids); err != nil {
		s.serverError(w, err)
		return
	}
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

// parseHabitForm reads and validates the add/edit habit form, returning a
// fully-populated Habit (formErr == "" on success) covering every field the
// form exposes: name/question/description/color, type, and — depending on
// type — either the numeric target fields or the frequency ratio.
func parseHabitForm(r *http.Request) (domain.Habit, string) {
	if err := r.ParseForm(); err != nil {
		return domain.Habit{}, "could not parse form"
	}

	name := strings.TrimSpace(r.PostForm.Get("name"))

	habitType := domain.YesNo
	if r.PostForm.Get("type") == string(domain.Numerical) {
		habitType = domain.Numerical
	}

	color, err := strconv.Atoi(r.PostForm.Get("color"))
	if err != nil || color < 0 || color >= len(domain.Palette) {
		color = 0
	}

	targetType := domain.AtLeast
	if r.PostForm.Get("target_type") == string(domain.AtMost) {
		targetType = domain.AtMost
	}
	targetValue, _ := strconv.ParseFloat(r.PostForm.Get("target_value"), 64)

	numerator, errNum := strconv.Atoi(r.PostForm.Get("freq_numerator"))
	if errNum != nil || numerator < 1 {
		numerator = 1
	}
	denominator, errDen := strconv.Atoi(r.PostForm.Get("freq_denominator"))
	if errDen != nil || denominator < 1 {
		denominator = 1
	}

	h := domain.Habit{
		Name:        name,
		Question:    strings.TrimSpace(r.PostForm.Get("question")),
		Description: strings.TrimSpace(r.PostForm.Get("description")),
		Color:       color,
		Type:        habitType,
		Unit:        strings.TrimSpace(r.PostForm.Get("unit")),
		TargetValue: targetValue,
		TargetType:  targetType,
		Freq:        domain.NewFrequency(numerator, denominator),
	}

	if name == "" {
		return h, "Name is required."
	}
	if numerator > denominator {
		return h, "Frequency can't require more than once per day — times must be at most days."
	}

	return h, ""
}

// saveReminderForm reads the reminder fields from the same submitted form
// parseHabitForm already parsed (r.PostForm is populated by then) and
// persists or clears the habit's reminder accordingly.
func (s *Server) saveReminderForm(r *http.Request, habitID int64) error {
	if r.PostForm.Get("reminder_enabled") != "on" {
		return s.reminders.Delete(r.Context(), habitID)
	}

	hour, minute := parseHHMM(r.PostForm.Get("reminder_time"))
	mask := 0
	for _, v := range r.PostForm["reminder_weekday"] {
		if bit, err := strconv.Atoi(v); err == nil && bit >= 0 && bit < 7 {
			mask |= 1 << bit
		}
	}
	if mask == 0 {
		mask = domain.AllWeekdaysMask
	}

	return s.reminders.Set(r.Context(), domain.Reminder{HabitID: habitID, Hour: hour, Minute: minute, WeekdayMask: mask})
}

// parseHHMM parses an <input type="time"> value ("HH:MM"), defaulting to
// 08:00 if missing or malformed.
func parseHHMM(s string) (hour, minute int) {
	hour, minute = 8, 0
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return hour, minute
	}
	h, errH := strconv.Atoi(parts[0])
	m, errM := strconv.Atoi(parts[1])
	if errH != nil || errM != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 8, 0
	}
	return h, m
}
