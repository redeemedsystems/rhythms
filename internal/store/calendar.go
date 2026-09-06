package store

import "time"

// MonthGrid returns the calendar weeks (each exactly 7 days, Sunday-Saturday)
// covering the given month, padded with the trailing days of the prior month
// and the leading days of the next month so every week is complete.
func MonthGrid(year int, month time.Month) [][]time.Time {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	start := first.AddDate(0, 0, -int(first.Weekday()))

	last := first.AddDate(0, 1, -1)
	end := last.AddDate(0, 0, 6-int(last.Weekday()))

	var weeks [][]time.Time
	var week []time.Time
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		week = append(week, d)
		if len(week) == 7 {
			weeks = append(weeks, week)
			week = nil
		}
	}
	return weeks
}
