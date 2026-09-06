package domain

import (
	"testing"
	"time"
)

func TestWeekdayMaskFromAndActiveOn(t *testing.T) {
	mask := WeekdayMaskFrom(time.Monday, time.Wednesday, time.Friday)
	r := Reminder{WeekdayMask: mask}

	active := map[time.Weekday]bool{
		time.Monday: true, time.Tuesday: false, time.Wednesday: true,
		time.Thursday: false, time.Friday: true, time.Saturday: false, time.Sunday: false,
	}
	for day, want := range active {
		if got := r.ActiveOn(day); got != want {
			t.Errorf("ActiveOn(%v) = %v, want %v", day, got, want)
		}
	}
}

func TestAllWeekdaysMaskIsEveryDay(t *testing.T) {
	r := Reminder{WeekdayMask: AllWeekdaysMask}
	for d := time.Sunday; d <= time.Saturday; d++ {
		if !r.ActiveOn(d) {
			t.Errorf("AllWeekdaysMask should be active on %v", d)
		}
	}
}

func TestReminderIsDue(t *testing.T) {
	// Monday 2026-09-07, 08:00 reminder, active every weekday.
	r := Reminder{Hour: 8, Minute: 0, WeekdayMask: AllWeekdaysMask}
	monday830 := time.Date(2026, 9, 7, 8, 30, 0, 0, time.UTC)
	monday759 := time.Date(2026, 9, 7, 7, 59, 0, 0, time.UTC)
	monday800 := time.Date(2026, 9, 7, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name                                    string
		now                                     time.Time
		alreadySentToday, alreadyCompletedToday bool
		want                                    bool
	}{
		{"before trigger time", monday759, false, false, false},
		{"exactly at trigger time", monday800, false, false, true},
		{"after trigger time", monday830, false, false, true},
		{"already sent today", monday830, true, false, false},
		{"already completed today", monday830, false, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.IsDue(tt.now, tt.alreadySentToday, tt.alreadyCompletedToday); got != tt.want {
				t.Errorf("IsDue(%v, sent=%v, done=%v) = %v, want %v", tt.now, tt.alreadySentToday, tt.alreadyCompletedToday, got, tt.want)
			}
		})
	}
}

func TestReminderIsDueRespectsWeekdayMask(t *testing.T) {
	// Reminder only active on weekends; Monday 2026-09-07 at 09:00 should not fire.
	r := Reminder{Hour: 8, Minute: 0, WeekdayMask: WeekdayMaskFrom(time.Saturday, time.Sunday)}
	monday := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)
	if r.IsDue(monday, false, false) {
		t.Error("expected reminder not active on Monday to not be due")
	}
}

func TestFormattedValue(t *testing.T) {
	tests := []struct {
		v    EntryValue
		want string
	}{
		{YesManual, "YES_MANUAL"},
		{YesAuto, "YES_AUTO"},
		{No, "NO"},
		{Skip, "SKIP"},
		{Unknown, "UNKNOWN"},
		{5000, "5000"}, // a numeric habit's raw fixed-point value
	}
	for _, tt := range tests {
		if got := FormattedValue(tt.v); got != tt.want {
			t.Errorf("FormattedValue(%d) = %q, want %q", tt.v, got, tt.want)
		}
	}
}
