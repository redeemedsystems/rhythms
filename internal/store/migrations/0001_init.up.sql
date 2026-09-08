PRAGMA foreign_keys = ON;

-- google_sub is '' for an invited-but-not-yet-signed-in user (an admin
-- creates the row; the invited email's first successful Google sign-in
-- fills this in) — the partial unique index below only enforces uniqueness
-- once it's actually set, so multiple pending invites don't collide on ''.
CREATE TABLE users (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    email       TEXT NOT NULL UNIQUE,
    google_sub  TEXT NOT NULL DEFAULT '',
    is_admin    INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE UNIQUE INDEX idx_users_google_sub ON users(google_sub) WHERE google_sub != '';

CREATE TABLE habits (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    uuid            TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    question        TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    color           INTEGER NOT NULL DEFAULT 0,
    position        INTEGER NOT NULL,
    archived        INTEGER NOT NULL DEFAULT 0,
    habit_type      TEXT NOT NULL CHECK (habit_type IN ('YES_NO','NUMERICAL')),
    unit            TEXT NOT NULL DEFAULT '',
    target_value    REAL NOT NULL DEFAULT 0,
    target_type     TEXT NOT NULL DEFAULT 'AT_LEAST' CHECK (target_type IN ('AT_LEAST','AT_MOST')),
    freq_num        INTEGER NOT NULL DEFAULT 1,
    freq_den        INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX idx_habits_position ON habits(position);
CREATE INDEX idx_habits_archived ON habits(archived);
CREATE INDEX idx_habits_user_id ON habits(user_id);

CREATE TABLE entries (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    habit_id    INTEGER NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    date        TEXT NOT NULL,
    value       INTEGER NOT NULL,
    notes       TEXT NOT NULL DEFAULT '',
    UNIQUE(habit_id, date)
);

CREATE INDEX idx_entries_habit_date ON entries(habit_id, date);
