package store

import (
	"database/sql"
	"errors"
	"time"
)

type PasswordReset struct {
	TokenHash string
	UserID    int64
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

func CreatePasswordReset(db *sql.DB, tokenHash string, userID int64, ttl time.Duration) error {
	expiresAt := time.Now().Add(ttl).UTC().Format("2006-01-02 15:04:05")
	_, err := db.Exec(
		`INSERT INTO password_resets (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		tokenHash, userID, expiresAt,
	)
	return err
}

// GetPasswordReset looks up a reset request by its token hash. It returns
// ErrNotFound if the token is unknown, expired, or already used, mirroring
// GetSession's handling of expiry.
func GetPasswordReset(db *sql.DB, tokenHash string) (*PasswordReset, error) {
	var pr PasswordReset
	var createdAt, expiresAt string
	var usedAt sql.NullString
	err := db.QueryRow(
		`SELECT token_hash, user_id, created_at, expires_at, used_at FROM password_resets WHERE token_hash = ?`,
		tokenHash,
	).Scan(&pr.TokenHash, &pr.UserID, &createdAt, &expiresAt, &usedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	pr.CreatedAt = parseTimestamp(createdAt)
	pr.ExpiresAt = parseTimestamp(expiresAt)
	if usedAt.Valid {
		t := parseTimestamp(usedAt.String)
		pr.UsedAt = &t
	}

	if pr.UsedAt != nil || pr.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrNotFound
	}

	return &pr, nil
}

func MarkPasswordResetUsed(db *sql.DB, tokenHash string) error {
	_, err := db.Exec(`UPDATE password_resets SET used_at = datetime('now') WHERE token_hash = ?`, tokenHash)
	return err
}
