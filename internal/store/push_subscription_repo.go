package store

import (
	"context"
	"database/sql"
	"fmt"

	"rhythms/internal/domain"
)

var _ domain.PushSubscriptionRepo = (*PushSubscriptionRepo)(nil)

type PushSubscriptionRepo struct {
	db *sql.DB
}

func NewPushSubscriptionRepo(db *sql.DB) *PushSubscriptionRepo {
	return &PushSubscriptionRepo{db: db}
}

func (r *PushSubscriptionRepo) List(ctx context.Context, userID int64) ([]domain.PushSubscription, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT user_id, endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = ?`, userID)
	if err != nil {
		return nil, fmt.Errorf("list push subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []domain.PushSubscription
	for rows.Next() {
		var s domain.PushSubscription
		if err := rows.Scan(&s.UserID, &s.Endpoint, &s.P256dh, &s.Auth); err != nil {
			return nil, fmt.Errorf("scan push subscription: %w", err)
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

// Upsert sets user_id on both insert and conflict-update, so if the same
// browser endpoint later subscribes under a different signed-in account on
// a shared device, ownership correctly transfers to whoever subscribed most
// recently rather than staying pinned to the first owner.
func (r *PushSubscriptionRepo) Upsert(ctx context.Context, userID int64, s domain.PushSubscription) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth) VALUES (?, ?, ?, ?)
		ON CONFLICT(endpoint) DO UPDATE SET user_id = excluded.user_id, p256dh = excluded.p256dh, auth = excluded.auth`,
		userID, s.Endpoint, s.P256dh, s.Auth)
	if err != nil {
		return fmt.Errorf("upsert push subscription: %w", err)
	}
	return nil
}

func (r *PushSubscriptionRepo) Delete(ctx context.Context, endpoint string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM push_subscriptions WHERE endpoint = ?`, endpoint)
	if err != nil {
		return fmt.Errorf("delete push subscription: %w", err)
	}
	return nil
}
