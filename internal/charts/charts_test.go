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

func TestWeekdayBarChartAllZero(t *testing.T) {
	var counts [7]int
	got := string(WeekdayBarChart(counts, "#388e3c"))
	if !strings.Contains(got, "<svg") {
		t.Errorf("expected valid svg for all-zero input, got %s", got)
	}
	if strings.Contains(got, "<rect") {
		t.Errorf("expected no bars when all counts are zero, got %s", got)
	}
}

func TestWeekdayBarChartRendersNonZero(t *testing.T) {
	counts := [7]int{1, 0, 3, 0, 5, 0, 2}
	got := string(WeekdayBarChart(counts, "#388e3c"))
	if strings.Count(got, "<rect") != 4 {
		t.Errorf("expected 4 bars (one per nonzero day), got %d in %s", strings.Count(got, "<rect"), got)
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
