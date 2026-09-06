package domain

import "testing"

func TestComputeEntriesDailyPassesThroughManualEntries(t *testing.T) {
	h := Habit{Type: YesNo, Freq: DailyFrequency()}
	original := []Entry{
		{Date: NewDate(2026, 1, 1), Value: YesManual},
		{Date: NewDate(2026, 1, 2), Value: No},
		{Date: NewDate(2026, 1, 3), Value: YesManual},
	}
	computed := ComputeEntries(h, original)

	byDate := indexByDate(computed)
	if byDate["2026-01-01"] != YesManual {
		t.Errorf("2026-01-01 = %v, want YesManual", byDate["2026-01-01"])
	}
	if byDate["2026-01-02"] != No {
		t.Errorf("2026-01-02 = %v, want No", byDate["2026-01-02"])
	}
	if byDate["2026-01-03"] != YesManual {
		t.Errorf("2026-01-03 = %v, want YesManual", byDate["2026-01-03"])
	}
}

func TestComputeEntriesWeeklyAutoFillsRestOfWeek(t *testing.T) {
	// Weekly habit (1/7), checked manually only on 2026-01-04 and 2026-01-11
	// (a week apart). Every day in the 7-day window starting at each manual
	// check should be auto-filled, since the quota (1 check per 7 days) was
	// met by that single check.
	h := Habit{Type: YesNo, Freq: NewFrequency(1, 7)}
	original := []Entry{
		{Date: NewDate(2026, 1, 4), Value: YesManual},
		{Date: NewDate(2026, 1, 11), Value: YesManual},
	}
	computed := ComputeEntries(h, original)
	byDate := indexByDate(computed)

	for day := 4; day <= 10; day++ {
		date := NewDate(2026, 1, day).String()
		v, ok := byDate[date]
		if !ok {
			t.Errorf("%s: expected an entry, got none", date)
			continue
		}
		if day == 4 {
			if v != YesManual {
				t.Errorf("%s = %v, want YesManual (the actual check)", date, v)
			}
		} else if v != YesAuto {
			t.Errorf("%s = %v, want YesAuto (auto-filled by the week's quota)", date, v)
		}
	}
}

func TestComputeEntriesWeeklyLeavesGapBetweenNonAdjacentWeeks(t *testing.T) {
	// Checked on 2026-01-04, then not again until 2026-02-01 (four weeks
	// later) — the intervening days fall outside any 7-day window and must
	// stay unfilled (absent from the computed list, i.e. Unknown).
	h := Habit{Type: YesNo, Freq: NewFrequency(1, 7)}
	original := []Entry{
		{Date: NewDate(2026, 1, 4), Value: YesManual},
		{Date: NewDate(2026, 2, 1), Value: YesManual},
	}
	computed := ComputeEntries(h, original)
	byDate := indexByDate(computed)

	if _, ok := byDate["2026-01-20"]; ok {
		t.Errorf("2026-01-20 should have no entry (outside any quota window), got %v", byDate["2026-01-20"])
	}
}

func TestComputeEntriesSkipIsPreservedInsideAndOutsideIntervals(t *testing.T) {
	h := Habit{Type: YesNo, Freq: NewFrequency(1, 7)}
	original := []Entry{
		{Date: NewDate(2026, 1, 4), Value: YesManual},
		{Date: NewDate(2026, 1, 6), Value: Skip}, // inside the week-1 interval
	}
	computed := ComputeEntries(h, original)
	byDate := indexByDate(computed)

	if byDate["2026-01-06"] != Skip {
		t.Errorf("2026-01-06 = %v, want Skip (explicit SKIP must survive interval overlay)", byDate["2026-01-06"])
	}
}

func TestComputeEntriesExplicitNoInsideIntervalBecomesAuto(t *testing.T) {
	// A day explicitly marked No, but falling inside a window whose quota
	// was already met some other day, should be overridden to YesAuto —
	// the interval's coverage wins over an explicit "not done".
	h := Habit{Type: YesNo, Freq: NewFrequency(1, 7)}
	original := []Entry{
		{Date: NewDate(2026, 1, 4), Value: YesManual},
		{Date: NewDate(2026, 1, 6), Value: No},
	}
	computed := ComputeEntries(h, original)
	byDate := indexByDate(computed)

	if byDate["2026-01-06"] != YesAuto {
		t.Errorf("2026-01-06 = %v, want YesAuto (interval coverage overrides explicit No)", byDate["2026-01-06"])
	}
}

func TestComputeEntriesThreeTimesPerWeek(t *testing.T) {
	// 3 times per 7 days, checked on Mon/Wed/Fri of the same week — all
	// three checks fall within a 7-day span, so the whole week auto-fills.
	h := Habit{Type: YesNo, Freq: NewFrequency(3, 7)}
	original := []Entry{
		{Date: NewDate(2026, 1, 5), Value: YesManual}, // Mon
		{Date: NewDate(2026, 1, 7), Value: YesManual}, // Wed
		{Date: NewDate(2026, 1, 9), Value: YesManual}, // Fri
	}
	computed := ComputeEntries(h, original)
	byDate := indexByDate(computed)

	if byDate["2026-01-06"] != YesAuto { // Tue, between two manual checks
		t.Errorf("2026-01-06 = %v, want YesAuto", byDate["2026-01-06"])
	}
	if byDate["2026-01-11"] != YesAuto { // Sun, within the 7-day window from Mon
		t.Errorf("2026-01-11 = %v, want YesAuto", byDate["2026-01-11"])
	}
}

func TestComputeEntriesNumericIsPassthrough(t *testing.T) {
	h := Habit{Type: Numerical, Freq: NewFrequency(1, 1), TargetType: AtLeast, TargetValue: 5}
	original := []Entry{
		{Date: NewDate(2026, 1, 1), NumericValue: 3},
		{Date: NewDate(2026, 1, 3), NumericValue: 7},
	}
	computed := ComputeEntries(h, original)
	if len(computed) != 2 {
		t.Fatalf("len(computed) = %d, want 2 (numeric habits pass through unchanged)", len(computed))
	}
	if computed[0].NumericValue != 3 || computed[1].NumericValue != 7 {
		t.Errorf("computed = %+v, want unchanged copies of original", computed)
	}
}

func TestComputeEntriesEmptyHistory(t *testing.T) {
	h := Habit{Type: YesNo, Freq: DailyFrequency()}
	if got := ComputeEntries(h, nil); len(got) != 0 {
		t.Errorf("ComputeEntries(nil) = %v, want empty", got)
	}
}

func TestComputeEntriesZeroFrequencyDoesNotPanic(t *testing.T) {
	// Habit carries no constructor-enforced invariant, so a zero-value
	// Frequency (e.g. from a caller that forgot to default it) must not
	// reach buildIntervals' loop bound and index a slice at -1.
	h := Habit{Type: YesNo}
	original := []Entry{{Date: NewDate(2026, 1, 1), Value: YesManual}}
	if got := ComputeEntries(h, original); len(got) == 0 {
		t.Error("expected at least the manual entry to survive, got none")
	}
}

func TestNewFrequencyNormalizesEqualRatiosToOne(t *testing.T) {
	got := NewFrequency(5, 5)
	want := Frequency{Numerator: 1, Denominator: 1}
	if got != want {
		t.Errorf("NewFrequency(5,5) = %v, want %v", got, want)
	}
}

func indexByDate(entries []Entry) map[string]EntryValue {
	m := make(map[string]EntryValue, len(entries))
	for _, e := range entries {
		m[e.Date.String()] = e.Value
	}
	return m
}
