package reminder

import (
	"context"
	"sync"
	"testing"
	"time"

	"rhythms/internal/domain"
)

// --- minimal in-memory fakes, just enough for scheduler tests ---

type fakeHabits struct{ habits map[int64]domain.Habit }

func (f *fakeHabits) List(ctx context.Context, userID int64, includeArchived bool) ([]domain.Habit, error) {
	return nil, nil
}
func (f *fakeHabits) Get(ctx context.Context, userID, id int64) (domain.Habit, error) {
	return f.habits[id], nil
}
func (f *fakeHabits) GetAny(ctx context.Context, id int64) (domain.Habit, error) {
	return f.habits[id], nil
}
func (f *fakeHabits) Create(ctx context.Context, userID int64, h domain.Habit) (int64, error) {
	return 0, nil
}
func (f *fakeHabits) Update(ctx context.Context, userID int64, h domain.Habit) error { return nil }
func (f *fakeHabits) SetArchived(ctx context.Context, userID, id int64, archived bool) error {
	return nil
}
func (f *fakeHabits) Delete(ctx context.Context, userID, id int64) error { return nil }
func (f *fakeHabits) Reorder(ctx context.Context, userID int64, orderedIDs []int64) error {
	return nil
}

type fakeEntries struct{ byHabit map[int64][]domain.Entry }

func (f *fakeEntries) ListAll(ctx context.Context, habitID int64) ([]domain.Entry, error) {
	return f.byHabit[habitID], nil
}
func (f *fakeEntries) ListRange(ctx context.Context, habitID int64, from, to domain.Date) ([]domain.Entry, error) {
	return nil, nil
}
func (f *fakeEntries) Get(ctx context.Context, habitID int64, date domain.Date) (domain.Entry, bool, error) {
	return domain.Entry{}, false, nil
}
func (f *fakeEntries) Upsert(ctx context.Context, habitType domain.HabitType, e domain.Entry) error {
	return nil
}

type fakeReminders struct{ active map[int64]domain.Reminder }

func (f *fakeReminders) Get(ctx context.Context, habitID int64) (domain.Reminder, bool, error) {
	r, ok := f.active[habitID]
	return r, ok, nil
}
func (f *fakeReminders) Set(ctx context.Context, r domain.Reminder) error { return nil }
func (f *fakeReminders) Delete(ctx context.Context, habitID int64) error  { return nil }
func (f *fakeReminders) ListActive(ctx context.Context) (map[int64]domain.Reminder, error) {
	return f.active, nil
}

type fakeReminderLog struct {
	mu   sync.Mutex
	sent map[int64]domain.Date
}

func (f *fakeReminderLog) WasSent(ctx context.Context, habitID int64, date domain.Date) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.sent[habitID]
	return ok && d.Equal(date), nil
}
func (f *fakeReminderLog) MarkSent(ctx context.Context, habitID int64, date domain.Date) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sent == nil {
		f.sent = map[int64]domain.Date{}
	}
	f.sent[habitID] = date
	return nil
}

type fakeSubscriptions struct{ subs []domain.PushSubscription }

func (f *fakeSubscriptions) List(ctx context.Context, userID int64) ([]domain.PushSubscription, error) {
	var out []domain.PushSubscription
	for _, s := range f.subs {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}
func (f *fakeSubscriptions) Upsert(ctx context.Context, userID int64, s domain.PushSubscription) error {
	s.UserID = userID
	f.subs = append(f.subs, s)
	return nil
}
func (f *fakeSubscriptions) Delete(ctx context.Context, endpoint string) error {
	var out []domain.PushSubscription
	for _, s := range f.subs {
		if s.Endpoint != endpoint {
			out = append(out, s)
		}
	}
	f.subs = out
	return nil
}

type fakeSender struct {
	mu    sync.Mutex
	sent  []domain.PushSubscription
	reply map[string]int // endpoint -> status code to return
}

func (f *fakeSender) Send(ctx context.Context, sub domain.PushSubscription, payload []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, sub)
	if code, ok := f.reply[sub.Endpoint]; ok {
		return code, nil
	}
	return 201, nil
}

// --- tests ---

func newTestScheduler(t *testing.T) (*Scheduler, *fakeHabits, *fakeEntries, *fakeReminders, *fakeReminderLog, *fakeSubscriptions, *fakeSender) {
	t.Helper()
	habits := &fakeHabits{habits: map[int64]domain.Habit{}}
	entries := &fakeEntries{byHabit: map[int64][]domain.Entry{}}
	reminders := &fakeReminders{active: map[int64]domain.Reminder{}}
	log := &fakeReminderLog{}
	subs := &fakeSubscriptions{}
	sender := &fakeSender{}
	s := NewScheduler(habits, entries, reminders, log, subs, sender)
	return s, habits, entries, reminders, log, subs, sender
}

func TestTickSendsForDueIncompleteHabit(t *testing.T) {
	s, habits, _, reminders, _, subs, sender := newTestScheduler(t)
	monday900 := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return monday900 }

	habits.habits[1] = domain.Habit{ID: 1, Name: "Meditate", Type: domain.YesNo}
	reminders.active[1] = domain.Reminder{HabitID: 1, Hour: 8, Minute: 0, WeekdayMask: domain.AllWeekdaysMask}
	subs.subs = []domain.PushSubscription{{Endpoint: "https://push.example/abc"}}

	s.Tick(context.Background())

	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 push sent, got %d", len(sender.sent))
	}
}

func TestTickSkipsAlreadyCompletedHabit(t *testing.T) {
	s, habits, entries, reminders, _, subs, sender := newTestScheduler(t)
	monday900 := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return monday900 }

	habits.habits[1] = domain.Habit{ID: 1, Name: "Meditate", Type: domain.YesNo}
	entries.byHabit[1] = []domain.Entry{{HabitID: 1, Date: domain.NewDate(2026, 9, 7), Value: domain.YesManual}}
	reminders.active[1] = domain.Reminder{HabitID: 1, Hour: 8, Minute: 0, WeekdayMask: domain.AllWeekdaysMask}
	subs.subs = []domain.PushSubscription{{Endpoint: "https://push.example/abc"}}

	s.Tick(context.Background())

	if len(sender.sent) != 0 {
		t.Errorf("expected no push for an already-completed habit, got %d", len(sender.sent))
	}
}

func TestTickDoesNotDoubleSendSameDay(t *testing.T) {
	s, habits, _, reminders, _, subs, sender := newTestScheduler(t)
	monday900 := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return monday900 }

	habits.habits[1] = domain.Habit{ID: 1, Name: "Meditate", Type: domain.YesNo}
	reminders.active[1] = domain.Reminder{HabitID: 1, Hour: 8, Minute: 0, WeekdayMask: domain.AllWeekdaysMask}
	subs.subs = []domain.PushSubscription{{Endpoint: "https://push.example/abc"}}

	s.Tick(context.Background())
	s.Tick(context.Background())
	s.Tick(context.Background())

	if len(sender.sent) != 1 {
		t.Errorf("expected exactly 1 push across repeated ticks the same day, got %d", len(sender.sent))
	}
}

func TestTickRetriesWhenAllDeliveriesFail(t *testing.T) {
	s, habits, _, reminders, log, subs, sender := newTestScheduler(t)
	monday900 := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return monday900 }

	habits.habits[1] = domain.Habit{ID: 1, Name: "Meditate", Type: domain.YesNo}
	reminders.active[1] = domain.Reminder{HabitID: 1, Hour: 8, Minute: 0, WeekdayMask: domain.AllWeekdaysMask}
	subs.subs = []domain.PushSubscription{{Endpoint: "https://push.example/broken"}}
	sender.reply = map[string]int{"https://push.example/broken": 500} // a transient server error, not a dead subscription

	s.Tick(context.Background())

	sent, err := log.WasSent(context.Background(), 1, domain.NewDate(2026, 9, 7))
	if err != nil {
		t.Fatalf("WasSent: %v", err)
	}
	if sent {
		t.Error("a reminder where every delivery failed must not be marked sent — the next tick should retry")
	}
	if len(subs.subs) != 1 {
		t.Errorf("a 500 (unlike 404/410) must not prune the subscription, got %d remaining", len(subs.subs))
	}

	// A second tick should retry (still not marked sent).
	s.Tick(context.Background())
	if len(sender.sent) != 2 {
		t.Errorf("expected a retry on the next tick, got %d total send attempts", len(sender.sent))
	}
}

func TestTickMarksSentWithNoSubscriptions(t *testing.T) {
	s, habits, _, reminders, log, _, sender := newTestScheduler(t)
	monday900 := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return monday900 }

	habits.habits[1] = domain.Habit{ID: 1, Name: "Meditate", Type: domain.YesNo}
	reminders.active[1] = domain.Reminder{HabitID: 1, Hour: 8, Minute: 0, WeekdayMask: domain.AllWeekdaysMask}
	// No subscriptions at all.

	s.Tick(context.Background())

	sent, err := log.WasSent(context.Background(), 1, domain.NewDate(2026, 9, 7))
	if err != nil || !sent {
		t.Errorf("expected marked sent when there are no subscriptions to retry against: sent=%v err=%v", sent, err)
	}
	if len(sender.sent) != 0 {
		t.Errorf("expected no send attempts with zero subscriptions, got %d", len(sender.sent))
	}
}

func TestTickPrunesStaleSubscriptionOn410(t *testing.T) {
	s, habits, _, reminders, _, subs, sender := newTestScheduler(t)
	monday900 := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return monday900 }

	habits.habits[1] = domain.Habit{ID: 1, Name: "Meditate", Type: domain.YesNo}
	reminders.active[1] = domain.Reminder{HabitID: 1, Hour: 8, Minute: 0, WeekdayMask: domain.AllWeekdaysMask}
	subs.subs = []domain.PushSubscription{{Endpoint: "https://push.example/gone"}}
	sender.reply = map[string]int{"https://push.example/gone": 410}

	s.Tick(context.Background())

	if len(subs.subs) != 0 {
		t.Errorf("expected stale subscription to be pruned, still have %d", len(subs.subs))
	}
}

func TestTickOnlyNotifiesTheHabitOwnersSubscriptions(t *testing.T) {
	s, habits, _, reminders, _, subs, sender := newTestScheduler(t)
	monday900 := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return monday900 }

	habits.habits[1] = domain.Habit{ID: 1, UserID: 1, Name: "Meditate", Type: domain.YesNo}
	reminders.active[1] = domain.Reminder{HabitID: 1, Hour: 8, Minute: 0, WeekdayMask: domain.AllWeekdaysMask}
	subs.subs = []domain.PushSubscription{
		{UserID: 1, Endpoint: "https://push.example/owner"},
		{UserID: 2, Endpoint: "https://push.example/someone-else"},
	}

	s.Tick(context.Background())

	if len(sender.sent) != 1 || sender.sent[0].Endpoint != "https://push.example/owner" {
		t.Errorf("expected exactly 1 push to the habit owner's own subscription, got %+v", sender.sent)
	}
}

func TestTickSkipsInactiveWeekday(t *testing.T) {
	s, habits, _, reminders, _, subs, sender := newTestScheduler(t)
	monday900 := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC) // Monday
	s.Now = func() time.Time { return monday900 }

	habits.habits[1] = domain.Habit{ID: 1, Name: "Meditate", Type: domain.YesNo}
	reminders.active[1] = domain.Reminder{HabitID: 1, Hour: 8, Minute: 0, WeekdayMask: domain.WeekdayMaskFrom(time.Saturday, time.Sunday)}
	subs.subs = []domain.PushSubscription{{Endpoint: "https://push.example/abc"}}

	s.Tick(context.Background())

	if len(sender.sent) != 0 {
		t.Errorf("expected no push on an inactive weekday, got %d", len(sender.sent))
	}
}
