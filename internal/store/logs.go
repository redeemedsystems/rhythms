package store

import (
	"database/sql"
	"time"
)

type HabitLog struct {
	ID         int64
	HabitID    int64
	UserID     int64
	OccurredAt time.Time
	LogDate    string
	Amount     int
	Note       string
}

// CreateLog records one log entry for a habit on logDate (the user's local
// YYYY-MM-DD) and returns the new total amount logged for that habit/day.
func CreateLog(db *sql.DB, habitID, userID int64, logDate string, amount int, note string) error {
	_, err := db.Exec(
		`INSERT INTO habit_logs (habit_id, user_id, log_date, amount, note) VALUES (?, ?, ?, ?, ?)`,
		habitID, userID, logDate, amount, nullableStr(note),
	)
	return err
}

// DeleteTodayLogs removes all log entries for a habit on logDate, used to
// let a user "un-check" a boolean habit.
func DeleteTodayLogs(db *sql.DB, habitID int64, logDate string) error {
	_, err := db.Exec(`DELETE FROM habit_logs WHERE habit_id = ? AND log_date = ?`, habitID, logDate)
	return err
}

// AmountForDate returns the sum of amounts logged for a habit on logDate.
func AmountForDate(db *sql.DB, habitID int64, logDate string) (int, error) {
	var total sql.NullInt64
	err := db.QueryRow(
		`SELECT SUM(amount) FROM habit_logs WHERE habit_id = ? AND log_date = ?`, habitID, logDate,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	return int(total.Int64), nil
}

// LogsForDate returns the individual log entries for a habit on logDate,
// oldest first, as shown in the day-detail view.
func LogsForDate(db *sql.DB, habitID int64, logDate string) ([]HabitLog, error) {
	rows, err := db.Query(
		`SELECT id, habit_id, user_id, occurred_at, log_date, amount, COALESCE(note, '')
		 FROM habit_logs WHERE habit_id = ? AND log_date = ? ORDER BY occurred_at`, habitID, logDate,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	logs := make([]HabitLog, 0)
	for rows.Next() {
		var l HabitLog
		var occurredAt string
		if err := rows.Scan(&l.ID, &l.HabitID, &l.UserID, &occurredAt, &l.LogDate, &l.Amount, &l.Note); err != nil {
			return nil, err
		}
		l.OccurredAt = parseTimestamp(occurredAt)
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// DeleteLog removes one log entry, scoped to habitID so a habit's owner can
// only delete entries belonging to that habit.
func DeleteLog(db *sql.DB, logID, habitID int64) error {
	_, err := db.Exec(`DELETE FROM habit_logs WHERE id = ? AND habit_id = ?`, logID, habitID)
	return err
}

// CompletionByDate returns, for each log_date with any activity for the
// habit, the total amount logged that day.
func CompletionByDate(db *sql.DB, habitID int64, from, to string) (map[string]int, error) {
	rows, err := db.Query(
		`SELECT log_date, SUM(amount) FROM habit_logs
		 WHERE habit_id = ? AND log_date BETWEEN ? AND ?
		 GROUP BY log_date`, habitID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := map[string]int{}
	for rows.Next() {
		var date string
		var amount int
		if err := rows.Scan(&date, &amount); err != nil {
			return nil, err
		}
		out[date] = amount
	}
	return out, rows.Err()
}
