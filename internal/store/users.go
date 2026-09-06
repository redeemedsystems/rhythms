package store

import (
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Timezone     string
	CreatedAt    time.Time
}

func CreateUser(db *sql.DB, email, passwordHash, timezone string) (*User, error) {
	res, err := db.Exec(
		`INSERT INTO users (email, password_hash, timezone) VALUES (?, ?, ?)`,
		email, passwordHash, timezone,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return GetUserByID(db, id)
}

func GetUserByID(db *sql.DB, id int64) (*User, error) {
	return scanUser(db.QueryRow(
		`SELECT id, email, password_hash, timezone, created_at FROM users WHERE id = ?`, id,
	))
}

func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	return scanUser(db.QueryRow(
		`SELECT id, email, password_hash, timezone, created_at FROM users WHERE email = ?`, email,
	))
}

func UpdatePassword(db *sql.DB, userID int64, passwordHash string) error {
	_, err := db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, userID)
	return err
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	var createdAt string
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Timezone, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u.CreatedAt = parseTimestamp(createdAt)
	return &u, nil
}

func parseTimestamp(s string) time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05", s)
	return t
}
