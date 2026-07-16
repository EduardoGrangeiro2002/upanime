import json
import os
import subprocess


def is_valid_video(path: str) -> bool:
    if not os.path.exists(path):
        return False
    result = subprocess.run(
        ["ffprobe", "-v", "error", "-show_entries", "format=format_name,duration", "-of", "json", path],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        return False
    try:
        fmt = json.loads(result.stdout).get("format", {})
    except json.JSONDecodeError:
        return False
    if "mp4" not in fmt.get("format_name", ""):
        return False
    try:
        return float(fmt.get("duration", 0)) > 0
    except (TypeError, ValueError):
        return False
