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
	_ domain.HabitRepo = (*fakeHabitRepo)(nil)
	_ domain.EntryRepo = (*fakeEntryRepo)(nil)
)

type fakeHabitRepo struct {
	habits map[int64]domain.Habit
	nextID int64
}

func newFakeHabitRepo() *fakeHabitRepo {
	return &fakeHabitRepo{habits: map[int64]domain.Habit{}, nextID: 1}
}

func (f *fakeHabitRepo) List(ctx context.Context, includeArchived bool) ([]domain.Habit, error) {
	var out []domain.Habit
	for _, h := range f.habits {
		if h.Archived && !includeArchived {
			continue
		}
		out = append(out, h)
	}
	return out, nil
}

func (f *fakeHabitRepo) Get(ctx context.Context, id int64) (domain.Habit, error) {
	h, ok := f.habits[id]
	if !ok {
		return domain.Habit{}, fmt.Errorf("habit %d: %w", id, store.ErrNotFound)
	}
	return h, nil
}

func (f *fakeHabitRepo) Create(ctx context.Context, h domain.Habit) (int64, error) {
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
	f.nextID++
	f.habits[h.ID] = h
	return h.ID, nil
}

func (f *fakeHabitRepo) Update(ctx context.Context, h domain.Habit) error {
	if _, ok := f.habits[h.ID]; !ok {
		return fmt.Errorf("habit %d: %w", h.ID, store.ErrNotFound)
	}
	f.habits[h.ID] = h
	return nil
}

func (f *fakeHabitRepo) SetArchived(ctx context.Context, id int64, archived bool) error {
	h, ok := f.habits[id]
	if !ok {
		return fmt.Errorf("habit %d: %w", id, store.ErrNotFound)
	}
	h.Archived = archived
	f.habits[id] = h
	return nil
}

func (f *fakeHabitRepo) Delete(ctx context.Context, id int64) error {
	if _, ok := f.habits[id]; !ok {
		return fmt.Errorf("habit %d: %w", id, store.ErrNotFound)
	}
	delete(f.habits, id)
	return nil
}

func (f *fakeHabitRepo) Reorder(ctx context.Context, orderedIDs []int64) error {
	for pos, id := range orderedIDs {
		h, ok := f.habits[id]
		if !ok {
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
