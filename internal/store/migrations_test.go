package store

import (
	"path/filepath"
	"testing"
)

func TestMigrate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	for _, table := range []string{"users", "habits", "entries", "schema_migrations"} {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Errorf("expected table %q to exist: %v", table, err)
		}
	}

	// Re-opening (and thus re-migrating) an already-migrated database must
	// be idempotent, since Open runs Migrate on every process start.
	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer db2.Close()
}

func TestHabitsEntriesConstraints(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	userRes, err := db.Exec(`INSERT INTO users (email) VALUES (?)`, "test@example.com")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID, _ := userRes.LastInsertId()

	res, err := db.Exec(`INSERT INTO habits (user_id, uuid, name, position, habit_type) VALUES (?, ?, ?, ?, ?)`,
		userID, "uuid-1", "Meditate", 0, "YES_NO")
	if err != nil {
		t.Fatalf("insert habit: %v", err)
	}
	habitID, _ := res.LastInsertId()

	if _, err := db.Exec(`INSERT INTO entries (habit_id, date, value) VALUES (?, ?, ?)`, habitID, "2026-09-06", 2); err != nil {
		t.Fatalf("insert entry: %v", err)
	}

	// UNIQUE(habit_id, date) must reject a duplicate entry for the same day.
	if _, err := db.Exec(`INSERT INTO entries (habit_id, date, value) VALUES (?, ?, ?)`, habitID, "2026-09-06", 0); err == nil {
		t.Error("expected duplicate (habit_id, date) insert to fail, it succeeded")
	}

	// ON DELETE CASCADE must remove the habit's entries when the habit is deleted.
	if _, err := db.Exec(`DELETE FROM habits WHERE id = ?`, habitID); err != nil {
		t.Fatalf("delete habit: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM entries WHERE habit_id = ?`, habitID).Scan(&count); err != nil {
		t.Fatalf("count entries: %v", err)
	}
	if count != 0 {
		t.Errorf("expected cascade delete to remove entries, %d remain", count)
	}
}
