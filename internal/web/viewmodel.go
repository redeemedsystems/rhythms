package web

import (
	"context"
	"fmt"

	"rhythms/internal/domain"
)

const historyDays = 7

type historyDayVM struct {
	Date      string
	Value     domain.EntryValue
	Completed bool
	IsToday   bool
}

type habitVM struct {
	ID                int64
	Name              string
	Question          string
	Color             string
	Archived          bool
	IsNumerical       bool
	Unit              string
	TodayDate         string
	TodayValue        domain.EntryValue
	TodayNumericValue float64
	TodayCompleted    bool
	CurrentStreak     int
	ScorePercent      int
	History           []historyDayVM // oldest first, historyDays long, ending today

	// ListFilterQuery is appended to this row's action URLs (e.g.
	// "?archived=1") so archiving/deleting from the archived view re-renders
	// that same view instead of silently switching back to the active one.
	ListFilterQuery string
}

// buildHabitVM computes a habit's full display state: it fetches the
// habit's ENTIRE entry history (required by ComputeEntries — see its doc
// comment), derives the computed timeline once, then reads streak, score,
// and the display strip off of that single pass.
func (s *Server) buildHabitVM(ctx context.Context, h domain.Habit) (habitVM, error) {
	today := domain.Today()

	original, err := s.entries.ListAll(ctx, h.ID)
	if err != nil {
		return habitVM{}, fmt.Errorf("list entries for habit %d: %w", h.ID, err)
	}
	computed := domain.ComputeEntries(h, original)

	from := today.AddDays(-(historyDays - 1))
	for _, e := range computed {
		if e.Date.Before(from) {
			from = e.Date
		}
	}
	dense := domain.DenseRange(computed, from, today)

	vm := habitVM{
		ID:            h.ID,
		Name:          h.Name,
		Question:      h.Question,
		Color:         colorHex(h.Color),
		Archived:      h.Archived,
		IsNumerical:   h.Type == domain.Numerical,
		Unit:          h.Unit,
		TodayDate:     today.String(),
		CurrentStreak: domain.CurrentStreak(h, dense, today),
	}

	todayEntry := dense[len(dense)-1] // dense always ends at `to` == today
	vm.TodayValue = todayEntry.Value
	vm.TodayNumericValue = todayEntry.NumericValue
	vm.TodayCompleted = domain.IsCompleted(h, todayEntry)

	if series := domain.ScoreSeries(h, dense); len(series) > 0 {
		vm.ScorePercent = int(series[len(series)-1].Value*100 + 0.5)
	}

	// `from` is always <= today-(historyDays-1) by construction above, so
	// dense's last historyDays elements are exactly the display strip.
	for _, e := range dense[len(dense)-historyDays:] {
		vm.History = append(vm.History, historyDayVM{
			Date:      e.Date.String(),
			Value:     e.Value,
			Completed: domain.IsCompleted(h, e),
			IsToday:   e.Date.Equal(today),
		})
	}

	return vm, nil
}

func colorHex(index int) string {
	if index < 0 || index >= len(domain.Palette) {
		return domain.Palette[0]
	}
	return domain.Palette[index]
}
