package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"rhythms/internal/domain"
)

var _ domain.HabitRepo = (*HabitRepo)(nil)

type HabitRepo struct {
	db *sql.DB
}

func NewHabitRepo(db *sql.DB) *HabitRepo {
	return &HabitRepo{db: db}
}

func (r *HabitRepo) List(ctx context.Context, includeArchived bool) ([]domain.Habit, error) {
	query := `SELECT id, uuid, name, question, description, color, position, archived,
		habit_type, unit, target_value, target_type, freq_num, freq_den
		FROM habits`
	if !includeArchived {
		query += ` WHERE archived = 0`
	}
	query += ` ORDER BY position ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list habits: %w", err)
	}
	defer rows.Close()

	var habits []domain.Habit
	for rows.Next() {
		h, err := scanHabit(rows)
		if err != nil {
			return nil, fmt.Errorf("scan habit: %w", err)
		}
		habits = append(habits, h)
	}
	return habits, rows.Err()
}

func (r *HabitRepo) Get(ctx context.Context, id int64) (domain.Habit, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, uuid, name, question, description, color,
		position, archived, habit_type, unit, target_value, target_type, freq_num, freq_den
		FROM habits WHERE id = ?`, id)
	h, err := scanHabit(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Habit{}, fmt.Errorf("habit %d: %w", id, ErrNotFound)
	}
	if err != nil {
		return domain.Habit{}, fmt.Errorf("get habit %d: %w", id, err)
	}
	return h, nil
}

func (r *HabitRepo) Create(ctx context.Context, h domain.Habit) (int64, error) {
	if h.UUID == "" {
		h.UUID = uuid.NewString()
	}
	if h.Type == "" {
		h.Type = domain.YesNo
	}
	if h.TargetType == "" {
		h.TargetType = domain.AtLeast
	}
	if h.Freq == (domain.Frequency{}) {
		h.Freq = domain.DailyFrequency()
	}

	var nextPos int
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(position) + 1, 0) FROM habits`).Scan(&nextPos); err != nil {
		return 0, fmt.Errorf("compute next position: %w", err)
	}

	res, err := r.db.ExecContext(ctx, `INSERT INTO habits
		(uuid, name, question, description, color, position, archived, habit_type,
		 unit, target_value, target_type, freq_num, freq_den)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		h.UUID, h.Name, h.Question, h.Description, h.Color, nextPos, boolToInt(h.Archived),
		string(h.Type), h.Unit, h.TargetValue, string(h.TargetType), h.Freq.Numerator, h.Freq.Denominator)
	if err != nil {
		return 0, fmt.Errorf("insert habit: %w", err)
	}
	return res.LastInsertId()
}

func (r *HabitRepo) Update(ctx context.Context, h domain.Habit) error {
	res, err := r.db.ExecContext(ctx, `UPDATE habits SET
		name = ?, question = ?, description = ?, color = ?, habit_type = ?,
		unit = ?, target_value = ?, target_type = ?, freq_num = ?, freq_den = ?
		WHERE id = ?`,
		h.Name, h.Question, h.Description, h.Color, string(h.Type),
		h.Unit, h.TargetValue, string(h.TargetType), h.Freq.Numerator, h.Freq.Denominator, h.ID)
	if err != nil {
		return fmt.Errorf("update habit %d: %w", h.ID, err)
	}
	return requireRowAffected(res, h.ID)
}

func (r *HabitRepo) SetArchived(ctx context.Context, id int64, archived bool) error {
	res, err := r.db.ExecContext(ctx, `UPDATE habits SET archived = ? WHERE id = ?`, boolToInt(archived), id)
	if err != nil {
		return fmt.Errorf("set archived habit %d: %w", id, err)
	}
	return requireRowAffected(res, id)
}

func (r *HabitRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM habits WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete habit %d: %w", id, err)
	}
	return requireRowAffected(res, id)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanHabit(row rowScanner) (domain.Habit, error) {
	var h domain.Habit
	var archived int
	var habitType, targetType string
	if err := row.Scan(&h.ID, &h.UUID, &h.Name, &h.Question, &h.Description, &h.Color,
		&h.Position, &archived, &habitType, &h.Unit, &h.TargetValue, &targetType,
		&h.Freq.Numerator, &h.Freq.Denominator); err != nil {
		return domain.Habit{}, err
	}
	h.Archived = archived != 0
	h.Type = domain.HabitType(habitType)
	h.TargetType = domain.TargetType(targetType)
	return h, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func requireRowAffected(res sql.Result, id int64) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("habit %d: %w", id, ErrNotFound)
	}
	return nil
}
