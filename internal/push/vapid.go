// Package push wraps Web Push (VAPID) key management and notification sending.
package push

import (
	"database/sql"
	"errors"

	webpush "github.com/SherClockHolmes/webpush-go"

	"rhythms/internal/store"
)

const (
	configPublicKey  = "vapid_public_key"
	configPrivateKey = "vapid_private_key"
)

// LoadOrGenerateVAPIDKeys returns the app's VAPID keypair, generating and
// persisting one on first run so it survives restarts.
func LoadOrGenerateVAPIDKeys(db *sql.DB) (publicKey, privateKey string, err error) {
	publicKey, err = store.GetConfig(db, configPublicKey)
	if err == nil {
		privateKey, err = store.GetConfig(db, configPrivateKey)
		if err == nil {
			return publicKey, privateKey, nil
		}
	}
	if !errors.Is(err, store.ErrNotFound) {
		return "", "", err
	}

	privateKey, publicKey, err = webpush.GenerateVAPIDKeys()
	if err != nil {
		return "", "", err
	}
	if err := store.SetConfig(db, configPublicKey, publicKey); err != nil {
		return "", "", err
	}
	if err := store.SetConfig(db, configPrivateKey, privateKey); err != nil {
		return "", "", err
	}
	return publicKey, privateKey, nil
}
