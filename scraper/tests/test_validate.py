import sys
import os
import subprocess
import tempfile

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from download.validate import is_valid_video


def test_missing_file_is_invalid():
    assert is_valid_video("/nonexistent/file.mp4") is False


def test_html_disguised_as_mp4_is_invalid():
    with tempfile.NamedTemporaryFile(suffix=".mp4", delete=False) as f:
        f.write(b"<html><body>not a video</body></html>")
        path = f.name
    try:
        assert is_valid_video(path) is False
    finally:
        os.unlink(path)


def test_mpegts_disguised_as_mp4_is_invalid():
    path = tempfile.mktemp(suffix=".mp4")
    subprocess.run(
        ["ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=duration=1:size=64x64:rate=10",
         "-f", "mpegts", path],
        check=True,
    )
    try:
        assert is_valid_video(path) is False
    finally:
        os.unlink(path)


def test_real_mp4_is_valid():
    path = tempfile.mktemp(suffix=".mp4")
    subprocess.run(
        ["ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=duration=1:size=64x64:rate=10",
         "-f", "mp4", path],
        check=True,
    )
    try:
        assert is_valid_video(path) is True
    finally:
        os.unlink(path)
