CREATE TABLE habit_logs (
    id          INTEGER PRIMARY KEY,
    habit_id    INTEGER NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    occurred_at TEXT NOT NULL DEFAULT (datetime('now')),
    log_date    TEXT NOT NULL,
    amount      INTEGER NOT NULL DEFAULT 1,
    note        TEXT,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_logs_habit_date ON habit_logs(habit_id, log_date);
CREATE INDEX idx_logs_user_date  ON habit_logs(user_id, log_date);
