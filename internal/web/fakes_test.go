package web

import (
	"context"
	"fmt"
	"sort"

	"rhythms/internal/domain"
	"rhythms/internal/store"
)

// fakeHabitRepo and fakeEntryRepo satisfy domain.HabitRepo/domain.EntryRepo
// entirely in memory, so handler tests exercise real routing/rendering
// against a fast, deterministic backend instead of a real SQLite file.

var (
	_ domain.HabitRepo            = (*fakeHabitRepo)(nil)
	_ domain.EntryRepo            = (*fakeEntryRepo)(nil)
	_ domain.ReminderRepo         = (*fakeReminderRepo)(nil)
	_ domain.PushSubscriptionRepo = (*fakePushSubscriptionRepo)(nil)
	_ domain.UserRepo             = (*fakeUserRepo)(nil)
)

type fakeHabitRepo struct {
	habits map[int64]domain.Habit
	nextID int64
}

func newFakeHabitRepo() *fakeHabitRepo {
	return &fakeHabitRepo{habits: map[int64]domain.Habit{}, nextID: 1}
}

func (f *fakeHabitRepo) List(ctx context.Context, userID int64, includeArchived bool) ([]domain.Habit, error) {
	var out []domain.Habit
	for _, h := range f.habits {
		if h.UserID != userID {
			continue
		}
		if h.Archived && !includeArchived {
			continue
		}
		out = append(out, h)
	}
	return out, nil
}

func (f *fakeHabitRepo) Get(ctx context.Context, userID, id int64) (domain.Habit, error) {
	h, ok := f.habits[id]
	if !ok || h.UserID != userID {
		return domain.Habit{}, fmt.Errorf("habit %d: %w", id, store.ErrNotFound)
	}
	return h, nil
}

func (f *fakeHabitRepo) GetAny(ctx context.Context, id int64) (domain.Habit, error) {
	h, ok := f.habits[id]
	if !ok {
		return domain.Habit{}, fmt.Errorf("habit %d: %w", id, store.ErrNotFound)
	}
	return h, nil
}

func (f *fakeHabitRepo) Create(ctx context.Context, userID int64, h domain.Habit) (int64, error) {
	if h.Type == "" {
		h.Type = domain.YesNo
	}
	if h.TargetType == "" {
		h.TargetType = domain.AtLeast
	}
	if h.Freq == (domain.Frequency{}) {
		h.Freq = domain.DailyFrequency()
	}
	h.ID = f.nextID
	h.UserID = userID
	f.nextID++
	f.habits[h.ID] = h
	return h.ID, nil
}

func (f *fakeHabitRepo) Update(ctx context.Context, userID int64, h domain.Habit) error {
	existing, ok := f.habits[h.ID]
	if !ok || existing.UserID != userID {
		return fmt.Errorf("habit %d: %w", h.ID, store.ErrNotFound)
	}
	h.UserID = userID
	f.habits[h.ID] = h
	return nil
}

func (f *fakeHabitRepo) SetArchived(ctx context.Context, userID, id int64, archived bool) error {
	h, ok := f.habits[id]
	if !ok || h.UserID != userID {
		return fmt.Errorf("habit %d: %w", id, store.ErrNotFound)
	}
	h.Archived = archived
	f.habits[id] = h
	return nil
}

func (f *fakeHabitRepo) Delete(ctx context.Context, userID, id int64) error {
	h, ok := f.habits[id]
	if !ok || h.UserID != userID {
		return fmt.Errorf("habit %d: %w", id, store.ErrNotFound)
	}
	delete(f.habits, id)
	return nil
}

func (f *fakeHabitRepo) Reorder(ctx context.Context, userID int64, orderedIDs []int64) error {
	for pos, id := range orderedIDs {
		h, ok := f.habits[id]
		if !ok || h.UserID != userID {
			return fmt.Errorf("habit %d: %w", id, store.ErrNotFound)
		}
		h.Position = pos
		f.habits[id] = h
	}
	return nil
}

type entryKey struct {
	habitID int64
	date    string
}

type fakeEntryRepo struct {
	entries map[entryKey]domain.Entry
}

func newFakeEntryRepo() *fakeEntryRepo {
	return &fakeEntryRepo{entries: map[entryKey]domain.Entry{}}
}

func (f *fakeEntryRepo) ListAll(ctx context.Context, habitID int64) ([]domain.Entry, error) {
	var out []domain.Entry
	for k, e := range f.entries {
		if k.habitID == habitID {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out, nil
}

func (f *fakeEntryRepo) ListRange(ctx context.Context, habitID int64, from, to domain.Date) ([]domain.Entry, error) {
	var out []domain.Entry
	for d := from; !d.After(to); d = d.AddDays(1) {
		if e, ok := f.entries[entryKey{habitID, d.String()}]; ok {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeEntryRepo) Get(ctx context.Context, habitID int64, date domain.Date) (domain.Entry, bool, error) {
	e, ok := f.entries[entryKey{habitID, date.String()}]
	return e, ok, nil
}

func (f *fakeEntryRepo) Upsert(ctx context.Context, habitType domain.HabitType, e domain.Entry) error {
	f.entries[entryKey{e.HabitID, e.Date.String()}] = e
	return nil
}

type fakeReminderRepo struct {
	reminders map[int64]domain.Reminder
}

func newFakeReminderRepo() *fakeReminderRepo {
	return &fakeReminderRepo{reminders: map[int64]domain.Reminder{}}
}

func (f *fakeReminderRepo) Get(ctx context.Context, habitID int64) (domain.Reminder, bool, error) {
	r, ok := f.reminders[habitID]
	return r, ok, nil
}

func (f *fakeReminderRepo) Set(ctx context.Context, r domain.Reminder) error {
	f.reminders[r.HabitID] = r
	return nil
}

func (f *fakeReminderRepo) Delete(ctx context.Context, habitID int64) error {
	delete(f.reminders, habitID)
	return nil
}

func (f *fakeReminderRepo) ListActive(ctx context.Context) (map[int64]domain.Reminder, error) {
	return f.reminders, nil
}

type fakePushSubscriptionRepo struct {
	subs map[string]domain.PushSubscription
}

func newFakePushSubscriptionRepo() *fakePushSubscriptionRepo {
	return &fakePushSubscriptionRepo{subs: map[string]domain.PushSubscription{}}
}

func (f *fakePushSubscriptionRepo) List(ctx context.Context, userID int64) ([]domain.PushSubscription, error) {
	var out []domain.PushSubscription
	for _, s := range f.subs {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakePushSubscriptionRepo) Upsert(ctx context.Context, userID int64, s domain.PushSubscription) error {
	s.UserID = userID
	f.subs[s.Endpoint] = s
	return nil
}

func (f *fakePushSubscriptionRepo) Delete(ctx context.Context, endpoint string) error {
	delete(f.subs, endpoint)
	return nil
}

type fakeUserRepo struct {
	users  map[int64]domain.User
	nextID int64
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[int64]domain.User{}, nextID: 1}
}

func (f *fakeUserRepo) Get(ctx context.Context, id int64) (domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return domain.User{}, fmt.Errorf("user %d: %w", id, store.ErrNotFound)
	}
	return u, nil
}

func (f *fakeUserRepo) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, fmt.Errorf("user %q: %w", email, store.ErrNotFound)
}

func (f *fakeUserRepo) List(ctx context.Context) ([]domain.User, error) {
	out := make([]domain.User, 0, len(f.users))
	for _, u := range f.users {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeUserRepo) Create(ctx context.Context, u domain.User) (int64, error) {
	for _, existing := range f.users {
		if existing.Email == u.Email {
			return 0, fmt.Errorf("user %q: %w", u.Email, store.ErrAlreadyExists)
		}
	}
	u.ID = f.nextID
	f.nextID++
	f.users[u.ID] = u
	return u.ID, nil
}

func (f *fakeUserRepo) SetGoogleSub(ctx context.Context, id int64, sub string) error {
	u, ok := f.users[id]
	if !ok {
		return fmt.Errorf("user %d: %w", id, store.ErrNotFound)
	}
	u.GoogleSub = sub
	f.users[id] = u
	return nil
}

func (f *fakeUserRepo) Delete(ctx context.Context, id int64) error {
	if _, ok := f.users[id]; !ok {
		return fmt.Errorf("user %d: %w", id, store.ErrNotFound)
	}
	delete(f.users, id)
	return nil
}
