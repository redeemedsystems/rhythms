package domain

// PushSubscription is one browser's Web Push registration — the endpoint
// URL and keys needed to encrypt a message to it. A deployment can have
// several (e.g. phone + desktop), all notified for the same reminder.
type PushSubscription struct {
	UserID   int64
	Endpoint string
	P256dh   string
	Auth     string
}
