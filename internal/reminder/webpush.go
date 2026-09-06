package reminder

import (
	"context"

	webpush "github.com/SherClockHolmes/webpush-go"

	"rhythms/internal/domain"
)

// WebPushSender is the real Sender, backed by the Web Push protocol
// (VAPID-signed, encrypted per RFC 8291 — handled entirely by the
// webpush-go library rather than hand-rolled here).
type WebPushSender struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	// Subject is the "mailto:" or URL contact required by the VAPID spec,
	// so a push service can reach the sender about a misbehaving app.
	Subject string
}

func (w *WebPushSender) Send(ctx context.Context, sub domain.PushSubscription, payload []byte) (int, error) {
	resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}, &webpush.Options{
		Subscriber:      w.Subject,
		VAPIDPublicKey:  w.VAPIDPublicKey,
		VAPIDPrivateKey: w.VAPIDPrivateKey,
		TTL:             3600,
		Urgency:         webpush.UrgencyNormal,
	})
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}
