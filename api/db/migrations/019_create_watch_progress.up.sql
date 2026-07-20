CREATE TABLE watch_progress (
    user_email TEXT NOT NULL,
    episode_id INTEGER NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    position_seconds REAL NOT NULL,
    duration_seconds REAL NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_email, episode_id)
);

CREATE INDEX idx_watch_progress_user_updated ON watch_progress(user_email, updated_at);
