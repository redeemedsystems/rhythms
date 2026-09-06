package scheduler

import (
	"errors"
	"log/slog"

	"rhythms/internal/push"
	"rhythms/internal/store"
)

func (s *Scheduler) notifyUser(userID int64, h *store.Habit) {
	subs, err := store.ListSubscriptionsForUser(s.db, userID)
	if err != nil {
		slog.Error("scheduler: list subscriptions", "err", err)
		return
	}

	payload := push.Payload{
		Title: "Rhythms",
		Body:  "Time for: " + h.Name,
		URL:   "/today",
	}

	for _, sub := range subs {
		if err := s.sender.Send(sub, payload); err != nil {
			if errors.Is(err, push.ErrSubscriptionExpired) {
				_ = store.DeleteSubscription(s.db, sub.Endpoint)
				continue
			}
			slog.Warn("scheduler: push send failed", "err", err)
		}
	}
}
