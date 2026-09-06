package domain

import "testing"

func TestStreaksSingleDay(t *testing.T) {
	h := Habit{Type: YesNo}
	dense := DenseRange([]Entry{{Date: NewDate(2026, 1, 5), Value: YesManual}}, NewDate(2026, 1, 1), NewDate(2026, 1, 10))
	streaks := Streaks(h, dense)
	if len(streaks) != 1 {
		t.Fatalf("len(streaks) = %d, want 1", len(streaks))
	}
	if streaks[0].Length != 1 || !streaks[0].Start.Equal(NewDate(2026, 1, 5)) {
		t.Errorf("streak = %+v, want a single day on 2026-01-05", streaks[0])
	}
}

func TestStreaksMonthBoundary(t *testing.T) {
	h := Habit{Type: YesNo}
	original := []Entry{
		{Date: NewDate(2026, 1, 30), Value: YesManual},
		{Date: NewDate(2026, 1, 31), Value: YesManual},
		{Date: NewDate(2026, 2, 1), Value: YesManual},
		{Date: NewDate(2026, 2, 2), Value: YesManual},
	}
	dense := DenseRange(original, NewDate(2026, 1, 28), NewDate(2026, 2, 4))
	streaks := Streaks(h, dense)
	if len(streaks) != 1 {
		t.Fatalf("len(streaks) = %d, want 1 (streak should span the month boundary)", len(streaks))
	}
	if streaks[0].Length != 4 {
		t.Errorf("streak length = %d, want 4", streaks[0].Length)
	}
}

func TestStreaksSkipDoesNotBreakStreak(t *testing.T) {
	h := Habit{Type: YesNo}
	original := []Entry{
		{Date: NewDate(2026, 1, 1), Value: YesManual},
		{Date: NewDate(2026, 1, 2), Value: Skip},
		{Date: NewDate(2026, 1, 3), Value: YesManual},
	}
	dense := DenseRange(original, NewDate(2026, 1, 1), NewDate(2026, 1, 3))
	streaks := Streaks(h, dense)
	if len(streaks) != 1 || streaks[0].Length != 3 {
		t.Errorf("streaks = %+v, want one 3-day streak (SKIP counts as neutral, not a break)", streaks)
	}
}

func TestStreaksRealMissBreaksStreak(t *testing.T) {
	h := Habit{Type: YesNo}
	original := []Entry{
		{Date: NewDate(2026, 1, 1), Value: YesManual},
		{Date: NewDate(2026, 1, 2), Value: No},
		{Date: NewDate(2026, 1, 3), Value: YesManual},
	}
	dense := DenseRange(original, NewDate(2026, 1, 1), NewDate(2026, 1, 3))
	streaks := Streaks(h, dense)
	if len(streaks) != 2 {
		t.Fatalf("len(streaks) = %d, want 2 (a real No breaks the streak)", len(streaks))
	}
	if streaks[0].Length != 1 || streaks[1].Length != 1 {
		t.Errorf("streaks = %+v, want two single-day streaks", streaks)
	}
}

func TestStreaksEmptyHistory(t *testing.T) {
	h := Habit{Type: YesNo}
	dense := DenseRange(nil, NewDate(2026, 1, 1), NewDate(2026, 1, 5))
	if got := Streaks(h, dense); got != nil {
		t.Errorf("Streaks with no completions = %v, want nil", got)
	}
}

func TestStreaksAllUnknownHistory(t *testing.T) {
	h := Habit{Type: YesNo}
	dense := DenseRange(nil, NewDate(2026, 1, 1), NewDate(2026, 1, 5))
	for _, e := range dense {
		if e.Value != Unknown {
			t.Fatalf("expected all-Unknown dense range, got %+v", e)
		}
	}
	if got := Streaks(h, dense); len(got) != 0 {
		t.Errorf("Streaks over all-Unknown history = %+v, want none", got)
	}
}

func TestCurrentStreakEndingToday(t *testing.T) {
	h := Habit{Type: YesNo}
	today := NewDate(2026, 1, 10)
	original := []Entry{
		{Date: NewDate(2026, 1, 8), Value: YesManual},
		{Date: NewDate(2026, 1, 9), Value: YesManual},
		{Date: NewDate(2026, 1, 10), Value: YesManual},
	}
	dense := DenseRange(original, NewDate(2026, 1, 1), today)
	if got := CurrentStreak(h, dense, today); got != 3 {
		t.Errorf("CurrentStreak = %d, want 3", got)
	}
}

func TestCurrentStreakZeroWhenTodayNotCompleted(t *testing.T) {
	h := Habit{Type: YesNo}
	today := NewDate(2026, 1, 10)
	original := []Entry{
		{Date: NewDate(2026, 1, 8), Value: YesManual},
		{Date: NewDate(2026, 1, 9), Value: YesManual},
	}
	dense := DenseRange(original, NewDate(2026, 1, 1), today)
	if got := CurrentStreak(h, dense, today); got != 0 {
		t.Errorf("CurrentStreak = %d, want 0 (today itself isn't completed)", got)
	}
}

func TestStreaksNumericAtLeast(t *testing.T) {
	h := Habit{Type: Numerical, TargetType: AtLeast, TargetValue: 5}
	original := []Entry{
		{Date: NewDate(2026, 1, 1), NumericValue: 5},
		{Date: NewDate(2026, 1, 2), NumericValue: 4},
		{Date: NewDate(2026, 1, 3), NumericValue: 10},
	}
	dense := DenseRange(original, NewDate(2026, 1, 1), NewDate(2026, 1, 3))
	streaks := Streaks(h, dense)
	if len(streaks) != 2 {
		t.Fatalf("streaks = %+v, want 2 (day 2 misses target)", streaks)
	}
}
