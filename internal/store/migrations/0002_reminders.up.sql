CREATE TABLE reminders (
    habit_id        INTEGER PRIMARY KEY REFERENCES habits(id) ON DELETE CASCADE,
    hour            INTEGER NOT NULL,
    minute          INTEGER NOT NULL,
    weekday_mask    INTEGER NOT NULL DEFAULT 127
);
