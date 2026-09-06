package web

import (
	"net/http"

	"rhythms/internal/domain"
)

type todayPageVM struct {
	Date  string
	Items []habitVM
}

// handleToday renders a compact due-today checklist: every active habit not
// yet completed today. This is the practical, zero-infrastructure precursor
// to the M5 PWA "today" view and push reminders — a page you check, rather
// than a notification that finds you.
func (s *Server) handleToday(w http.ResponseWriter, r *http.Request) {
	habits, err := s.habits.List(r.Context(), false)
	if err != nil {
		s.serverError(w, err)
		return
	}

	var items []habitVM
	for _, h := range habits {
		vm, err := s.buildHabitVM(r.Context(), h)
		if err != nil {
			s.serverError(w, err)
			return
		}
		if vm.TodayCompleted {
			continue
		}
		items = append(items, vm)
	}

	s.render(w, "page_today", todayPageVM{
		Date:  domain.Today().Format("Monday, January 2"),
		Items: items,
	})
}
