package server

import (
	"strconv"

	"rhythms/internal/store"
)

// HabitWithStatus is a habit annotated with today's logged amount and streak,
// as needed by the today-view and habit-card partial templates.
type HabitWithStatus struct {
	*store.Habit
	TodayAmount   int
	Complete      bool
	CurrentStreak int
}

func newHabitWithStatus(h *store.Habit, amount int) HabitWithStatus {
	return HabitWithStatus{
		Habit:       h,
		TodayAmount: amount,
		Complete:    store.IsComplete(h, amount),
	}
}

// ProgressLabel renders a short human string like "2/3 times" or "5/8 cups"
// for count/quantity habits, used by the habit_card partial.
func (h HabitWithStatus) ProgressLabel() string {
	switch h.Type {
	case store.HabitTypeCount, store.HabitTypeQuantity:
		unit := h.Unit.String
		if unit == "" {
			unit = "times"
		}
		if h.TargetCount.Valid {
			return strconv.Itoa(h.TodayAmount) + "/" + strconv.Itoa(int(h.TargetCount.Int64)) + " " + unit
		}
		return strconv.Itoa(h.TodayAmount) + " " + unit
	default:
		return ""
	}
}
