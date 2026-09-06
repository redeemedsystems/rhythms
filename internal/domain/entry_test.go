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
		{"no to manual, skip off", No, false, YesManual},
		{"no to manual, skip on", No, true, YesManual},
		{"manual to no, skip off", YesManual, false, No},
		{"manual to skip, skip on", YesManual, true, Skip},
		{"skip to no, skip off", Skip, false, No},
		{"skip to unknown, skip on", Skip, true, Unknown},
		{"unknown to no, skip off", Unknown, false, No},
		{"unknown to no, skip on", Unknown, true, No},
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

func TestNextToggleValueFullCycleWithSkipEnabled(t *testing.T) {
	// Regression test: the first tap on an untouched day must land on
	// YesManual (mark done), not Unknown — that's the overwhelmingly
	// common interaction, and getting the cycle order backwards here once
	// made every fresh habit's first click do the wrong thing.
	got := No
	want := []EntryValue{YesManual, Skip, Unknown, No}
	for i, w := range want {
		got = NextToggleValue(got, true)
		if got != w {
			t.Fatalf("step %d: got %v, want %v", i, got, w)
		}
	}
}

func TestNextToggleValueTwoStateCycleWithSkipDisabled(t *testing.T) {
	got := No
	for i := 0; i < 4; i++ {
		got = NextToggleValue(got, false)
		want := YesManual
		if i%2 == 1 {
			want = No
		}
		if got != want {
			t.Fatalf("step %d: got %v, want %v", i, got, want)
		}
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
		{"at-most with zero target, no data recorded", Habit{Type: Numerical, TargetType: AtMost, TargetValue: 0}, Entry{Value: Unknown, NumericValue: 0}, false},
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
