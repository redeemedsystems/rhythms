package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// SettingsRepo is a small generic key-value store for app-level config that
// isn't tied to any habit — currently just the VAPID keypair, generated once
// on first run and persisted here rather than requiring manual setup.
type SettingsRepo struct {
	db *sql.DB
}

func NewSettingsRepo(db *sql.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) Get(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get setting %q: %w", key, err)
	}
	return value, true, nil
}

func (r *SettingsRepo) Set(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}

const (
	settingVAPIDPublicKey  = "vapid_public_key"
	settingVAPIDPrivateKey = "vapid_private_key"
)

// EnsureVAPIDKeys returns the app's persisted VAPID keypair, generating and
// storing one on first run. Every push subscription is only valid for the
// keypair it was created under, so this must stay stable across restarts —
// hence persisting it in the database rather than regenerating per process.
func (r *SettingsRepo) EnsureVAPIDKeys(ctx context.Context) (public, private string, err error) {
	public, hasPublic, err := r.Get(ctx, settingVAPIDPublicKey)
	if err != nil {
		return "", "", err
	}
	private, hasPrivate, err := r.Get(ctx, settingVAPIDPrivateKey)
	if err != nil {
		return "", "", err
	}
	if hasPublic && hasPrivate {
		return public, private, nil
	}

	private, public, err = webpush.GenerateVAPIDKeys()
	if err != nil {
		return "", "", fmt.Errorf("generate VAPID keys: %w", err)
	}
	if err := r.Set(ctx, settingVAPIDPublicKey, public); err != nil {
		return "", "", err
	}
	if err := r.Set(ctx, settingVAPIDPrivateKey, private); err != nil {
		return "", "", err
	}
	return public, private, nil
}

const settingSessionSecret = "session_hmac_secret"

// EnsureSessionSecret returns the app's persisted session-signing key,
// generating and storing one on first run — same pattern as
// EnsureVAPIDKeys, and for the same reason: it must stay stable across
// restarts, or every existing session cookie would fail to verify the
// moment the process restarted.
func (r *SettingsRepo) EnsureSessionSecret(ctx context.Context) ([]byte, error) {
	hexSecret, ok, err := r.Get(ctx, settingSessionSecret)
	if err != nil {
		return nil, err
	}
	if ok {
		secret, err := hex.DecodeString(hexSecret)
		if err != nil {
			return nil, fmt.Errorf("decode session secret: %w", err)
		}
		return secret, nil
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate session secret: %w", err)
	}
	if err := r.Set(ctx, settingSessionSecret, hex.EncodeToString(secret)); err != nil {
		return nil, err
	}
	return secret, nil
}
