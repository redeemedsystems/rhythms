package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"rhythms/internal/domain"
)

var _ domain.UserRepo = (*UserRepo)(nil)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Get(ctx context.Context, id int64) (domain.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, email, google_sub, is_admin, created_at FROM users WHERE id = ?`, id)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, fmt.Errorf("user %d: %w", id, ErrNotFound)
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user %d: %w", id, err)
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, email, google_sub, is_admin, created_at FROM users WHERE email = ?`, email)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, fmt.Errorf("user %q: %w", email, ErrNotFound)
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user %q: %w", email, err)
	}
	return u, nil
}

func (r *UserRepo) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, email, google_sub, is_admin, created_at FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepo) Create(ctx context.Context, u domain.User) (int64, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO users (email, google_sub, is_admin) VALUES (?, ?, ?)`,
		u.Email, u.GoogleSub, boolToInt(u.IsAdmin))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return 0, fmt.Errorf("user %q: %w", u.Email, ErrAlreadyExists)
		}
		return 0, fmt.Errorf("insert user: %w", err)
	}
	return res.LastInsertId()
}

func (r *UserRepo) SetGoogleSub(ctx context.Context, id int64, sub string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE users SET google_sub = ? WHERE id = ?`, sub, id)
	if err != nil {
		return fmt.Errorf("set google_sub for user %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("user %d: %w", id, ErrNotFound)
	}
	return nil
}

func (r *UserRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("user %d: %w", id, ErrNotFound)
	}
	return nil
}

// EnsureAdmin guarantees the given email exists as an admin, creating it if
// missing or promoting it if it already exists as a non-admin — run on
// every startup (idempotent), same pattern as EnsureVAPIDKeys. This is the
// only bootstrap into an otherwise invite-only system, and also doubles as
// lockout recovery: restoring RHYTHMS_ADMIN_EMAIL and restarting always
// re-establishes an admin even if that row was deleted in-app.
func (r *UserRepo) EnsureAdmin(ctx context.Context, email string) error {
	u, err := r.GetByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		_, err := r.Create(ctx, domain.User{Email: email, IsAdmin: true})
		return err
	}
	if err != nil {
		return err
	}
	if u.IsAdmin {
		return nil
	}
	_, err = r.db.ExecContext(ctx, `UPDATE users SET is_admin = 1 WHERE id = ?`, u.ID)
	if err != nil {
		return fmt.Errorf("promote user %d to admin: %w", u.ID, err)
	}
	return nil
}

func scanUser(row rowScanner) (domain.User, error) {
	var u domain.User
	var isAdmin int
	if err := row.Scan(&u.ID, &u.Email, &u.GoogleSub, &isAdmin, &u.CreatedAt); err != nil {
		return domain.User{}, err
	}
	u.IsAdmin = isAdmin != 0
	return u, nil
}
