package domain

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

type Entry struct {
	HabitID      int64
	Date         Date
	Value        EntryValue
	NumericValue float64
	Notes        string
}

// NextToggleValue implements the checkmark button's click-cycle:
// YesAuto -> YesManual -> (Skip if enabled, else No) -> No -> (Unknown if
// enabled, else YesManual). skipEnabled is a per-deployment preference; M1
// hardcodes it to false (Skip/Unknown land in M2).
func NextToggleValue(current EntryValue, skipEnabled bool) EntryValue {
	switch current {
	case YesAuto:
		return YesManual
	case YesManual:
		if skipEnabled {
			return Skip
		}
		return No
	case Skip:
		return No
	case No:
		if skipEnabled {
			return Unknown
		}
		return YesManual
	default: // Unknown, or the zero value when no entry exists yet
		return YesManual
	}
}

// IsCompleted reports whether an entry counts toward a streak, per the
// habit's type and target.
func IsCompleted(h Habit, e Entry) bool {
	switch h.Type {
	case Numerical:
		if h.TargetType == AtMost {
			return e.NumericValue <= h.TargetValue
		}
		return e.NumericValue >= h.TargetValue
	default:
		return e.Value > No
	}
}
