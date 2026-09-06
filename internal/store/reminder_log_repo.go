package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"rhythms/internal/domain"
)

var _ domain.ReminderLogRepo = (*ReminderLogRepo)(nil)

type ReminderLogRepo struct {
	db *sql.DB
}

func NewReminderLogRepo(db *sql.DB) *ReminderLogRepo {
	return &ReminderLogRepo{db: db}
}

func (r *ReminderLogRepo) WasSent(ctx context.Context, habitID int64, date domain.Date) (bool, error) {
	var one int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM reminder_log WHERE habit_id = ? AND date = ?`, habitID, date.String()).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check reminder log for habit %d on %s: %w", habitID, date, err)
	}
	return true, nil
}

func (r *ReminderLogRepo) MarkSent(ctx context.Context, habitID int64, date domain.Date) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO reminder_log (habit_id, date) VALUES (?, ?)
		ON CONFLICT(habit_id, date) DO NOTHING`, habitID, date.String())
	if err != nil {
		return fmt.Errorf("mark reminder sent for habit %d on %s: %w", habitID, date, err)
	}
	return nil
}
