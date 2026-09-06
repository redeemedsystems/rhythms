// Package scheduler runs the background reminder loop that sends scheduled
// Web Push notifications for habits with specific reminder times.
package scheduler

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"rhythms/internal/push"
	"rhythms/internal/store"

	"database/sql"
)

type Scheduler struct {
	db       *sql.DB
	sender   *push.Sender
	interval time.Duration
}

func New(db *sql.DB, sender *push.Sender) *Scheduler {
	return &Scheduler{db: db, sender: sender, interval: time.Minute}
}

// Run ticks every interval until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.tick(now)
		}
	}
}

func (s *Scheduler) tick(now time.Time) {
	habits, err := store.ListAllActiveHabits(s.db)
	if err != nil {
		slog.Error("scheduler: list habits", "err", err)
		return
	}

	for _, h := range habits {
		user, err := store.GetUserByID(s.db, h.UserID)
		if err != nil {
			continue
		}

		logDate, hhmm := store.LocalClock(user.Timezone, now)
		if !slices.Contains(h.ScheduleTimes, hhmm) {
			continue
		}

		sent, err := store.TryMarkNotified(s.db, h.ID, logDate, hhmm)
		if err != nil || !sent {
			continue
		}

		if amount, err := store.AmountForDate(s.db, h.ID, logDate); err == nil && store.IsComplete(h, amount) {
			continue // already logged for today, no need to nag
		}

		s.notifyUser(user.ID, h)
	}
}
