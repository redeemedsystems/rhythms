package domain

import "testing"

func TestNextToggleValue(t *testing.T) {
	tests := []struct {
		name        string
		current     EntryValue
		skipEnabled bool
		want        EntryValue
	}{
		{"auto to manual, skip off", YesAuto, false, YesManual},
		{"auto to manual, skip on", YesAuto, true, YesManual},
		{"manual to no, skip off", YesManual, false, No},
		{"manual to skip, skip on", YesManual, true, Skip},
		{"no to manual, skip off", No, false, YesManual},
		{"no to unknown, skip on", No, true, Unknown},
		{"skip to no, skip off", Skip, false, No},
		{"skip to no, skip on", Skip, true, No},
		{"unknown to manual, skip off", Unknown, false, YesManual},
		{"unknown to manual, skip on", Unknown, true, YesManual},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NextToggleValue(tt.current, tt.skipEnabled)
			if got != tt.want {
				t.Errorf("NextToggleValue(%v, %v) = %v, want %v", tt.current, tt.skipEnabled, got, tt.want)
			}
		})
	}
}

func TestIsCompleted(t *testing.T) {
	yesNo := Habit{Type: YesNo}
	numericAtLeast := Habit{Type: Numerical, TargetType: AtLeast, TargetValue: 5}
	numericAtMost := Habit{Type: Numerical, TargetType: AtMost, TargetValue: 5}

	tests := []struct {
		name  string
		habit Habit
		entry Entry
		want  bool
	}{
		{"yes/no manual counts", yesNo, Entry{Value: YesManual}, true},
		{"yes/no auto counts", yesNo, Entry{Value: YesAuto}, true},
		{"yes/no no doesn't count", yesNo, Entry{Value: No}, false},
		// Skip is a neutral "rest day" — uHabits' own completion check is a
		// plain value>0, so Skip(3) satisfies it same as YesAuto/YesManual:
		// it doesn't break a streak, which this shared check implements by
		// treating it as completed rather than special-casing it separately.
		{"yes/no skip counts (neutral day, doesn't break streak)", yesNo, Entry{Value: Skip}, true},
		{"at-least above target", numericAtLeast, Entry{NumericValue: 6}, true},
		{"at-least exactly target", numericAtLeast, Entry{NumericValue: 5}, true},
		{"at-least below target", numericAtLeast, Entry{NumericValue: 4}, false},
		{"at-most below target", numericAtMost, Entry{NumericValue: 4}, true},
		{"at-most above target", numericAtMost, Entry{NumericValue: 6}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCompleted(tt.habit, tt.entry)
			if got != tt.want {
				t.Errorf("IsCompleted(...) = %v, want %v", got, tt.want)
			}
		})
	}
}
