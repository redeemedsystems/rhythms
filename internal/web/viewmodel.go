package web

import (
	"context"
	"fmt"

	"rhythms/internal/domain"
)

const historyDays = 7

type historyDayVM struct {
	Date    string
	Value   domain.EntryValue
	IsToday bool
}

type habitVM struct {
	ID         int64
	Name       string
	Question   string
	Color      string
	Archived   bool
	TodayDate  string
	TodayValue domain.EntryValue
	History    []historyDayVM // oldest first, historyDays long, ending today

	// ListFilterQuery is appended to this row's action URLs (e.g.
	// "?archived=1") so archiving/deleting from the archived view re-renders
	// that same view instead of silently switching back to the active one.
	ListFilterQuery string
}

func (s *Server) buildHabitVM(ctx context.Context, h domain.Habit) (habitVM, error) {
	today := domain.Today()
	from := today.AddDays(-(historyDays - 1))

	entries, err := s.entries.ListRange(ctx, h.ID, from, today)
	if err != nil {
		return habitVM{}, fmt.Errorf("list entries for habit %d: %w", h.ID, err)
	}
	byDate := make(map[string]domain.EntryValue, len(entries))
	for _, e := range entries {
		byDate[e.Date.String()] = e.Value
	}

	vm := habitVM{
		ID:        h.ID,
		Name:      h.Name,
		Question:  h.Question,
		Color:     colorHex(h.Color),
		Archived:  h.Archived,
		TodayDate: today.String(),
	}
	vm.TodayValue = byDate[vm.TodayDate] // zero value (No) if absent, which is correct

	for d := from; !d.After(today); d = d.AddDays(1) {
		ds := d.String()
		vm.History = append(vm.History, historyDayVM{
			Date:    ds,
			Value:   byDate[ds],
			IsToday: ds == vm.TodayDate,
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
