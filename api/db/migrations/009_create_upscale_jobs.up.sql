CREATE TABLE upscale_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id INTEGER NOT NULL REFERENCES episodes(id),
    anime_id INTEGER NOT NULL REFERENCES animes(id),
    source_storage_key TEXT NOT NULL,
    result_storage_key TEXT NOT NULL DEFAULT '',
    runpod_job_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'queued'
        CHECK(status IN ('queued','submitting','processing','completed','failed')),
    error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
