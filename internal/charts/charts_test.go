package charts

import (
	"strings"
	"testing"

	"rhythms/internal/domain"
)

func TestScoreLineChartEmpty(t *testing.T) {
	got := ScoreLineChart(nil, "#000000")
	if !strings.Contains(string(got), "<svg") {
		t.Errorf("expected valid svg for empty input, got %s", got)
	}
}

func TestScoreLineChartRenders(t *testing.T) {
	points := []domain.ScorePoint{
		{Date: domain.NewDate(2026, 1, 1), Value: 0.2},
		{Date: domain.NewDate(2026, 1, 2), Value: 0.5},
		{Date: domain.NewDate(2026, 1, 3), Value: 0.9},
	}
	got := string(ScoreLineChart(points, "#388e3c"))
	if !strings.Contains(got, "<path") || !strings.Contains(got, "#388e3c") {
		t.Errorf("expected a path using the given color, got %s", got)
	}
}

func TestStreakBarChartEmpty(t *testing.T) {
	got := StreakBarChart(nil, "#000000")
	if !strings.Contains(string(got), "<svg") {
		t.Errorf("expected valid svg for empty input, got %s", got)
	}
}

func TestStreakBarChartCapsAtTenLongest(t *testing.T) {
	var streaks []domain.Streak
	for i := 0; i < 15; i++ {
		start := domain.NewDate(2026, 1, 1).AddDays(i * 10)
		streaks = append(streaks, domain.Streak{Start: start, End: start.AddDays(i), Length: i + 1})
	}
	got := string(StreakBarChart(streaks, "#388e3c"))
	if strings.Count(got, "<rect") != 10 {
		t.Errorf("expected exactly 10 bars, got %d rects in %s", strings.Count(got, "<rect"), got)
	}
}

func TestNumericValueLineChartEmpty(t *testing.T) {
	got := NumericValueLineChart(nil, "#000000")
	if !strings.Contains(string(got), "<svg") {
		t.Errorf("expected valid svg for empty input, got %s", got)
	}
}

func TestNumericValueLineChartSinglePoint(t *testing.T) {
	entries := []domain.Entry{{Date: domain.NewDate(2026, 1, 1), NumericValue: 5}}
	got := string(NumericValueLineChart(entries, "#388e3c"))
	if !strings.Contains(got, "Not enough data yet") {
		t.Errorf("expected the not-enough-data placeholder for a single point, got %s", got)
	}
}

func TestNumericValueLineChartFlatSeriesDoesNotDivideByZero(t *testing.T) {
	entries := []domain.Entry{
		{Date: domain.NewDate(2026, 1, 1), NumericValue: 3},
		{Date: domain.NewDate(2026, 1, 2), NumericValue: 3},
		{Date: domain.NewDate(2026, 1, 3), NumericValue: 3},
	}
	got := string(NumericValueLineChart(entries, "#388e3c"))
	if strings.Contains(got, "NaN") || strings.Contains(got, "Inf") {
		t.Errorf("expected a finite path for a flat series, got %s", got)
	}
	if !strings.Contains(got, "<path") {
		t.Errorf("expected a path for a flat series, got %s", got)
	}
}

func TestNumericValueLineChartScalesToMinMax(t *testing.T) {
	entries := []domain.Entry{
		{Date: domain.NewDate(2026, 1, 1), NumericValue: 10},
		{Date: domain.NewDate(2026, 1, 2), NumericValue: 20},
	}
	got := string(NumericValueLineChart(entries, "#388e3c"))
	if !strings.Contains(got, "#388e3c") {
		t.Errorf("expected a path using the given color, got %s", got)
	}
	if !strings.Contains(got, ">10<") || !strings.Contains(got, ">20<") {
		t.Errorf("expected min/max value labels 10 and 20, got %s", got)
	}
}

func TestCalendarHeatmapEmpty(t *testing.T) {
	h := domain.Habit{Type: domain.YesNo}
	got := CalendarHeatmap(h, nil, 26, "#388e3c")
	if !strings.Contains(string(got), "<svg") {
		t.Errorf("expected valid svg for empty input, got %s", got)
	}
}

func TestCalendarHeatmapMarksCompletedDays(t *testing.T) {
	h := domain.Habit{Type: domain.YesNo}
	today := domain.NewDate(2026, 9, 6)
	dense := domain.DenseRange([]domain.Entry{{Date: today, Value: domain.YesManual}}, today.AddDays(-6), today)
	got := string(CalendarHeatmap(h, dense, 2, "#388e3c"))
	if !strings.Contains(got, "#388e3c") {
		t.Errorf("expected the completed day to use the habit color, got %s", got)
	}
	if !strings.Contains(got, "<title>2026-09-06</title>") {
		t.Errorf("expected a title tooltip for the completed day, got %s", got)
	}
}
