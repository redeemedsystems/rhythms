package store

import (
	"database/sql"
	"testing"
	"time"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	// A real temp file, not ":memory:" — SQLite's in-memory mode gives each
	// connection its own empty database, which breaks under database/sql's
	// connection pool (Open sets MaxOpenConns(4)).
	path := t.TempDir() + "/test.db"
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestMigrateIsIdempotent(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("second migrate call failed: %v", err)
	}
}

func TestCreateAndGetUser(t *testing.T) {
	db := testDB(t)

	u, err := CreateUser(db, "test@example.com", "hash", "America/Chicago")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.ID == 0 {
		t.Fatal("expected non-zero user id")
	}

	got, err := GetUserByEmail(db, "test@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if got.ID != u.ID {
		t.Fatalf("expected id %d, got %d", u.ID, got.ID)
	}

	if _, err := GetUserByEmail(db, "nope@example.com"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSessionLifecycle(t *testing.T) {
	db := testDB(t)
	u, _ := CreateUser(db, "sess@example.com", "hash", "UTC")

	if err := CreateSession(db, "sid1", u.ID, "csrf1", 24*time.Hour, "test-agent"); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sess, err := GetSession(db, "sid1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if sess.UserID != u.ID || sess.CSRFToken != "csrf1" {
		t.Fatalf("unexpected session: %+v", sess)
	}

	if err := DeleteSession(db, "sid1"); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := GetSession(db, "sid1"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestHabitCRUD(t *testing.T) {
	db := testDB(t)
	u, _ := CreateUser(db, "habits@example.com", "hash", "UTC")

	target := int64(3)
	h, err := CreateHabit(db, u.ID, HabitInput{
		Name:          "Pray",
		Type:          HabitTypeCount,
		TargetCount:   &target,
		Unit:          "times",
		ScheduleKind:  ScheduleSpecificTimes,
		ScheduleTimes: []string{"07:00", "12:00", "19:00"},
	})
	if err != nil {
		t.Fatalf("CreateHabit: %v", err)
	}
	if len(h.ScheduleTimes) != 3 {
		t.Fatalf("expected 3 schedule times, got %v", h.ScheduleTimes)
	}

	habits, err := ListHabitsForUser(db, u.ID, true)
	if err != nil {
		t.Fatalf("ListHabitsForUser: %v", err)
	}
	if len(habits) != 1 {
		t.Fatalf("expected 1 habit, got %d", len(habits))
	}

	if err := ArchiveHabit(db, h.ID); err != nil {
		t.Fatalf("ArchiveHabit: %v", err)
	}
	habits, _ = ListHabitsForUser(db, u.ID, true)
	if len(habits) != 0 {
		t.Fatalf("expected 0 active habits after archive, got %d", len(habits))
	}
}

func TestLogsAndCompletion(t *testing.T) {
	db := testDB(t)
	u, _ := CreateUser(db, "logs@example.com", "hash", "UTC")
	h, _ := CreateHabit(db, u.ID, HabitInput{
		Name: "Water", Type: HabitTypeBoolean, ScheduleKind: ScheduleDaily,
	})

	if err := CreateLog(db, h.ID, u.ID, "2026-09-01", 1, ""); err != nil {
		t.Fatalf("CreateLog: %v", err)
	}
	amount, err := AmountForDate(db, h.ID, "2026-09-01")
	if err != nil {
		t.Fatalf("AmountForDate: %v", err)
	}
	if amount != 1 {
		t.Fatalf("expected amount 1, got %d", amount)
	}

	if err := DeleteTodayLogs(db, h.ID, "2026-09-01"); err != nil {
		t.Fatalf("DeleteTodayLogs: %v", err)
	}
	amount, _ = AmountForDate(db, h.ID, "2026-09-01")
	if amount != 0 {
		t.Fatalf("expected amount 0 after delete, got %d", amount)
	}
}

func TestPushSubscriptionAndNotificationDedup(t *testing.T) {
	db := testDB(t)
	u, _ := CreateUser(db, "push@example.com", "hash", "UTC")
	h, _ := CreateHabit(db, u.ID, HabitInput{
		Name: "Pray", Type: HabitTypeCount, ScheduleKind: ScheduleSpecificTimes, ScheduleTimes: []string{"07:00"},
	})

	if err := SaveSubscription(db, u.ID, "https://push.example/abc", "p256dh", "auth", "ua"); err != nil {
		t.Fatalf("SaveSubscription: %v", err)
	}
	subs, err := ListSubscriptionsForUser(db, u.ID)
	if err != nil || len(subs) != 1 {
		t.Fatalf("expected 1 subscription, got %d (err=%v)", len(subs), err)
	}

	sent1, err := TryMarkNotified(db, h.ID, "2026-09-01", "07:00")
	if err != nil || !sent1 {
		t.Fatalf("expected first TryMarkNotified to succeed, got sent=%v err=%v", sent1, err)
	}
	sent2, err := TryMarkNotified(db, h.ID, "2026-09-01", "07:00")
	if err != nil || sent2 {
		t.Fatalf("expected second TryMarkNotified to be a no-op, got sent=%v err=%v", sent2, err)
	}
}

func TestAppConfigRoundTrip(t *testing.T) {
	db := testDB(t)
	if _, err := GetConfig(db, "missing"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := SetConfig(db, "k", "v1"); err != nil {
		t.Fatalf("SetConfig: %v", err)
	}
	if err := SetConfig(db, "k", "v2"); err != nil {
		t.Fatalf("SetConfig overwrite: %v", err)
	}
	v, err := GetConfig(db, "k")
	if err != nil || v != "v2" {
		t.Fatalf("expected v2, got %q (err=%v)", v, err)
	}
}
