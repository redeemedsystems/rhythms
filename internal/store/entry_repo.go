package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"rhythms/internal/domain"
)

var _ domain.EntryRepo = (*EntryRepo)(nil)

type EntryRepo struct {
	db *sql.DB
}

func NewEntryRepo(db *sql.DB) *EntryRepo {
	return &EntryRepo{db: db}
}

func (r *EntryRepo) ListAll(ctx context.Context, habitID int64) ([]domain.Entry, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT date, value, notes FROM entries
		WHERE habit_id = ? ORDER BY date ASC`, habitID)
	if err != nil {
		return nil, fmt.Errorf("list all entries for habit %d: %w", habitID, err)
	}
	defer rows.Close()

	var entries []domain.Entry
	for rows.Next() {
		e, err := scanEntry(rows, habitID)
		if err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *EntryRepo) ListRange(ctx context.Context, habitID int64, from, to domain.Date) ([]domain.Entry, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT date, value, notes FROM entries
		WHERE habit_id = ? AND date BETWEEN ? AND ? ORDER BY date ASC`,
		habitID, from.String(), to.String())
	if err != nil {
		return nil, fmt.Errorf("list entries for habit %d: %w", habitID, err)
	}
	defer rows.Close()

	var entries []domain.Entry
	for rows.Next() {
		e, err := scanEntry(rows, habitID)
		if err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *EntryRepo) Get(ctx context.Context, habitID int64, date domain.Date) (domain.Entry, bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT date, value, notes FROM entries
		WHERE habit_id = ? AND date = ?`, habitID, date.String())
	e, err := scanEntry(row, habitID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Entry{}, false, nil
	}
	if err != nil {
		return domain.Entry{}, false, fmt.Errorf("get entry for habit %d on %s: %w", habitID, date, err)
	}
	return e, true, nil
}

func (r *EntryRepo) Upsert(ctx context.Context, habitType domain.HabitType, e domain.Entry) error {
	value := encodeEntryValue(habitType, e)
	_, err := r.db.ExecContext(ctx, `INSERT INTO entries (habit_id, date, value, notes)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(habit_id, date) DO UPDATE SET value = excluded.value, notes = excluded.notes`,
		e.HabitID, e.Date.String(), value, e.Notes)
	if err != nil {
		return fmt.Errorf("upsert entry for habit %d on %s: %w", e.HabitID, e.Date, err)
	}
	return nil
}

func encodeEntryValue(habitType domain.HabitType, e domain.Entry) int {
	if habitType == domain.Numerical {
		return int(math.Round(e.NumericValue * 1000))
	}
	return int(e.Value)
}

// scanEntry decodes the stored column as both a boolean EntryValue and a
// numeric value; callers use whichever one domain.IsCompleted (and friends)
// says is meaningful for the habit's type.
func scanEntry(row rowScanner, habitID int64) (domain.Entry, error) {
	var e domain.Entry
	var dateStr string
	var value int
	if err := row.Scan(&dateStr, &value, &e.Notes); err != nil {
		return domain.Entry{}, err
	}
	date, err := domain.ParseDate(dateStr)
	if err != nil {
		return domain.Entry{}, fmt.Errorf("parse date %q: %w", dateStr, err)
	}
	e.HabitID = habitID
	e.Date = date
	e.Value = domain.EntryValue(value)
	e.NumericValue = float64(value) / 1000
	return e, nil
}
