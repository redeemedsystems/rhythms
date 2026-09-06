// Package store is the SQLite-backed persistence layer: one repo type per
// domain concept (habits, entries, reminders, push subscriptions, settings),
// each implementing the corresponding interface from internal/domain so
// internal/web and internal/reminder depend only on those interfaces, never
// on this package's concrete types or on database/sql directly.
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open opens the SQLite database at path, sets pragmas suited to a
// single-writer web app, and runs any pending migrations.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// A single-process app talking to one SQLite file only needs one
	// writer at a time; capping open conns avoids SQLITE_BUSY churn.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if err := Migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}
