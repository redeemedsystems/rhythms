package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"rhythms/internal/domain"
)

// newTestDB returns a HabitRepo backed by a fresh temp-file database, plus
// the id of a seeded user — habits.user_id is a NOT NULL foreign key, so
// every test needs an owner to create habits under.
func newTestDB(t *testing.T) (*HabitRepo, int64) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	userID, err := NewUserRepo(db).Create(context.Background(), domain.User{Email: "test@example.com"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return NewHabitRepo(db), userID
}

func TestHabitRepoCreateGetList(t *testing.T) {
	ctx := context.Background()
	repo, userID := newTestDB(t)

	id, err := repo.Create(ctx, userID, domain.Habit{Name: "Meditate", Question: "Did you meditate?", Color: 3})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, userID, id)
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
	if got.UserID != userID {
		t.Errorf("Get.UserID = %d, want %d", got.UserID, userID)
	}

	habits, err := repo.List(ctx, userID, false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(habits) != 1 {
		t.Fatalf("List returned %d habits, want 1", len(habits))
	}
}

func TestHabitRepoGetNotFound(t *testing.T) {
	repo, userID := newTestDB(t)
	_, err := repo.Get(context.Background(), userID, 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(999) error = %v, want ErrNotFound", err)
	}
}

func TestHabitRepoGetWrongOwnerNotFound(t *testing.T) {
	ctx := context.Background()
	repo, userID := newTestDB(t)
	otherUserID, err := NewUserRepo(repo.db).Create(ctx, domain.User{Email: "other@example.com"})
	if err != nil {
		t.Fatalf("seed second user: %v", err)
	}

	id, err := repo.Create(ctx, userID, domain.Habit{Name: "Private"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := repo.Get(ctx, otherUserID, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get by wrong owner error = %v, want ErrNotFound (must not leak existence)", err)
	}
	if err := repo.Delete(ctx, otherUserID, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete by wrong owner error = %v, want ErrNotFound", err)
	}

	// The rightful owner can still see it — confirms the row wasn't
	// actually affected by the wrong-owner calls above.
	if _, err := repo.Get(ctx, userID, id); err != nil {
		t.Errorf("Get by rightful owner after wrong-owner attempts: %v", err)
	}
}

func TestHabitRepoArchiveExcludedFromDefaultList(t *testing.T) {
	ctx := context.Background()
	repo, userID := newTestDB(t)

	id, err := repo.Create(ctx, userID, domain.Habit{Name: "Read"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.SetArchived(ctx, userID, id, true); err != nil {
		t.Fatalf("SetArchived: %v", err)
	}

	active, err := repo.List(ctx, userID, false)
	if err != nil {
		t.Fatalf("List(false): %v", err)
	}
	if len(active) != 0 {
		t.Errorf("List(false) returned %d habits, want 0 (archived excluded)", len(active))
	}

	all, err := repo.List(ctx, userID, true)
	if err != nil {
		t.Fatalf("List(true): %v", err)
	}
	if len(all) != 1 {
		t.Errorf("List(true) returned %d habits, want 1 (archived included)", len(all))
	}
}

func TestHabitRepoUpdate(t *testing.T) {
	ctx := context.Background()
	repo, userID := newTestDB(t)

	id, err := repo.Create(ctx, userID, domain.Habit{Name: "Read"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	h, err := repo.Get(ctx, userID, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	h.Name = "Read daily"
	h.Color = 7
	if err := repo.Update(ctx, userID, h); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.Get(ctx, userID, id)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Name != "Read daily" || got.Color != 7 {
		t.Errorf("Get after update = %+v, want Name=Read daily Color=7", got)
	}
}

func TestHabitRepoDelete(t *testing.T) {
	ctx := context.Background()
	repo, userID := newTestDB(t)

	id, err := repo.Create(ctx, userID, domain.Habit{Name: "Temp"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.Delete(ctx, userID, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.Get(ctx, userID, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after delete error = %v, want ErrNotFound", err)
	}
	if err := repo.Delete(ctx, userID, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("second Delete error = %v, want ErrNotFound", err)
	}
}

func TestHabitRepoCreateAssignsSequentialPositions(t *testing.T) {
	ctx := context.Background()
	repo, userID := newTestDB(t)

	var last domain.Habit
	for _, name := range []string{"A", "B", "C"} {
		id, err := repo.Create(ctx, userID, domain.Habit{Name: name})
		if err != nil {
			t.Fatalf("Create(%s): %v", name, err)
		}
		h, err := repo.Get(ctx, userID, id)
		if err != nil {
			t.Fatalf("Get(%s): %v", name, err)
		}
		if h.Position <= last.Position && name != "A" {
			t.Errorf("habit %s got position %d, want > %d", name, h.Position, last.Position)
		}
		last = h
	}
}

func TestHabitRepoGetAnyIgnoresOwnership(t *testing.T) {
	ctx := context.Background()
	repo, userID := newTestDB(t)
	otherUserID, err := NewUserRepo(repo.db).Create(ctx, domain.User{Email: "other2@example.com"})
	if err != nil {
		t.Fatalf("seed second user: %v", err)
	}

	id, err := repo.Create(ctx, userID, domain.Habit{Name: "Anyone's"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetAny(ctx, id)
	if err != nil {
		t.Fatalf("GetAny: %v", err)
	}
	if got.UserID != userID {
		t.Errorf("GetAny.UserID = %d, want %d", got.UserID, userID)
	}
	_ = otherUserID
}

func TestHabitRepoReorderRejectsForeignID(t *testing.T) {
	ctx := context.Background()
	repo, userID := newTestDB(t)
	otherUserID, err := NewUserRepo(repo.db).Create(ctx, domain.User{Email: "other3@example.com"})
	if err != nil {
		t.Fatalf("seed second user: %v", err)
	}

	mine, err := repo.Create(ctx, userID, domain.Habit{Name: "Mine"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	theirs, err := repo.Create(ctx, otherUserID, domain.Habit{Name: "Theirs"})
	if err != nil {
		t.Fatalf("Create (other user): %v", err)
	}

	if err := repo.Reorder(ctx, userID, []int64{mine, theirs}); !errors.Is(err, ErrNotFound) {
		t.Errorf("Reorder with a foreign id error = %v, want ErrNotFound", err)
	}
}
