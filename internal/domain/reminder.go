package domain

import "time"

// Reminder is a per-habit daily alert time, gated to specific weekdays.
// Configuring one doesn't yet send anything — that's M5 (Web Push); for now
// it's just data a future notification path will read.
type Reminder struct {
	HabitID     int64
	Hour        int // 0-23
	Minute      int // 0-59
	WeekdayMask int // bit0=Mon .. bit6=Sun; AllWeekdaysMask = every day
}

const AllWeekdaysMask = 0b1111111

// weekdayBit maps a time.Weekday (Sunday=0..Saturday=6) to this package's
// bit index (Monday=0..Sunday=6) — Monday-first matches how the habit form
// lists days, and keeps the stored mask independent of time.Weekday's
// Sunday-first numbering.
func weekdayBit(w time.Weekday) int {
	return (int(w) + 6) % 7
}

// ActiveOn reports whether the reminder fires on the given weekday.
func (r Reminder) ActiveOn(w time.Weekday) bool {
	return r.WeekdayMask&(1<<weekdayBit(w)) != 0
}

// WeekdayMaskFrom builds a mask from the set of active weekdays.
func WeekdayMaskFrom(weekdays ...time.Weekday) int {
	mask := 0
	for _, w := range weekdays {
		mask |= 1 << weekdayBit(w)
	}
	return mask
}
