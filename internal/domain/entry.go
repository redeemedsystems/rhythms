package domain

import "strconv"

// EntryValue is the state of a boolean habit's entry for one day. Numeric
// habits ignore this field and use NumericValue instead.
type EntryValue int

const (
	Unknown   EntryValue = -1
	No        EntryValue = 0
	YesAuto   EntryValue = 1
	YesManual EntryValue = 2
	Skip      EntryValue = 3
)

// Entry is one habit's record for one calendar day. Value is meaningful for
// boolean habits, NumericValue for numeric ones — which field to read
// depends on the owning Habit's Type (see IsCompleted).
type Entry struct {
	HabitID      int64
	Date         Date
	Value        EntryValue
	NumericValue float64
	Notes        string
}

// NextToggleValue implements the checkmark button's click-cycle:
// No -> YesManual -> (Skip -> Unknown ->) No. The first tap on an untouched
// day must land on YesManual — that's the overwhelmingly common case (mark
// today done) — with Skip/Unknown reachable on further taps only when
// skipEnabled is on; otherwise it's a plain two-state No/YesManual toggle.
func NextToggleValue(current EntryValue, skipEnabled bool) EntryValue {
	switch current {
	case No:
		return YesManual
	case YesManual:
		if skipEnabled {
			return Skip
		}
		return No
	case Skip:
		if skipEnabled {
			return Unknown
		}
		return No
	case Unknown:
		return No
	default: // YesAuto: tapping an auto-filled day confirms it explicitly
		return YesManual
	}
}

// IsCompleted reports whether an entry counts toward a streak, per the
// habit's type and target. Ported from uHabits' StreakList.recompute filter.
func IsCompleted(h Habit, e Entry) bool {
	switch h.Type {
	case Numerical:
		if h.TargetType == AtMost {
			// A day with no recorded value at all (Unknown) must not count
			// as "at or under budget" — without this guard, an AT_MOST
			// habit with target 0 would treat every untouched day as a
			// perfect success, per uHabits' own explicit Unknown exclusion.
			return e.Value != Unknown && e.NumericValue <= h.TargetValue
		}
		return e.NumericValue >= h.TargetValue
	default:
		return e.Value > No
	}
}

// FormattedValue is the raw stored value's CSV representation, ported
// verbatim from uHabits' Entry.formattedValue: the boolean sentinels get
// their name, anything else (a numeric habit's fixed-point value) is just
// its integer string. Numeric habits share this column's raw encoding with
// boolean ones (see the store package's entry encoding), so the same
// formatting function applies regardless of habit type.
func FormattedValue(v EntryValue) string {
	switch v {
	case YesManual:
		return "YES_MANUAL"
	case YesAuto:
		return "YES_AUTO"
	case No:
		return "NO"
	case Skip:
		return "SKIP"
	case Unknown:
		return "UNKNOWN"
	default:
		return strconv.Itoa(int(v))
	}
}
