import sqlite3
import tempfile
import os
import pytest


@pytest.fixture
def test_db():
    fd, path = tempfile.mkstemp(suffix=".db")
    os.close(fd)
    conn = sqlite3.connect(path)
    conn.executescript("""
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

        CREATE TABLE animes (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            title TEXT NOT NULL,
            url TEXT NOT NULL UNIQUE,
            image_url TEXT NOT NULL DEFAULT '',
            description TEXT NOT NULL DEFAULT '',
            scraper_id INTEGER NOT NULL REFERENCES scrapers(id),
            created_at TEXT NOT NULL DEFAULT (datetime('now')),
            updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        );

        CREATE TABLE seasons (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            anime_id INTEGER NOT NULL REFERENCES animes(id) ON DELETE CASCADE,
            number INTEGER NOT NULL,
            label TEXT NOT NULL DEFAULT '',
            type TEXT NOT NULL DEFAULT 'episode' CHECK(type IN ('episode', 'movie', 'ova')),
            UNIQUE(anime_id, number, type)
        );

        CREATE TABLE episodes (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            season_id INTEGER NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
            anime_id INTEGER NOT NULL REFERENCES animes(id) ON DELETE CASCADE,
            title TEXT NOT NULL,
            number TEXT NOT NULL DEFAULT '',
            url TEXT NOT NULL,
            type TEXT NOT NULL DEFAULT 'episode' CHECK(type IN ('episode', 'movie', 'ova'))
        );

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
    """)
    conn.close()
    yield path
    os.unlink(path)
