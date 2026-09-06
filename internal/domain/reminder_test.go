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
