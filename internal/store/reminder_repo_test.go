package store

import (
	"context"
	"path/filepath"
	"testing"

	"rhythms/internal/domain"
)

func TestReminderRepoSetGetDelete(t *testing.T) {
	ctx := context.Background()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	userID, err := NewUserRepo(db).Create(ctx, domain.User{Email: "test@example.com"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	habitID, err := NewHabitRepo(db).Create(ctx, userID, domain.Habit{Name: "Meditate"})
	if err != nil {
		t.Fatalf("Create habit: %v", err)
	}
	repo := NewReminderRepo(db)

	_, ok, err := repo.Get(ctx, habitID)
	if err != nil {
		t.Fatalf("Get before set: %v", err)
	}
	if ok {
		t.Fatal("Get before set: expected no reminder")
	}

	if err := repo.Set(ctx, domain.Reminder{HabitID: habitID, Hour: 8, Minute: 30, WeekdayMask: domain.AllWeekdaysMask}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	rem, ok, err := repo.Get(ctx, habitID)
	if err != nil || !ok {
		t.Fatalf("Get after set: ok=%v err=%v", ok, err)
	}
	if rem.Hour != 8 || rem.Minute != 30 || rem.WeekdayMask != domain.AllWeekdaysMask {
		t.Errorf("Get = %+v, want Hour=8 Minute=30 WeekdayMask=%d", rem, domain.AllWeekdaysMask)
	}

	// Set again with different values must overwrite, not duplicate (the
	// table's PRIMARY KEY is habit_id, one reminder per habit).
	if err := repo.Set(ctx, domain.Reminder{HabitID: habitID, Hour: 20, Minute: 0, WeekdayMask: 1}); err != nil {
		t.Fatalf("second Set: %v", err)
	}
	rem, ok, err = repo.Get(ctx, habitID)
	if err != nil || !ok || rem.Hour != 20 {
		t.Fatalf("Get after overwrite: rem=%+v ok=%v err=%v", rem, ok, err)
	}

	if err := repo.Delete(ctx, habitID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, ok, err = repo.Get(ctx, habitID)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if ok {
		t.Error("Get after delete: expected no reminder")
	}
}

func TestReminderCascadeDeletesWithHabit(t *testing.T) {
	ctx := context.Background()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	userID, err := NewUserRepo(db).Create(ctx, domain.User{Email: "test@example.com"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	habitID, err := NewHabitRepo(db).Create(ctx, userID, domain.Habit{Name: "Temp"})
	if err != nil {
		t.Fatalf("Create habit: %v", err)
	}
	repo := NewReminderRepo(db)
	if err := repo.Set(ctx, domain.Reminder{HabitID: habitID, Hour: 8, WeekdayMask: domain.AllWeekdaysMask}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := NewHabitRepo(db).Delete(ctx, userID, habitID); err != nil {
		t.Fatalf("Delete habit: %v", err)
	}

	_, ok, err := repo.Get(ctx, habitID)
	if err != nil {
		t.Fatalf("Get after habit delete: %v", err)
	}
	if ok {
		t.Error("expected reminder to be cascade-deleted with its habit")
	}
}
