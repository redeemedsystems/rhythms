package scheduler

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"rhythms/internal/push"
	"rhythms/internal/store"
)

func (s *Scheduler) notifyUser(userID int64, h *store.Habit) {
	s.sendPush(userID, push.Payload{
		Title: "Rhythms",
		Body:  "Time for: " + h.Name,
		URL:   "/today",
	})
}

// sendDigest sends user a once-daily summary of active habits not yet
// logged for logDate. Skipped entirely for a user with no habits at all,
// rather than sending an empty "All done today!" congratulating them on
// nothing.
func (s *Scheduler) sendDigest(user *store.User, logDate string) {
	habits, err := store.ListHabitsForUser(s.db, user.ID, true)
	if err != nil {
		slog.Error("scheduler: list habits for digest", "err", err)
		return
	}
	if len(habits) == 0 {
		return
	}

	var remaining []string
	for _, h := range habits {
		amount, err := store.AmountForDate(s.db, h.ID, logDate)
		if err != nil {
			continue
		}
		if !store.IsComplete(h, amount) {
			remaining = append(remaining, h.Name)
		}
	}

	body := "All done today! 🎉"
	if len(remaining) > 0 {
		body = fmt.Sprintf("%d left today: %s", len(remaining), strings.Join(remaining, ", "))
	}

	s.sendPush(user.ID, push.Payload{Title: "Rhythms", Body: body, URL: "/today"})
}

// sendPush delivers payload to every one of userID's push subscriptions,
// dropping any that have expired.
func (s *Scheduler) sendPush(userID int64, payload push.Payload) {
	subs, err := store.ListSubscriptionsForUser(s.db, userID)
	if err != nil {
		slog.Error("scheduler: list subscriptions", "err", err)
		return
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
