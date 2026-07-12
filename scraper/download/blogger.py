import os
import re
import threading
import httpx
from browser import get_page

_BLOGGER_RE = re.compile(r"blogger\.com/video\.g\?token=")
_GOOGLEVIDEO_RE = re.compile(r'"(https://[^"]*googlevideo\.com/videoplayback[^"]*)"')

_DOWNLOAD_HEADERS = {
    "Referer": "https://www.blogger.com/",
    "Origin": "https://www.blogger.com",
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
}


def is_blogger(url: str) -> bool:
    return bool(_BLOGGER_RE.search(url))


def _unescape_googlevideo_url(raw: str) -> str:
    return re.sub(r"\\{1,2}u([0-9a-fA-F]{4})", lambda m: chr(int(m.group(1), 16)), raw).rstrip("\\")


def resolve_blogger_url(embed_url: str) -> str | None:
    captured_url: list[str] = []
    event = threading.Event()

    with get_page() as page:
        def handle_response(response):
            if "batchexecute" not in response.url:
                return
            try:
                body = response.body().decode("utf-8")
            except Exception:
                return
            match = _GOOGLEVIDEO_RE.search(body)
            if not match:
                return
            captured_url.append(_unescape_googlevideo_url(match.group(1)))
            event.set()

        page.on("response", handle_response)
        page.goto(embed_url, wait_until="domcontentloaded")
        event.wait(timeout=15.0)

    if not captured_url:
        return None
    return captured_url[0]


def download_blogger(embed_url: str, dest_path: str, on_progress=None) -> bool:
    video_url = resolve_blogger_url(embed_url)
    if not video_url:
        return False

    if on_progress:
        on_progress(5)

    os.makedirs(os.path.dirname(dest_path), exist_ok=True)

    with httpx.Client(headers=_DOWNLOAD_HEADERS, follow_redirects=True, timeout=600.0) as client:
        with client.stream("GET", video_url) as resp:
            if resp.status_code != 200:
                return False
            total = int(resp.headers.get("content-length", 0))
            downloaded = 0
            last_pct = 0
            with open(dest_path, "wb") as f:
                for chunk in resp.iter_bytes(chunk_size=256 * 1024):
                    f.write(chunk)
                    downloaded += len(chunk)
                    if total <= 0:
                        continue
                    pct = int(downloaded / total * 100)
                    if pct < last_pct + 5:
                        continue
                    last_pct = pct
                    if on_progress:
                        on_progress(pct)

    return True
