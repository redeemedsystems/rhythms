ALTER TABLE users ADD COLUMN digest_time TEXT NOT NULL DEFAULT '';

CREATE TABLE digest_log (
    user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    log_date TEXT NOT NULL,
    sent_at  TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, log_date)
);
