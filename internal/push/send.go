package push

import (
	"encoding/json"
	"errors"
	"fmt"

	webpush "github.com/SherClockHolmes/webpush-go"

	"rhythms/internal/store"
)

// ErrSubscriptionExpired signals the endpoint is gone (404/410) and the
// caller should delete the subscription row.
var ErrSubscriptionExpired = errors.New("push subscription expired")

type Payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

type Sender struct {
	PublicKey  string
	PrivateKey string
	Subscriber string // a mailto: or https: contact URL, required by the VAPID spec
}

func NewSender(publicKey, privateKey, subscriber string) *Sender {
	return &Sender{PublicKey: publicKey, PrivateKey: privateKey, Subscriber: subscriber}
}

func (s *Sender) Send(sub *store.PushSubscription, payload Payload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := webpush.SendNotification(body, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			Auth:   sub.AuthKey,
			P256dh: sub.P256dhKey,
		},
	}, &webpush.Options{
		Subscriber:      s.Subscriber,
		VAPIDPublicKey:  s.PublicKey,
		VAPIDPrivateKey: s.PrivateKey,
		TTL:             60,
	})
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 404 || resp.StatusCode == 410 {
		return ErrSubscriptionExpired
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("push failed: status %d", resp.StatusCode)
	}
	return nil
}
