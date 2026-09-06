package server

import (
	"net/http"
	"time"

	"rhythms/internal/auth"
	"rhythms/internal/store"
)

type CalendarDay struct {
	Date     string
	Day      int
	InMonth  bool
	Complete bool
	Today    bool
}

type HabitStats struct {
	Habit         *store.Habit
	CurrentStreak int
	BestStreak    int
	CompletionPct int
	Weeks         [][]CalendarDay
}

type DashboardPageData struct {
	Base
	MonthLabel string
	PrevMonth  string
	NextMonth  string
	Stats      []HabitStats
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildDashboardData(r, "")
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.render.Page(w, "dashboard.html", *data)
}

func (s *Server) handleDashboardPartial(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildDashboardData(r, r.URL.Query().Get("month"))
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.render.Partial(w, "dashboard_stats", *data)
}

func (s *Server) buildDashboardData(r *http.Request, monthParam string) (*DashboardPageData, error) {
	user := auth.UserFromContext(r.Context())
	now := store.NowIn(user.Timezone)
	today := now.Format("2006-01-02")

	year, month := now.Year(), now.Month()
	if parsed, err := time.Parse("2006-01", monthParam); err == nil {
		year, month = parsed.Year(), parsed.Month()
	}

	monthFirst := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	monthLast := monthFirst.AddDate(0, 1, -1)
	monthDates := store.DateRange(monthFirst, monthLast)

	weeks := store.MonthGrid(year, month)
	gridFrom := weeks[0][0].Format("2006-01-02")
	gridTo := weeks[len(weeks)-1][6].Format("2006-01-02")

	habits, err := store.ListHabitsForUser(s.db, user.ID, true)
	if err != nil {
		return nil, err
	}

	stats := make([]HabitStats, 0, len(habits))
	for _, h := range habits {
		completion, err := store.CompletionByDate(s.db, h.ID, gridFrom, gridTo)
		if err != nil {
			return nil, err
		}

		currentStreak, err := s.currentStreakFor(h, now)
		if err != nil {
			return nil, err
		}
		best := store.BestStreak(h, completion, monthDates)

		completeDays := 0
		for _, d := range monthDates {
			if store.IsComplete(h, completion[d]) {
				completeDays++
			}
		}
		pct := 0
		if len(monthDates) > 0 {
			pct = (completeDays * 100) / len(monthDates)
		}

		calWeeks := make([][]CalendarDay, len(weeks))
		for wi, week := range weeks {
			days := make([]CalendarDay, 7)
			for di, d := range week {
				dateStr := d.Format("2006-01-02")
				days[di] = CalendarDay{
					Date:     dateStr,
					Day:      d.Day(),
					InMonth:  d.Month() == month,
					Complete: store.IsComplete(h, completion[dateStr]),
					Today:    dateStr == today,
				}
			}
			calWeeks[wi] = days
		}

		stats = append(stats, HabitStats{
			Habit:         h,
			CurrentStreak: currentStreak,
			BestStreak:    best,
			CompletionPct: pct,
			Weeks:         calWeeks,
		})
	}

	prev := monthFirst.AddDate(0, -1, 0)
	next := monthFirst.AddDate(0, 1, 0)

	return &DashboardPageData{
		Base:       s.baseFor(r, "dashboard"),
		MonthLabel: monthFirst.Format("January 2006"),
		PrevMonth:  prev.Format("2006-01"),
		NextMonth:  next.Format("2006-01"),
		Stats:      stats,
	}, nil
}
