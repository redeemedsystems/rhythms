package web

import (
	"errors"
	"html/template"
	"net/http"

	"rhythms/internal/charts"
	"rhythms/internal/domain"
	"rhythms/internal/store"
)

const scoreChartDays = 90

type habitDetailVM struct {
	ID            int64
	Name          string
	Question      string
	Description   string
	Color         string
	IsNumerical   bool
	Unit          string
	CurrentStreak int
	ScorePercent  int

	ScoreChart   template.HTML
	StreakChart  template.HTML
	HeatmapChart template.HTML
	WeekdayChart template.HTML

	Calendar calendarVM
}

// handleHabitDetail renders the habit detail page: score/streak/weekday
// charts plus a calendar heatmap, all server-rendered SVG built from a
// single pass over the habit's full computed history (see buildHabitVM's
// doc comment for why ComputeEntries needs the whole history, not a window).
func (s *Server) handleHabitDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h, err := s.habits.Get(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.serverError(w, err)
		return
	}

	original, err := s.entries.ListAll(r.Context(), id)
	if err != nil {
		s.serverError(w, err)
		return
	}
	computed := domain.ComputeEntries(h, original)

	today := domain.Today()
	from := today.AddDays(-(charts.HeatmapDefaultWeeks*7 - 1))
	for _, e := range computed {
		if e.Date.Before(from) {
			from = e.Date
		}
	}
	dense := domain.DenseRange(computed, from, today)

	scoreSeries := domain.ScoreSeries(h, dense)
	scoreWindow := scoreSeries
	if len(scoreWindow) > scoreChartDays {
		scoreWindow = scoreWindow[len(scoreWindow)-scoreChartDays:]
	}

	var weekdayCounts [7]int
	for _, e := range dense {
		if domain.IsCompleted(h, e) {
			weekdayCounts[int(e.Date.Weekday())]++
		}
	}

	color := colorHex(h.Color)
	vm := habitDetailVM{
		ID:            h.ID,
		Name:          h.Name,
		Question:      h.Question,
		Description:   h.Description,
		Color:         color,
		IsNumerical:   h.Type == domain.Numerical,
		Unit:          h.Unit,
		CurrentStreak: domain.CurrentStreak(h, dense, today),
		ScoreChart:    charts.ScoreLineChart(scoreWindow, color),
		StreakChart:   charts.StreakBarChart(domain.Streaks(h, dense), color),
		HeatmapChart:  charts.CalendarHeatmap(h, dense, charts.HeatmapDefaultWeeks, color),
		WeekdayChart:  charts.WeekdayBarChart(weekdayCounts, color),
	}
	if len(scoreSeries) > 0 {
		vm.ScorePercent = int(scoreSeries[len(scoreSeries)-1].Value*100 + 0.5)
	}

	calVM, err := s.buildCalendarVM(r.Context(), h, today.StartOfMonth())
	if err != nil {
		s.serverError(w, err)
		return
	}
	vm.Calendar = calVM

	s.render(w, "page_habit_detail", vm)
}
