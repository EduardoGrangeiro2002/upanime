import os
import yt_dlp


def download_ytdlp(embed_url: str, dest_path: str, on_progress=None) -> bool:
    os.makedirs(os.path.dirname(dest_path), exist_ok=True)

    last_pct = [0]

    def progress_hook(d):
        if d["status"] != "downloading":
            return
        total = d.get("total_bytes") or d.get("total_bytes_estimate") or 0
        if total <= 0:
            return
        downloaded = d.get("downloaded_bytes", 0)
        pct = int(downloaded / total * 100)
        if pct < last_pct[0] + 5:
            return
        last_pct[0] = pct
        if on_progress:
            speed = d.get("speed")
            eta = d.get("eta")
            speed_str = f"{speed / 1024 / 1024:.1f} MB/s" if speed else ""
            eta_str = f"{eta}s" if eta else ""
            on_progress(pct, speed_str, eta_str)

    ydl_opts = {
        "quiet": True,
        "no_warnings": True,
        "compat_opts": ["no-certifi"],
        "format": "best[ext=mp4]/best",
        "outtmpl": dest_path,
        "progress_hooks": [progress_hook],
        "postprocessor_args": {"ffmpeg": ["-movflags", "+faststart"]},
        "postprocessors": [{"key": "FFmpegVideoConvertor", "preferedformat": "mp4"}],
    }

    try:
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            ydl.download([embed_url])
        return True
    except Exception:
        return False
