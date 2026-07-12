CREATE TABLE downloads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id INTEGER NOT NULL REFERENCES episodes(id),
    anime_id INTEGER NOT NULL REFERENCES animes(id),
    status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','resolving','downloading','completed','failed')),
    progress INTEGER NOT NULL DEFAULT 0,
    speed TEXT NOT NULL DEFAULT '',
    eta TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    dest_path TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
