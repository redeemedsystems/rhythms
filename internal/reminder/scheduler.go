// Package reminder polls for due habit reminders and sends Web Push
// notifications for them.
package reminder

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"rhythms/internal/domain"
)

// Sender delivers one push message to one subscription, returning the
// response status code (used to detect and prune expired subscriptions).
// Abstracted so the scheduler's due-reminder selection logic is testable
// without a real Web Push endpoint.
type Sender interface {
	Send(ctx context.Context, sub domain.PushSubscription, payload []byte) (statusCode int, err error)
}

// Scheduler polls for due habit reminders and pushes notifications for
// them. Construct with NewScheduler; run it with Run (or drive it
// tick-by-tick with Tick, as tests do).
type Scheduler struct {
	Habits        domain.HabitRepo
	Entries       domain.EntryRepo
	Reminders     domain.ReminderRepo
	ReminderLog   domain.ReminderLogRepo
	Subscriptions domain.PushSubscriptionRepo
	Sender        Sender

	// Interval between polls. Reminders are checked for "at or past due",
	// not an exact instant, so this bounds how late a notification can be.
	Interval time.Duration

	// Now is overridable for tests; defaults to time.Now in NewScheduler.
	Now func() time.Time
}

func NewScheduler(habits domain.HabitRepo, entries domain.EntryRepo, reminders domain.ReminderRepo,
	log domain.ReminderLogRepo, subs domain.PushSubscriptionRepo, sender Sender) *Scheduler {
	return &Scheduler{
		Habits: habits, Entries: entries, Reminders: reminders,
		ReminderLog: log, Subscriptions: subs, Sender: sender,
		Interval: time.Minute,
		Now:      time.Now,
	}
}

// Run polls until ctx is canceled. Intended to run as a background goroutine
// for the lifetime of the process.
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Tick(ctx)
		}
	}
}

// Tick checks every configured reminder once and sends push notifications
// for whichever are due. Exported so tests (and a manual trigger) can call
// a single pass directly instead of waiting on the ticker.
func (s *Scheduler) Tick(ctx context.Context) {
	active, err := s.Reminders.ListActive(ctx)
	if err != nil {
		slog.Error("reminder scheduler: list active reminders", "error", err)
		return
	}
	if len(active) == 0 {
		return
	}

	now := s.Now()
	today := domain.NewDate(now.Year(), now.Month(), now.Day())

	for habitID, rem := range active {
		sent, err := s.ReminderLog.WasSent(ctx, habitID, today)
		if err != nil {
			slog.Error("reminder scheduler: check sent log", "habit_id", habitID, "error", err)
			continue
		}

		h, err := s.Habits.Get(ctx, habitID)
		if err != nil {
			slog.Error("reminder scheduler: get habit", "habit_id", habitID, "error", err)
			continue
		}

		completed, err := s.isCompletedToday(ctx, h, today)
		if err != nil {
			slog.Error("reminder scheduler: check completion", "habit_id", habitID, "error", err)
			continue
		}

		if !rem.IsDue(now, sent, completed) {
			continue
		}

		// Only record the reminder as sent once a notification actually got
		// through (or there was truly no one to send to) — if every
		// delivery attempt fails (a transient network error, say), the next
		// tick should retry rather than silently giving up on the day.
		if s.notify(ctx, h) {
			if err := s.ReminderLog.MarkSent(ctx, habitID, today); err != nil {
				slog.Error("reminder scheduler: mark sent", "habit_id", habitID, "error", err)
			}
		}
	}
}

func (s *Scheduler) isCompletedToday(ctx context.Context, h domain.Habit, today domain.Date) (bool, error) {
	original, err := s.Entries.ListAll(ctx, h.ID)
	if err != nil {
		return false, err
	}
	computed := domain.ComputeEntries(h, original)
	dense := domain.DenseRange(computed, today, today)
	return domain.IsCompleted(h, dense[0]), nil
}

type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

// notify sends a push to every subscription and reports whether the
// reminder should be considered delivered: true if at least one send
// succeeded, or if there were no subscriptions at all to try (nothing more
// a retry could accomplish there). False means every attempt failed, so the
// caller should leave the reminder unmarked and let the next tick retry.
func (s *Scheduler) notify(ctx context.Context, h domain.Habit) bool {
	subs, err := s.Subscriptions.List(ctx)
	if err != nil {
		slog.Error("reminder scheduler: list subscriptions", "error", err)
		return false
	}
	if len(subs) == 0 {
		return true
	}

	body := h.Question
	if body == "" {
		body = "Time to " + h.Name
	}
	payload, err := json.Marshal(pushPayload{Title: h.Name, Body: body, URL: "/today"})
	if err != nil {
		slog.Error("reminder scheduler: marshal payload", "error", err)
		return false
	}

	delivered := false
	for _, sub := range subs {
		status, err := s.Sender.Send(ctx, sub, payload)
		if err != nil {
			slog.Error("reminder scheduler: send push", "endpoint", sub.Endpoint, "error", err)
			continue
		}
		// 404/410: the browser has invalidated this subscription (uninstalled,
		// permissions revoked, etc.) — it will never succeed again.
		if status == 404 || status == 410 {
			if err := s.Subscriptions.Delete(ctx, sub.Endpoint); err != nil {
				slog.Error("reminder scheduler: prune stale subscription", "error", err)
			}
			continue
		}
		if status >= 200 && status < 300 {
			delivered = true
		}
	}
	return delivered
}
