CREATE TABLE seasons (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    anime_id INTEGER NOT NULL REFERENCES animes(id) ON DELETE CASCADE,
    number INTEGER NOT NULL,
    label TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT 'episode' CHECK(type IN ('episode', 'movie', 'ova')),
    UNIQUE(anime_id, number, type)
);
