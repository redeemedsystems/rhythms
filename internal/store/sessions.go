package store

import (
	"database/sql"
	"errors"
	"time"
)

type Session struct {
	ID         string
	UserID     int64
	CSRFToken  string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastSeenAt time.Time
	UserAgent  string
}

func CreateSession(db *sql.DB, id string, userID int64, csrfToken string, ttl time.Duration, userAgent string) error {
	expiresAt := time.Now().Add(ttl).UTC().Format("2006-01-02 15:04:05")
	_, err := db.Exec(
		`INSERT INTO sessions (id, user_id, csrf_token, expires_at, user_agent) VALUES (?, ?, ?, ?, ?)`,
		id, userID, csrfToken, expiresAt, userAgent,
	)
	return err
}

func GetSession(db *sql.DB, id string) (*Session, error) {
	var s Session
	var createdAt, expiresAt, lastSeenAt string
	var userAgent sql.NullString
	err := db.QueryRow(
		`SELECT id, user_id, csrf_token, created_at, expires_at, last_seen_at, user_agent
		 FROM sessions WHERE id = ?`, id,
	).Scan(&s.ID, &s.UserID, &s.CSRFToken, &createdAt, &expiresAt, &lastSeenAt, &userAgent)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	s.CreatedAt = parseTimestamp(createdAt)
	s.ExpiresAt = parseTimestamp(expiresAt)
	s.LastSeenAt = parseTimestamp(lastSeenAt)
	s.UserAgent = userAgent.String

	if s.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrNotFound
	}

	return &s, nil
}

func TouchSession(db *sql.DB, id string) error {
	_, err := db.Exec(`UPDATE sessions SET last_seen_at = datetime('now') WHERE id = ?`, id)
	return err
}

func DeleteSession(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func DeleteExpiredSessions(db *sql.DB) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE expires_at < datetime('now')`)
	return err
}
