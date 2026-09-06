package server

import (
	"net/http"
	"strconv"
	"time"

	"rhythms/internal/auth"
	"rhythms/internal/store"
)

// DayDetailData is the day-detail panel opened by clicking a calendar day:
// what was logged for one habit on one date, plus the refreshed HabitStats
// card (sent back as an out-of-band swap so the calendar reflects the
// change immediately).
type DayDetailData struct {
	DateLabel string
	Date      string
	Month     string
	Status    HabitWithStatus
	Logs      []store.HabitLog
	CardStats HabitStats
}

func (s *Server) handleHabitDay(w http.ResponseWriter, r *http.Request) {
	habit, date, month, ok := s.habitDayParams(w, r)
	if !ok {
		return
	}
	s.renderDayDetail(w, r, habit, date, month, false)
}

func (s *Server) handleHabitDayLog(w http.ResponseWriter, r *http.Request) {
	habit, date, month, ok := s.habitDayParams(w, r)
	if !ok {
		return
	}

	user := auth.UserFromContext(r.Context())
	if habit.Type == store.HabitTypeBoolean {
		amount, err := store.AmountForDate(s.db, habit.ID, date)
		if err != nil {
			s.internalError(w, err)
			return
		}
		if amount > 0 {
			err = store.DeleteTodayLogs(s.db, habit.ID, date)
		} else {
			err = store.CreateLog(s.db, habit.ID, user.ID, date, 1, "")
		}
		if err != nil {
			s.internalError(w, err)
			return
		}
	} else if err := store.CreateLog(s.db, habit.ID, user.ID, date, 1, ""); err != nil {
		s.internalError(w, err)
		return
	}

	s.renderDayDetail(w, r, habit, date, month, true)
}

func (s *Server) handleHabitDayLogDelete(w http.ResponseWriter, r *http.Request) {
	habit, date, month, ok := s.habitDayParams(w, r)
	if !ok {
		return
	}
	logID, err := strconv.ParseInt(r.PathValue("logID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := store.DeleteLog(s.db, logID, habit.ID); err != nil {
		s.internalError(w, err)
		return
	}
	s.renderDayDetail(w, r, habit, date, month, true)
}

// habitDayParams resolves the {id} path habit (scoped to the requesting
// user, same as the other /habits/{id}/... routes) and the date/month
// params shared by the day-detail routes. month defaults to date's own
// month so a plain GET (no month param) still works.
func (s *Server) habitDayParams(w http.ResponseWriter, r *http.Request) (habit *store.Habit, date, month string, ok bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return nil, "", "", false
	}
	habit, err = s.habitForUser(r, id)
	if err != nil {
		http.NotFound(w, r)
		return nil, "", "", false
	}

	date = r.FormValue("date")
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		http.Error(w, "invalid date", http.StatusBadRequest)
		return nil, "", "", false
	}

	month = r.FormValue("month")
	if _, err := time.Parse("2006-01", month); err != nil {
		month = parsedDate.Format("2006-01")
	}

	return habit, date, month, true
}

func (s *Server) renderDayDetail(w http.ResponseWriter, r *http.Request, habit *store.Habit, date, month string, includeCard bool) {
	amount, err := store.AmountForDate(s.db, habit.ID, date)
	if err != nil {
		s.internalError(w, err)
		return
	}
	logs, err := store.LogsForDate(s.db, habit.ID, date)
	if err != nil {
		s.internalError(w, err)
		return
	}
	dateParsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		s.internalError(w, err)
		return
	}

	data := DayDetailData{
		DateLabel: dateParsed.Format("Monday, January 2"),
		Date:      date,
		Month:     month,
		Status:    newHabitWithStatus(habit, amount),
		Logs:      logs,
	}

	if includeCard {
		user := auth.UserFromContext(r.Context())
		today := store.LocalDate(user.Timezone, time.Now())
		monthParsed, err := time.Parse("2006-01", month)
		if err != nil {
			s.internalError(w, err)
			return
		}
		card, err := s.habitStatsForMonth(habit, monthParsed.Year(), monthParsed.Month(), today)
		if err != nil {
			s.internalError(w, err)
			return
		}
		data.CardStats = card
	}

	s.render.Partial(w, "day_detail", data)
}
