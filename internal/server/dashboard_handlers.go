package server

import (
	"html/template"
	"net/http"

	"rhythms/internal/auth"
	"rhythms/internal/charts"
	"rhythms/internal/store"
)

type HabitStats struct {
	Habit         *store.Habit
	CurrentStreak int
	BestStreak    int
	CompletionPct int
	ChartSVG      template.HTML
}

type DashboardPageData struct {
	Base
	Range string
	Stats []HabitStats
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildDashboardData(r, "week")
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.render.Page(w, "dashboard.html", *data)
}

func (s *Server) handleDashboardPartial(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildDashboardData(r, r.URL.Query().Get("range"))
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.render.Partial(w, "dashboard_stats", *data)
}

func (s *Server) buildDashboardData(r *http.Request, rangeParam string) (*DashboardPageData, error) {
	user := auth.UserFromContext(r.Context())

	rangeName, days := parseRange(rangeParam)

	habits, err := store.ListHabitsForUser(s.db, user.ID, true)
	if err != nil {
		return nil, err
	}

	now := store.NowIn(user.Timezone)
	to := now
	from := now.AddDate(0, 0, -(days - 1))
	dates := store.DateRange(from, to)

	stats := make([]HabitStats, 0, len(habits))
	for _, h := range habits {
		completion, err := store.CompletionByDate(s.db, h.ID, dates[0], dates[len(dates)-1])
		if err != nil {
			return nil, err
		}

		currentStreak, err := s.currentStreakFor(h, now)
		if err != nil {
			return nil, err
		}
		best := store.BestStreak(h, completion, dates)

		completeDays := 0
		for _, d := range dates {
			if store.IsComplete(h, completion[d]) {
				completeDays++
			}
		}
		pct := 0
		if len(dates) > 0 {
			pct = (completeDays * 100) / len(dates)
		}

		stats = append(stats, HabitStats{
			Habit:         h,
			CurrentStreak: currentStreak,
			BestStreak:    best,
			CompletionPct: pct,
			ChartSVG:      charts.CompletionStrip(h, dates, completion),
		})
	}

	return &DashboardPageData{
		Base:  s.baseFor(r, "dashboard"),
		Range: rangeName,
		Stats: stats,
	}, nil
}

func parseRange(v string) (name string, days int) {
	switch v {
	case "day":
		return "day", 1
	case "month":
		return "month", 30
	default:
		return "week", 7
	}
}
