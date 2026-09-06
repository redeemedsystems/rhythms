package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

const (
	HabitTypeBoolean  = "boolean"
	HabitTypeCount    = "count"
	HabitTypeQuantity = "quantity"

	ScheduleDaily         = "daily"
	ScheduleTimesPerDay   = "times_per_day"
	ScheduleSpecificTimes = "specific_times"
)

type Habit struct {
	ID            int64
	UserID        int64
	Name          string
	Type          string
	TargetCount   sql.NullInt64
	Unit          sql.NullString
	ScheduleKind  string
	ScheduleTimes []string
	Active        bool
	SortOrder     int
	CreatedAt     time.Time
	ArchivedAt    *time.Time
}

type HabitInput struct {
	Name          string
	Type          string
	TargetCount   *int64
	Unit          string
	ScheduleKind  string
	ScheduleTimes []string
}

func CreateHabit(db *sql.DB, userID int64, in HabitInput) (*Habit, error) {
	scheduleJSON, err := json.Marshal(in.ScheduleTimes)
	if err != nil {
		return nil, err
	}

	res, err := db.Exec(
		`INSERT INTO habits (user_id, name, type, target_count, unit, schedule_kind, schedule_times)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, in.Name, in.Type, nullableInt(in.TargetCount), nullableStr(in.Unit), in.ScheduleKind, string(scheduleJSON),
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return GetHabit(db, id)
}

func UpdateHabit(db *sql.DB, id int64, in HabitInput) error {
	scheduleJSON, err := json.Marshal(in.ScheduleTimes)
	if err != nil {
		return err
	}
	_, err = db.Exec(
		`UPDATE habits SET name = ?, type = ?, target_count = ?, unit = ?, schedule_kind = ?, schedule_times = ?
		 WHERE id = ?`,
		in.Name, in.Type, nullableInt(in.TargetCount), nullableStr(in.Unit), in.ScheduleKind, string(scheduleJSON), id,
	)
	return err
}

func ArchiveHabit(db *sql.DB, id int64) error {
	_, err := db.Exec(`UPDATE habits SET active = 0, archived_at = datetime('now') WHERE id = ?`, id)
	return err
}

func GetHabit(db *sql.DB, id int64) (*Habit, error) {
	return scanHabit(db.QueryRow(
		`SELECT id, user_id, name, type, target_count, unit, schedule_kind, schedule_times,
		        active, sort_order, created_at, archived_at
		 FROM habits WHERE id = ?`, id,
	))
}

func ListHabitsForUser(db *sql.DB, userID int64, activeOnly bool) ([]*Habit, error) {
	query := `SELECT id, user_id, name, type, target_count, unit, schedule_kind, schedule_times,
	                 active, sort_order, created_at, archived_at
	          FROM habits WHERE user_id = ?`
	if activeOnly {
		query += ` AND active = 1`
	}
	query += ` ORDER BY sort_order, id`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var habits []*Habit
	for rows.Next() {
		h, err := scanHabitRow(rows)
		if err != nil {
			return nil, err
		}
		habits = append(habits, h)
	}
	return habits, rows.Err()
}

// ListAllActiveHabits is used by the reminder scheduler, which operates
// across all users.
func ListAllActiveHabits(db *sql.DB) ([]*Habit, error) {
	rows, err := db.Query(
		`SELECT id, user_id, name, type, target_count, unit, schedule_kind, schedule_times,
		        active, sort_order, created_at, archived_at
		 FROM habits WHERE active = 1 AND schedule_kind = 'specific_times'`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var habits []*Habit
	for rows.Next() {
		h, err := scanHabitRow(rows)
		if err != nil {
			return nil, err
		}
		habits = append(habits, h)
	}
	return habits, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanHabit(row *sql.Row) (*Habit, error) {
	h, err := scanHabitRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return h, err
}

func scanHabitRow(row rowScanner) (*Habit, error) {
	var h Habit
	var targetCount sql.NullInt64
	var unit sql.NullString
	var scheduleTimes sql.NullString
	var createdAt string
	var archivedAt sql.NullString
	var active int

	if err := row.Scan(
		&h.ID, &h.UserID, &h.Name, &h.Type, &targetCount, &unit, &h.ScheduleKind, &scheduleTimes,
		&active, &h.SortOrder, &createdAt, &archivedAt,
	); err != nil {
		return nil, err
	}

	h.TargetCount = targetCount
	h.Unit = unit
	h.Active = active == 1
	h.CreatedAt = parseTimestamp(createdAt)
	if archivedAt.Valid {
		t := parseTimestamp(archivedAt.String)
		h.ArchivedAt = &t
	}
	if scheduleTimes.Valid && scheduleTimes.String != "" {
		_ = json.Unmarshal([]byte(scheduleTimes.String), &h.ScheduleTimes)
	}

	return &h, nil
}

func nullableInt(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
