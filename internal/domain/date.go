package domain

import "time"

const dateLayout = "2006-01-02"

// Date is a calendar day with no time-of-day or timezone component — habit
// entries are tracked per-day, not per-instant.
type Date struct {
	t time.Time
}

func NewDate(y int, m time.Month, d int) Date {
	return Date{time.Date(y, m, d, 0, 0, 0, 0, time.UTC)}
}

// Today returns the current local calendar date.
func Today() Date {
	now := time.Now()
	return NewDate(now.Year(), now.Month(), now.Day())
}

func ParseDate(s string) (Date, error) {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return Date{}, err
	}
	return Date{t}, nil
}

func (d Date) String() string        { return d.t.Format(dateLayout) }
func (d Date) AddDays(n int) Date    { return Date{d.t.AddDate(0, 0, n)} }
func (d Date) Before(o Date) bool    { return d.t.Before(o.t) }
func (d Date) After(o Date) bool     { return d.t.After(o.t) }
func (d Date) Equal(o Date) bool     { return d.t.Equal(o.t) }
func (d Date) Weekday() time.Weekday { return d.t.Weekday() }

// DaysUntil returns the number of days from d to other: positive if other is
// later, negative if other is earlier, zero if equal. Mirrors uHabits'
// Timestamp.daysUntil, which the ported interval/streak/score algorithms
// depend on for their exact sign conventions.
func (d Date) DaysUntil(other Date) int {
	return int(other.t.Sub(d.t).Hours() / 24)
}

// DaysInMonth returns the number of days in d's calendar month.
func (d Date) DaysInMonth() int {
	firstOfMonth := time.Date(d.t.Year(), d.t.Month(), 1, 0, 0, 0, 0, time.UTC)
	firstOfNextMonth := firstOfMonth.AddDate(0, 1, 0)
	return int(firstOfNextMonth.Sub(firstOfMonth).Hours() / 24)
}

// IsLastDayOfMonth reports whether d is the final calendar day of its month.
func (d Date) IsLastDayOfMonth() bool {
	return d.AddDays(1).t.Month() != d.t.Month()
}

// IsZero reports whether d is the zero Date (never explicitly set).
func (d Date) IsZero() bool { return d.t.IsZero() }
