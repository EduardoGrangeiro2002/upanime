CREATE TABLE scrapers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    domain TEXT NOT NULL UNIQUE,
    script_path TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO scrapers (name, domain, script_path)
VALUES ('animesonlinecc', 'animesonlinecc.to', 'sites/animesonlinecc.py');
