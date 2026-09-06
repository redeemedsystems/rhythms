CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE push_subscriptions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    endpoint    TEXT NOT NULL UNIQUE,
    p256dh      TEXT NOT NULL,
    auth        TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- Tracks which (habit, date) reminders have already fired, so the scheduler
-- (which polls periodically rather than firing at one exact instant) never
-- sends the same day's reminder twice.
CREATE TABLE reminder_log (
    habit_id INTEGER NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    date     TEXT NOT NULL,
    PRIMARY KEY (habit_id, date)
);
