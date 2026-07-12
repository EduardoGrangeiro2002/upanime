import sqlite3
import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from download.progress import update_progress


def test_update_progress(test_db):
    conn = sqlite3.connect(test_db)
    conn.execute("INSERT INTO animes (title, url, scraper_id) VALUES ('Test', 'http://test', 1)")
    conn.execute("INSERT INTO seasons (anime_id, number, type) VALUES (1, 1, 'episode')")
    conn.execute("INSERT INTO episodes (season_id, anime_id, title, url, type) VALUES (1, 1, 'Ep1', 'http://ep1', 'episode')")
    conn.execute("INSERT INTO downloads (episode_id, anime_id) VALUES (1, 1)")
    conn.commit()
    conn.close()

    update_progress(test_db, 1, 50, "2.5 MB/s", "30s")

    conn = sqlite3.connect(test_db)
    row = conn.execute("SELECT progress, speed, eta, status FROM downloads WHERE id = 1").fetchone()
    conn.close()

    assert row[0] == 50
    assert row[1] == "2.5 MB/s"
    assert row[2] == "30s"
    assert row[3] == "downloading"
