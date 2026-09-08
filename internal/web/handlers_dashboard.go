package web

import (
	"context"
	"html/template"
	"net/http"

	"rhythms/internal/charts"
	"rhythms/internal/domain"
)

// dashboardTrendDays matches valueChartDays (handlers_detail.go): a 90-day
// window is the app's established default for "recent history" charts.
const dashboardTrendDays = 90

// dashboardAccentColor colors the aggregate trend chart, which isn't tied
// to any one habit's color — it's the app's own theme color (see
// manifest.webmanifest's theme_color).
const dashboardAccentColor = "#00897B"

type dashboardHabitVM struct {
	ID             int64
	Name           string
	Color          string
	CurrentStreak  int
	ScorePercent   int
	TodayCompleted bool
}

type dashboardVM struct {
	HasHabits           bool
	TotalHabits         int
	DoneToday           int
	TodayPercent        int
	AverageScorePercent int
	Habits              []dashboardHabitVM
	TrendChart          template.HTML
}

// handleDashboard renders a cross-habit rollup: today's completion
// snapshot, average score over the last dashboardTrendDays days, and a
// compact per-habit list — complementing the per-habit detail page rather
// than duplicating it.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	habits, err := s.habits.List(ctx, mustUser(r).ID, false)
	if err != nil {
		s.serverError(w, err)
		return
	}

	vm := dashboardVM{HasHabits: len(habits) > 0, TotalHabits: len(habits)}
	if !vm.HasHabits {
		vm.TrendChart = charts.ScoreLineChart(nil, dashboardAccentColor)
		s.render(w, r, "page_dashboard", vm)
		return
	}

	today := domain.Today()
	windowFrom := today.AddDays(-(dashboardTrendDays - 1))

	sums := make(map[domain.Date]float64)
	counts := make(map[domain.Date]int)
	rawScoreSum := 0.0

	for _, h := range habits {
		row, series, err := s.dashboardHabitRow(ctx, h, today, windowFrom)
		if err != nil {
			s.serverError(w, err)
			return
		}
		vm.Habits = append(vm.Habits, row)
		if row.TodayCompleted {
			vm.DoneToday++
		}
		rawScoreSum += float64(row.ScorePercent) / 100

		for _, p := range series {
			if p.Date.Before(windowFrom) {
				continue
			}
			sums[p.Date] += p.Value
			counts[p.Date]++
		}
	}

	var trendPoints []domain.ScorePoint
	for d := windowFrom; !d.After(today); d = d.AddDays(1) {
		if c := counts[d]; c > 0 {
			trendPoints = append(trendPoints, domain.ScorePoint{Date: d, Value: sums[d] / float64(c)})
		}
	}
	vm.TrendChart = charts.ScoreLineChart(trendPoints, dashboardAccentColor)

	vm.TodayPercent = int(float64(vm.DoneToday)/float64(vm.TotalHabits)*100 + 0.5)
	vm.AverageScorePercent = int(rawScoreSum/float64(vm.TotalHabits)*100 + 0.5)

	s.render(w, r, "page_dashboard", vm)
}

// dashboardHabitRow computes one habit's snapshot row and its score series
// over the trend window, sharing a single dense pass over its history
// (same fetch/compute pattern as buildHabitVM in viewmodel.go).
func (s *Server) dashboardHabitRow(ctx context.Context, h domain.Habit, today, windowFrom domain.Date) (dashboardHabitVM, []domain.ScorePoint, error) {
	original, err := s.entries.ListAll(ctx, h.ID)
	if err != nil {
		return dashboardHabitVM{}, nil, err
	}
	computed := domain.ComputeEntries(h, original)

	from := windowFrom
	for _, e := range computed {
		if e.Date.Before(from) {
			from = e.Date
		}
	}
	dense := domain.DenseRange(computed, from, today)

	series := domain.ScoreSeries(h, dense)
	scorePercent := 0
	if len(series) > 0 {
		scorePercent = int(series[len(series)-1].Value*100 + 0.5)
	}

	row := dashboardHabitVM{
		ID:             h.ID,
		Name:           h.Name,
		Color:          colorHex(h.Color),
		CurrentStreak:  domain.CurrentStreak(h, dense, today),
		ScorePercent:   scorePercent,
		TodayCompleted: domain.IsCompleted(h, dense[len(dense)-1]),
	}
	return row, series, nil
}
