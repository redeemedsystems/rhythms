CREATE TABLE push_subscriptions (
    id           INTEGER PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint     TEXT NOT NULL UNIQUE,
    p256dh_key   TEXT NOT NULL,
    auth_key     TEXT NOT NULL,
    user_agent   TEXT,
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    last_used_at TEXT
);

CREATE TABLE notification_log (
    id        INTEGER PRIMARY KEY,
    habit_id  INTEGER NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    log_date  TEXT NOT NULL,
    time_slot TEXT NOT NULL,
    sent_at   TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(habit_id, log_date, time_slot)
);
