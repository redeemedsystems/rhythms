package domain

// Streak is a contiguous run of dates that all satisfy IsCompleted for a
// habit. Ported from uHabits' StreakList.recompute.
type Streak struct {
	Start, End Date // Start is the oldest date in the run, End the newest
	Length     int
}

// Streaks groups a dense, day-by-day computed-entry timeline (see
// DenseRange) into maximal contiguous completed runs. dense must be sorted
// ascending (oldest first) and contain one entry per calendar day with no
// gaps — exactly what DenseRange produces.
func Streaks(h Habit, dense []Entry) []Streak {
	var completed []Date
	for _, e := range dense {
		if IsCompleted(h, e) {
			completed = append(completed, e.Date)
		}
	}
	if len(completed) == 0 {
		return nil
	}

	var streaks []Streak
	start := completed[0]
	end := completed[0]
	for _, d := range completed[1:] {
		if d.Equal(end.AddDays(1)) {
			end = d
			continue
		}
		streaks = append(streaks, Streak{Start: start, End: end, Length: start.DaysUntil(end) + 1})
		start, end = d, d
	}
	streaks = append(streaks, Streak{Start: start, End: end, Length: start.DaysUntil(end) + 1})
	return streaks
}

// CurrentStreak returns the length of the streak ending on `today`, or 0 if
// today isn't itself completed (i.e. there's no active streak right now).
func CurrentStreak(h Habit, dense []Entry, today Date) int {
	streaks := Streaks(h, dense)
	if len(streaks) == 0 {
		return 0
	}
	last := streaks[len(streaks)-1]
	if last.End.Equal(today) {
		return last.Length
	}
	return 0
}

// DenseRange fills gaps in a sparse computed-entry list with Unknown
// placeholders, producing exactly one entry per day in [from, to] ascending
// — the shape Streaks and ScoreSeries require, matching what uHabits'
// EntryList.getByInterval effectively returns (it defaults any day with no
// stored row to Unknown).
func DenseRange(computed []Entry, from, to Date) []Entry {
	byDate := make(map[string]Entry, len(computed))
	for _, e := range computed {
		byDate[e.Date.String()] = e
	}

	n := from.DaysUntil(to) + 1
	if n <= 0 {
		return nil
	}
	out := make([]Entry, n)
	for i := range out {
		d := from.AddDays(i)
		if e, ok := byDate[d.String()]; ok {
			out[i] = e
		} else {
			out[i] = Entry{Date: d, Value: Unknown}
		}
	}
	return out
}
