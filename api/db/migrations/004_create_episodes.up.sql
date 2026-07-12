CREATE TABLE episodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    season_id INTEGER NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    anime_id INTEGER NOT NULL REFERENCES animes(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    number TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'episode' CHECK(type IN ('episode', 'movie', 'ova'))
);
