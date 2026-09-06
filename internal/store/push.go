package store

import (
	"database/sql"
	"errors"
)

type PushSubscription struct {
	ID        int64
	UserID    int64
	Endpoint  string
	P256dhKey string
	AuthKey   string
	UserAgent string
}

func SaveSubscription(db *sql.DB, userID int64, endpoint, p256dh, auth, userAgent string) error {
	_, err := db.Exec(
		`INSERT INTO push_subscriptions (user_id, endpoint, p256dh_key, auth_key, user_agent)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(endpoint) DO UPDATE SET p256dh_key = excluded.p256dh_key, auth_key = excluded.auth_key`,
		userID, endpoint, p256dh, auth, nullableStr(userAgent),
	)
	return err
}

func DeleteSubscription(db *sql.DB, endpoint string) error {
	_, err := db.Exec(`DELETE FROM push_subscriptions WHERE endpoint = ?`, endpoint)
	return err
}

func ListSubscriptionsForUser(db *sql.DB, userID int64) ([]*PushSubscription, error) {
	rows, err := db.Query(
		`SELECT id, user_id, endpoint, p256dh_key, auth_key FROM push_subscriptions WHERE user_id = ?`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var subs []*PushSubscription
	for rows.Next() {
		var s PushSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dhKey, &s.AuthKey); err != nil {
			return nil, err
		}
		subs = append(subs, &s)
	}
	return subs, rows.Err()
}

// TryMarkNotified records that a reminder was sent for habitID/logDate/timeSlot,
// returning true only if this call actually inserted the row (i.e. no
// reminder had already been sent for that slot), so the caller knows whether
// to actually push a notification.
func TryMarkNotified(db *sql.DB, habitID int64, logDate, timeSlot string) (bool, error) {
	res, err := db.Exec(
		`INSERT OR IGNORE INTO notification_log (habit_id, log_date, time_slot) VALUES (?, ?, ?)`,
		habitID, logDate, timeSlot,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func GetConfig(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRow(`SELECT value FROM app_config WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return value, err
}

func SetConfig(db *sql.DB, key, value string) error {
	_, err := db.Exec(
		`INSERT INTO app_config (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}
