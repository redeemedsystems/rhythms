package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"rhythms/internal/domain"
)

func newTestDB(t *testing.T) *HabitRepo {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewHabitRepo(db)
}

func TestHabitRepoCreateGetList(t *testing.T) {
	ctx := context.Background()
	repo := newTestDB(t)

	id, err := repo.Create(ctx, domain.Habit{Name: "Meditate", Question: "Did you meditate?", Color: 3})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "Meditate" || got.Color != 3 {
		t.Errorf("Get = %+v, want Name=Meditate Color=3", got)
	}
	if got.Type != domain.YesNo {
		t.Errorf("Get.Type = %v, want %v (default)", got.Type, domain.YesNo)
	}
	if got.Freq != domain.DailyFrequency() {
		t.Errorf("Get.Freq = %v, want daily (default)", got.Freq)
	}
	if got.UUID == "" {
		t.Error("expected a generated UUID, got empty string")
	}

	habits, err := repo.List(ctx, false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(habits) != 1 {
		t.Fatalf("List returned %d habits, want 1", len(habits))
	}
}

func TestHabitRepoGetNotFound(t *testing.T) {
	repo := newTestDB(t)
	_, err := repo.Get(context.Background(), 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(999) error = %v, want ErrNotFound", err)
	}
}

func TestHabitRepoArchiveExcludedFromDefaultList(t *testing.T) {
	ctx := context.Background()
	repo := newTestDB(t)

	id, err := repo.Create(ctx, domain.Habit{Name: "Read"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.SetArchived(ctx, id, true); err != nil {
		t.Fatalf("SetArchived: %v", err)
	}

	active, err := repo.List(ctx, false)
	if err != nil {
		t.Fatalf("List(false): %v", err)
	}
	if len(active) != 0 {
		t.Errorf("List(false) returned %d habits, want 0 (archived excluded)", len(active))
	}

	all, err := repo.List(ctx, true)
	if err != nil {
		t.Fatalf("List(true): %v", err)
	}
	if len(all) != 1 {
		t.Errorf("List(true) returned %d habits, want 1 (archived included)", len(all))
	}
}

func TestHabitRepoUpdate(t *testing.T) {
	ctx := context.Background()
	repo := newTestDB(t)

	id, err := repo.Create(ctx, domain.Habit{Name: "Read"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	h, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	h.Name = "Read daily"
	h.Color = 7
	if err := repo.Update(ctx, h); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Name != "Read daily" || got.Color != 7 {
		t.Errorf("Get after update = %+v, want Name=Read daily Color=7", got)
	}
}

func TestHabitRepoDelete(t *testing.T) {
	ctx := context.Background()
	repo := newTestDB(t)

	id, err := repo.Create(ctx, domain.Habit{Name: "Temp"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.Delete(ctx, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.Get(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after delete error = %v, want ErrNotFound", err)
	}
	if err := repo.Delete(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("second Delete error = %v, want ErrNotFound", err)
	}
}

func TestHabitRepoCreateAssignsSequentialPositions(t *testing.T) {
	ctx := context.Background()
	repo := newTestDB(t)

	var last domain.Habit
	for _, name := range []string{"A", "B", "C"} {
		id, err := repo.Create(ctx, domain.Habit{Name: name})
		if err != nil {
			t.Fatalf("Create(%s): %v", name, err)
		}
		h, err := repo.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get(%s): %v", name, err)
		}
		if h.Position <= last.Position && name != "A" {
			t.Errorf("habit %s got position %d, want > %d", name, h.Position, last.Position)
		}
		last = h
	}
}
