package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"rhythms/internal/domain"
)

var _ domain.ReminderRepo = (*ReminderRepo)(nil)

type ReminderRepo struct {
	db *sql.DB
}

func NewReminderRepo(db *sql.DB) *ReminderRepo {
	return &ReminderRepo{db: db}
}

func (r *ReminderRepo) Get(ctx context.Context, habitID int64) (domain.Reminder, bool, error) {
	var rem domain.Reminder
	rem.HabitID = habitID
	err := r.db.QueryRowContext(ctx, `SELECT hour, minute, weekday_mask FROM reminders WHERE habit_id = ?`, habitID).
		Scan(&rem.Hour, &rem.Minute, &rem.WeekdayMask)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Reminder{}, false, nil
	}
	if err != nil {
		return domain.Reminder{}, false, fmt.Errorf("get reminder for habit %d: %w", habitID, err)
	}
	return rem, true, nil
}

func (r *ReminderRepo) Set(ctx context.Context, rem domain.Reminder) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO reminders (habit_id, hour, minute, weekday_mask)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(habit_id) DO UPDATE SET hour = excluded.hour, minute = excluded.minute, weekday_mask = excluded.weekday_mask`,
		rem.HabitID, rem.Hour, rem.Minute, rem.WeekdayMask)
	if err != nil {
		return fmt.Errorf("set reminder for habit %d: %w", rem.HabitID, err)
	}
	return nil
}

func (r *ReminderRepo) Delete(ctx context.Context, habitID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM reminders WHERE habit_id = ?`, habitID)
	if err != nil {
		return fmt.Errorf("delete reminder for habit %d: %w", habitID, err)
	}
	return nil
}

// ListActive joins reminders against non-archived habits, so a reminder on
// an archived habit is naturally excluded from what the scheduler polls.
func (r *ReminderRepo) ListActive(ctx context.Context) (map[int64]domain.Reminder, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT reminders.habit_id, hour, minute, weekday_mask
		FROM reminders JOIN habits ON habits.id = reminders.habit_id
		WHERE habits.archived = 0`)
	if err != nil {
		return nil, fmt.Errorf("list active reminders: %w", err)
	}
	defer rows.Close()

	out := make(map[int64]domain.Reminder)
	for rows.Next() {
		var rem domain.Reminder
		if err := rows.Scan(&rem.HabitID, &rem.Hour, &rem.Minute, &rem.WeekdayMask); err != nil {
			return nil, fmt.Errorf("scan reminder: %w", err)
		}
		out[rem.HabitID] = rem
	}
	return out, rows.Err()
}
