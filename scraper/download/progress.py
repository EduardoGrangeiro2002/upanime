import sqlite3
from datetime import datetime


def update_progress(db_path: str, download_id: int, progress: int, speed: str = "", eta: str = ""):
    conn = sqlite3.connect(db_path)
    try:
        conn.execute(
            "UPDATE downloads SET progress = ?, speed = ?, eta = ?, status = 'downloading', updated_at = ? WHERE id = ?",
            (progress, speed, eta, datetime.now().isoformat(sep=" ", timespec="seconds"), download_id),
        )
        conn.commit()
    finally:
        conn.close()
