CREATE TABLE habits (
    id             INTEGER PRIMARY KEY,
    user_id        INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    type           TEXT NOT NULL CHECK (type IN ('boolean','count','quantity')),
    target_count   INTEGER,
    unit           TEXT,
    schedule_kind  TEXT NOT NULL CHECK (schedule_kind IN ('daily','times_per_day','specific_times')),
    schedule_times TEXT,
    active         INTEGER NOT NULL DEFAULT 1,
    sort_order     INTEGER NOT NULL DEFAULT 0,
    created_at     TEXT NOT NULL DEFAULT (datetime('now')),
    archived_at    TEXT
);
CREATE INDEX idx_habits_user_active ON habits(user_id, active);
