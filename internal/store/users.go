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
	DigestTime   string
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
		`SELECT id, email, password_hash, timezone, digest_time, created_at FROM users WHERE id = ?`, id,
	))
}

func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	return scanUser(db.QueryRow(
		`SELECT id, email, password_hash, timezone, digest_time, created_at FROM users WHERE email = ?`, email,
	))
}

func UpdatePassword(db *sql.DB, userID int64, passwordHash string) error {
	_, err := db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, userID)
	return err
}

// UpdateDigestTime sets the user's daily digest reminder time (HH:MM in
// their own timezone); an empty string disables it.
func UpdateDigestTime(db *sql.DB, userID int64, digestTime string) error {
	_, err := db.Exec(`UPDATE users SET digest_time = ? WHERE id = ?`, digestTime, userID)
	return err
}

// ListUsersWithDigest returns every user who has configured a daily digest
// reminder time, for the scheduler's digest tick.
func ListUsersWithDigest(db *sql.DB) ([]*User, error) {
	rows, err := db.Query(
		`SELECT id, email, password_hash, timezone, digest_time, created_at FROM users WHERE digest_time != ''`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var users []*User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func scanUser(row *sql.Row) (*User, error) {
	u, err := scanUserRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func scanUserRow(row rowScanner) (*User, error) {
	var u User
	var createdAt string
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Timezone, &u.DigestTime, &createdAt); err != nil {
		return nil, err
	}
	u.CreatedAt = parseTimestamp(createdAt)
	return &u, nil
}

func parseTimestamp(s string) time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05", s)
	return t
}
