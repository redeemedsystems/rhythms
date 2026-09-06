package server

import (
	"net/http"
	"strconv"
	"time"

	"rhythms/internal/auth"
	"rhythms/internal/store"
)

type TodayPageData struct {
	Base
	Habits  []HabitWithStatus
	LogDate string
}

func (s *Server) handleToday(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	habits, err := store.ListHabitsForUser(s.db, user.ID, true)
	if err != nil {
		s.internalError(w, err)
		return
	}

	now := store.NowIn(user.Timezone)
	logDate := now.Format("2006-01-02")

	items := make([]HabitWithStatus, 0, len(habits))
	for _, h := range habits {
		amount, err := store.AmountForDate(s.db, h.ID, logDate)
		if err != nil {
			s.internalError(w, err)
			return
		}
		hws := newHabitWithStatus(h, amount)
		if streak, err := s.currentStreakFor(h, now); err == nil {
			hws.CurrentStreak = streak
		}
		items = append(items, hws)
	}

	s.render.Page(w, "today.html", TodayPageData{
		Base:    s.baseFor(r, "today"),
		Habits:  items,
		LogDate: logDate,
	})
}

func (s *Server) handleHabitLog(w http.ResponseWriter, r *http.Request) {
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

	user := auth.UserFromContext(r.Context())
	logDate := store.LocalDate(user.Timezone, time.Now())

	if habit.Type == store.HabitTypeBoolean {
		amount, err := store.AmountForDate(s.db, id, logDate)
		if err != nil {
			s.internalError(w, err)
			return
		}
		if amount > 0 {
			err = store.DeleteTodayLogs(s.db, id, logDate)
		} else {
			err = store.CreateLog(s.db, id, user.ID, logDate, 1, "")
		}
		if err != nil {
			s.internalError(w, err)
			return
		}
	} else if err := store.CreateLog(s.db, id, user.ID, logDate, 1, ""); err != nil {
		s.internalError(w, err)
		return
	}

	amount, err := store.AmountForDate(s.db, id, logDate)
	if err != nil {
		s.internalError(w, err)
		return
	}

	hws := newHabitWithStatus(habit, amount)
	if streak, err := s.currentStreakFor(habit, store.NowIn(user.Timezone)); err == nil {
		hws.CurrentStreak = streak
	}

	s.render.Partial(w, "habit_card", hws)
}

// currentStreakFor pulls roughly a year of completion history for the habit
// and walks it backwards from now to find the current streak.
func (s *Server) currentStreakFor(habit *store.Habit, now time.Time) (int, error) {
	from := now.AddDate(-1, 0, -5)
	completion, err := store.CompletionByDate(s.db, habit.ID, from.Format("2006-01-02"), now.Format("2006-01-02"))
	if err != nil {
		return 0, err
	}
	return store.CurrentStreak(habit, completion, now), nil
}
