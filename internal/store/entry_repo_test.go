package store

import (
	"context"
	"path/filepath"
	"testing"

	"rhythms/internal/domain"
)

func newTestEntryRepo(t *testing.T) (*EntryRepo, int64) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	habitID, err := NewHabitRepo(db).Create(context.Background(), domain.Habit{Name: "Test habit"})
	if err != nil {
		t.Fatalf("Create habit: %v", err)
	}
	return NewEntryRepo(db), habitID
}

func TestEntryRepoUpsertAndGet(t *testing.T) {
	ctx := context.Background()
	repo, habitID := newTestEntryRepo(t)
	date := domain.NewDate(2026, 9, 6)

	_, ok, err := repo.Get(ctx, habitID, date)
	if err != nil {
		t.Fatalf("Get before upsert: %v", err)
	}
	if ok {
		t.Fatal("Get before upsert: expected no entry, got one")
	}

	if err := repo.Upsert(ctx, domain.Entry{HabitID: habitID, Date: date, Value: domain.YesManual}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	entry, ok, err := repo.Get(ctx, habitID, date)
	if err != nil {
		t.Fatalf("Get after upsert: %v", err)
	}
	if !ok {
		t.Fatal("Get after upsert: expected an entry")
	}
	if entry.Value != domain.YesManual {
		t.Errorf("entry.Value = %v, want YesManual", entry.Value)
	}
}

func TestEntryRepoUpsertOverwritesSameDate(t *testing.T) {
	ctx := context.Background()
	repo, habitID := newTestEntryRepo(t)
	date := domain.NewDate(2026, 9, 6)

	if err := repo.Upsert(ctx, domain.Entry{HabitID: habitID, Date: date, Value: domain.YesManual}); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}
	if err := repo.Upsert(ctx, domain.Entry{HabitID: habitID, Date: date, Value: domain.No}); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	entry, ok, err := repo.Get(ctx, habitID, date)
	if err != nil || !ok {
		t.Fatalf("Get: ok=%v err=%v", ok, err)
	}
	if entry.Value != domain.No {
		t.Errorf("entry.Value = %v, want No (second upsert should overwrite, not duplicate)", entry.Value)
	}
}

func TestEntryRepoListRange(t *testing.T) {
	ctx := context.Background()
	repo, habitID := newTestEntryRepo(t)

	for day := 1; day <= 5; day++ {
		date := domain.NewDate(2026, 9, day)
		if err := repo.Upsert(ctx, domain.Entry{HabitID: habitID, Date: date, Value: domain.YesManual}); err != nil {
			t.Fatalf("Upsert day %d: %v", day, err)
		}
	}

	entries, err := repo.ListRange(ctx, habitID, domain.NewDate(2026, 9, 2), domain.NewDate(2026, 9, 4))
	if err != nil {
		t.Fatalf("ListRange: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("ListRange returned %d entries, want 3", len(entries))
	}
	if entries[0].Date.String() != "2026-09-02" || entries[2].Date.String() != "2026-09-04" {
		t.Errorf("ListRange bounds = [%s, %s], want [2026-09-02, 2026-09-04]", entries[0].Date, entries[2].Date)
	}
}
