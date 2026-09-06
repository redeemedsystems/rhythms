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
