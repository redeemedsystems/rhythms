package store

import "time"

// LocalDate returns t formatted as YYYY-MM-DD in the given IANA timezone,
// falling back to UTC if tz is invalid.
func LocalDate(tz string, t time.Time) string {
	return t.In(loc(tz)).Format(dateLayout)
}

// LocalClock returns t's date and "HH:MM" in the given IANA timezone.
func LocalClock(tz string, t time.Time) (date, hhmm string) {
	lt := t.In(loc(tz))
	return lt.Format(dateLayout), lt.Format("15:04")
}

// NowIn returns the current instant expressed in the given IANA timezone.
func NowIn(tz string) time.Time {
	return time.Now().In(loc(tz))
}

func loc(tz string) *time.Location {
	l, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return l
}
