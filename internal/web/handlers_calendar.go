package web

import (
	"context"
	"errors"
	"net/http"
	"time"

	"rhythms/internal/domain"
	"rhythms/internal/store"
)

const monthParamLayout = "2006-01"

type calendarDayVM struct {
	Day          int // 0 means this cell falls outside the month (blank)
	Date         string
	Value        domain.EntryValue
	NumericValue float64
	Completed    bool
	IsToday      bool
	IsFuture     bool
}

type calendarVM struct {
	HabitID     int64
	IsNumerical bool
	Unit        string
	Color       string
	MonthLabel  string // "September 2026"
	MonthParam  string // "2026-09", for prev/next links
	PrevMonth   string
	NextMonth   string
	Weeks       [][7]calendarDayVM
}

// parseMonthParam parses a "YYYY-MM" query param into the first day of that
// month, defaulting to the current month when s is empty or malformed —
// a bad month in a URL a user is free to hand-edit should just fall back
// silently rather than error the whole page.
func parseMonthParam(s string) domain.Date {
	t, err := time.Parse(monthParamLayout, s)
	if err != nil {
		return domain.Today().StartOfMonth()
	}
	return domain.NewDate(t.Year(), t.Month(), 1)
}

// buildCalendarVM renders one month's worth of a habit's history as a
// traditional day-numbered grid — a browsable, clickable complement to the
// detail page's 26-week heatmap. month must be the first day of the month
// to render.
func (s *Server) buildCalendarVM(ctx context.Context, h domain.Habit, month domain.Date) (calendarVM, error) {
	original, err := s.entries.ListAll(ctx, h.ID)
	if err != nil {
		return calendarVM{}, err
	}
	computed := domain.ComputeEntries(h, original)

	lastOfMonth := month.AddDays(month.DaysInMonth() - 1)
	dense := domain.DenseRange(computed, month, lastOfMonth)
	byDate := make(map[string]domain.Entry, len(dense))
	for _, e := range dense {
		byDate[e.Date.String()] = e
	}

	today := domain.Today()

	var days []calendarDayVM
	for i := 0; i < int(month.Weekday()); i++ {
		days = append(days, calendarDayVM{})
	}
	for day := 1; day <= month.DaysInMonth(); day++ {
		d := month.AddDays(day - 1)
		e, ok := byDate[d.String()]
		if !ok {
			e = domain.Entry{Date: d, Value: domain.Unknown}
		}
		days = append(days, calendarDayVM{
			Day:          day,
			Date:         d.String(),
			Value:        e.Value,
			NumericValue: e.NumericValue,
			Completed:    domain.IsCompleted(h, e),
			IsToday:      d.Equal(today),
			IsFuture:     d.After(today),
		})
	}
	for len(days)%7 != 0 {
		days = append(days, calendarDayVM{})
	}

	weeks := make([][7]calendarDayVM, 0, len(days)/7)
	for i := 0; i < len(days); i += 7 {
		var week [7]calendarDayVM
		copy(week[:], days[i:i+7])
		weeks = append(weeks, week)
	}

	return calendarVM{
		HabitID:     h.ID,
		IsNumerical: h.Type == domain.Numerical,
		Unit:        h.Unit,
		Color:       colorHex(h.Color),
		MonthLabel:  month.Format("January 2006"),
		MonthParam:  month.Format(monthParamLayout),
		PrevMonth:   month.AddDays(-1).StartOfMonth().Format(monthParamLayout),
		NextMonth:   lastOfMonth.AddDays(1).Format(monthParamLayout),
		Weeks:       weeks,
	}, nil
}

// handleHabitCalendar serves one month's calendar grid, both for the detail
// page's initial render and for month-navigation HTMX swaps.
func (s *Server) handleHabitCalendar(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h, err := s.habits.Get(r.Context(), mustUser(r).ID, id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.serverError(w, err)
		return
	}

	month := parseMonthParam(r.URL.Query().Get("month"))
	vm, err := s.buildCalendarVM(r.Context(), h, month)
	if err != nil {
		s.serverError(w, err)
		return
	}
	s.renderPartial(w, "habit_calendar", vm)
}
