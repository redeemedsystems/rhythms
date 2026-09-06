package server

import (
	"net/http"
	"strconv"
	"strings"

	"rhythms/internal/auth"
	"rhythms/internal/store"
)

type HabitsPageData struct {
	Base
	Habits []*store.Habit
}

func (s *Server) handleHabitsList(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	habits, err := store.ListHabitsForUser(s.db, user.ID, true)
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.render.Page(w, "habits_list.html", HabitsPageData{
		Base:   s.baseFor(r, "habits"),
		Habits: habits,
	})
}

type HabitFormPageData struct {
	Base
	Habit *store.Habit // nil when creating
}

func (s *Server) handleHabitNewForm(w http.ResponseWriter, r *http.Request) {
	s.render.Page(w, "habit_form.html", HabitFormPageData{Base: s.baseFor(r, "habits")})
}

func (s *Server) handleHabitCreate(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())

	in, err := parseHabitForm(r)
	if err != nil {
		s.render.Page(w, "habit_form.html", HabitFormPageData{
			Base: Base{LoggedIn: true, Nav: "habits", Flash: err.Error(), CSRFToken: s.csrfFor(r)},
		})
		return
	}

	if _, err := store.CreateHabit(s.db, user.ID, in); err != nil {
		s.internalError(w, err)
		return
	}
	http.Redirect(w, r, "/habits", http.StatusSeeOther)
}

func (s *Server) handleHabitEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	habit, err := s.habitForUser(r, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	s.render.Page(w, "habit_form.html", HabitFormPageData{Base: s.baseFor(r, "habits"), Habit: habit})
}

func (s *Server) handleHabitUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	habit, err := s.habitForUser(r, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	in, err := parseHabitForm(r)
	if err != nil {
		s.render.Page(w, "habit_form.html", HabitFormPageData{
			Base:  Base{LoggedIn: true, Nav: "habits", Flash: err.Error(), CSRFToken: s.csrfFor(r)},
			Habit: habit,
		})
		return
	}

	if err := store.UpdateHabit(s.db, id, in); err != nil {
		s.internalError(w, err)
		return
	}
	http.Redirect(w, r, "/habits", http.StatusSeeOther)
}

func (s *Server) handleHabitArchive(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if _, err := s.habitForUser(r, id); err != nil {
		http.NotFound(w, r)
		return
	}
	if err := store.ArchiveHabit(s.db, id); err != nil {
		s.internalError(w, err)
		return
	}
	http.Redirect(w, r, "/habits", http.StatusSeeOther)
}

// habitForUser loads a habit and verifies it belongs to the request's user.
func (s *Server) habitForUser(r *http.Request, id int64) (*store.Habit, error) {
	habit, err := store.GetHabit(s.db, id)
	if err != nil {
		return nil, err
	}
	user := auth.UserFromContext(r.Context())
	if habit.UserID != user.ID {
		return nil, store.ErrNotFound
	}
	return habit, nil
}

func parseHabitForm(r *http.Request) (store.HabitInput, error) {
	if err := r.ParseForm(); err != nil {
		return store.HabitInput{}, errBadForm
	}

	name := strings.TrimSpace(r.FormValue("name"))
	habitType := r.FormValue("type")
	scheduleKind := r.FormValue("schedule_kind")
	unit := strings.TrimSpace(r.FormValue("unit"))

	if name == "" {
		return store.HabitInput{}, errValidation("Name is required.")
	}
	switch habitType {
	case store.HabitTypeBoolean, store.HabitTypeCount, store.HabitTypeQuantity:
	default:
		return store.HabitInput{}, errValidation("Invalid habit type.")
	}
	switch scheduleKind {
	case store.ScheduleDaily, store.ScheduleTimesPerDay, store.ScheduleSpecificTimes:
	default:
		return store.HabitInput{}, errValidation("Invalid schedule.")
	}

	var targetCount *int64
	if raw := strings.TrimSpace(r.FormValue("target_count")); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n <= 0 {
			return store.HabitInput{}, errValidation("Target count must be a positive number.")
		}
		targetCount = &n
	}

	var scheduleTimes []string
	if raw := strings.TrimSpace(r.FormValue("schedule_times")); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			t := strings.TrimSpace(part)
			if t == "" {
				continue
			}
			if !isValidHHMM(t) {
				return store.HabitInput{}, errValidation("Reminder times must be in HH:MM form, comma separated.")
			}
			scheduleTimes = append(scheduleTimes, t)
		}
	}
	if scheduleKind == store.ScheduleSpecificTimes && len(scheduleTimes) == 0 {
		return store.HabitInput{}, errValidation("Add at least one reminder time for a specific-times schedule.")
	}

	return store.HabitInput{
		Name:          name,
		Type:          habitType,
		TargetCount:   targetCount,
		Unit:          unit,
		ScheduleKind:  scheduleKind,
		ScheduleTimes: scheduleTimes,
	}, nil
}

func isValidHHMM(s string) bool {
	if len(s) != 5 || s[2] != ':' {
		return false
	}
	h, err1 := strconv.Atoi(s[0:2])
	m, err2 := strconv.Atoi(s[3:5])
	return err1 == nil && err2 == nil && h >= 0 && h <= 23 && m >= 0 && m <= 59
}

type validationError string

func (e validationError) Error() string { return string(e) }

func errValidation(msg string) error { return validationError(msg) }

var errBadForm = errValidation("Could not read form submission.")
