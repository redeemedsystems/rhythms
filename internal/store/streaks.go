package store

import "time"

const dateLayout = "2006-01-02"

// IsComplete reports whether the given total amount logged for a day
// satisfies the habit: any log at all for a boolean habit, or reaching the
// configured target_count for count/quantity habits (defaulting to "any
// activity" if no target was set).
func IsComplete(h *Habit, amount int) bool {
	if amount <= 0 {
		return false
	}
	if h.Type == HabitTypeBoolean {
		return true
	}
	if h.TargetCount.Valid && h.TargetCount.Int64 > 0 {
		return int64(amount) >= h.TargetCount.Int64
	}
	return amount >= 1
}

// DateRange returns YYYY-MM-DD strings from `from` to `to`, inclusive, ascending.
func DateRange(from, to time.Time) []string {
	dates := make([]string, 0)
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format(dateLayout))
	}
	return dates
}

// CurrentStreak counts consecutive complete days ending at `today`. If today
// isn't complete yet (the day may still be in progress), it looks for a
// streak ending yesterday instead so an unlogged "today" doesn't itself
// break a streak in progress.
func CurrentStreak(h *Habit, completion map[string]int, today time.Time) int {
	d := today
	if !IsComplete(h, completion[d.Format(dateLayout)]) {
		d = d.AddDate(0, 0, -1)
	}

	streak := 0
	for IsComplete(h, completion[d.Format(dateLayout)]) {
		streak++
		d = d.AddDate(0, 0, -1)
	}
	return streak
}

// BestStreak returns the longest run of consecutive complete days within
// orderedDates (which must be ascending, contiguous calendar days).
func BestStreak(h *Habit, completion map[string]int, orderedDates []string) int {
	best, running := 0, 0
	for _, d := range orderedDates {
		if IsComplete(h, completion[d]) {
			running++
			if running > best {
				best = running
			}
		} else {
			running = 0
		}
	}
	return best
}
