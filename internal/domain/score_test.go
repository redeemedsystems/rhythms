package domain

import (
	"math"
	"math/rand"
	"testing"
)

func TestScoreComputeHandCalculated(t *testing.T) {
	// Daily habit (freq=1.0): multiplier = 0.5^(sqrt(1)/13) = 0.5^(1/13).
	multiplier := math.Pow(0.5, 1.0/13.0)
	got := scoreCompute(1.0, 0.0, 1.0)
	want := 0.0*multiplier + 1.0*(1-multiplier)
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("scoreCompute(1.0, 0, 1) = %v, want %v", got, want)
	}

	// A second perfect day should push the score further toward 1.0 but
	// never reach it (pure EMA asymptote).
	got2 := scoreCompute(1.0, got, 1.0)
	if got2 <= got || got2 >= 1.0 {
		t.Errorf("scoreCompute should strictly increase toward but never reach 1.0, got %v then %v", got, got2)
	}
}

func TestScoreSeriesDailyPerfectHistoryApproachesOne(t *testing.T) {
	h := Habit{Type: YesNo, Freq: DailyFrequency()}
	from := NewDate(2026, 1, 1)
	to := from.AddDays(59)
	var original []Entry
	for i := 0; i < 60; i++ {
		original = append(original, Entry{Date: from.AddDays(i), Value: YesManual})
	}
	dense := DenseRange(original, from, to)
	series := ScoreSeries(h, dense)

	last := series[len(series)-1].Value
	if last < 0.95 {
		t.Errorf("score after 60 perfect days = %v, want close to 1.0 (>=0.95)", last)
	}
	// Score must be monotonically non-decreasing under an unbroken perfect
	// streak.
	for i := 1; i < len(series); i++ {
		if series[i].Value < series[i-1].Value-1e-12 {
			t.Errorf("score decreased at index %d: %v -> %v", i, series[i-1].Value, series[i].Value)
		}
	}
}

func TestScoreSeriesSkipCarriesForwardUnchanged(t *testing.T) {
	h := Habit{Type: YesNo, Freq: DailyFrequency()}
	original := []Entry{
		{Date: NewDate(2026, 1, 1), Value: YesManual},
		{Date: NewDate(2026, 1, 2), Value: Skip},
	}
	dense := DenseRange(original, NewDate(2026, 1, 1), NewDate(2026, 1, 2))
	series := ScoreSeries(h, dense)
	if len(series) != 2 {
		t.Fatalf("len(series) = %d, want 2", len(series))
	}
	if series[1].Value != series[0].Value {
		t.Errorf("score on a SKIP day = %v, want unchanged from previous day %v", series[1].Value, series[0].Value)
	}
}

func TestScoreSeriesNumericAtMostZeroTarget(t *testing.T) {
	h := Habit{Type: Numerical, Freq: DailyFrequency(), TargetType: AtMost, TargetValue: 0}
	original := []Entry{
		{Date: NewDate(2026, 1, 1), NumericValue: 0},
		{Date: NewDate(2026, 1, 2), NumericValue: 1},
	}
	dense := DenseRange(original, NewDate(2026, 1, 1), NewDate(2026, 1, 2))
	series := ScoreSeries(h, dense)

	// Day 1 (zero, on-budget) should score higher than day 2 (over budget).
	if series[1].Value >= series[0].Value {
		t.Errorf("scores = %v then %v, want a drop after exceeding a zero AT_MOST target", series[0].Value, series[1].Value)
	}
}

// Property: score is always within [0,1] and streak length is always within
// [0, len(history)], across randomly generated histories. Catches formula
// regressions that fixed-input unit tests might miss.
func TestScorePropertyStaysInUnitRange(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	values := []EntryValue{No, YesAuto, YesManual, Skip, Unknown}

	for trial := 0; trial < 200; trial++ {
		freq := NewFrequency(1+rng.Intn(5), 1+rng.Intn(14))
		h := Habit{Type: YesNo, Freq: freq}

		from := NewDate(2026, 1, 1)
		days := 5 + rng.Intn(90)
		to := from.AddDays(days - 1)

		var original []Entry
		for i := 0; i < days; i++ {
			original = append(original, Entry{Date: from.AddDays(i), Value: values[rng.Intn(len(values))]})
		}

		dense := DenseRange(original, from, to)
		series := ScoreSeries(h, dense)
		for _, sp := range series {
			if sp.Value < -1e-9 || sp.Value > 1+1e-9 {
				t.Fatalf("freq=%v: score %v out of [0,1] on %s", freq, sp.Value, sp.Date)
			}
		}

		streaks := Streaks(h, dense)
		for _, s := range streaks {
			if s.Length < 1 || s.Length > days {
				t.Fatalf("freq=%v: streak length %d out of [1,%d]", freq, s.Length, days)
			}
		}
	}
}
