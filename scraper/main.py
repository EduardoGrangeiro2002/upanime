import json
import os
import sys
import tempfile
from urllib.parse import urlparse

import certifi


def ensure_ca_bundle() -> str:
    extra = os.path.join(os.path.dirname(os.path.abspath(__file__)), "certs", "letsencrypt-yr.pem")
    if not os.path.exists(extra):
        return certifi.where()
    bundle = os.path.join(tempfile.gettempdir(), "upanime-ca.pem")
    tmp = f"{bundle}.{os.getpid()}"
    with open(certifi.where()) as base, open(extra) as ex, open(tmp, "w") as out:
        out.write(base.read())
        out.write("\n")
        out.write(ex.read())
    os.replace(tmp, bundle)
    return bundle


os.environ.setdefault("SSL_CERT_FILE", ensure_ca_bundle())

from sites import get_scraper
from browser import close_browser
from download.blogger import is_blogger, download_blogger
from download.ytdlp import download_ytdlp
from download.progress import update_progress
from download.validate import is_valid_video


def cmd_scrape(url: str):
    domain = urlparse(url).netloc
    scraper = get_scraper(domain)
    if not scraper:
        print(json.dumps({"error": f"no scraper for domain: {domain}"}), file=sys.stderr)
        sys.exit(1)

    result = scraper.scrape_anime(url)
    print(json.dumps(result, ensure_ascii=False))


def cmd_download(episode_url: str, dest_path: str, download_id: int, db_path: str):
    domain = urlparse(episode_url).netloc
    scraper = get_scraper(domain)
    if not scraper:
        print(f"no scraper for domain: {domain}", file=sys.stderr)
        sys.exit(1)

    sources = scraper.scrape_episode(episode_url)
    if not sources:
        print("no sources found", file=sys.stderr)
        sys.exit(1)

    def on_progress_blogger(pct):
        update_progress(db_path, download_id, pct)

    def on_progress_ytdlp(pct, speed="", eta=""):
        update_progress(db_path, download_id, pct, speed, eta)

    for source in sources:
        embed_url = source["embed_url"]
        if is_blogger(embed_url):
            ok = download_blogger(embed_url, dest_path, on_progress=on_progress_blogger)
            if ok and is_valid_video(dest_path):
                update_progress(db_path, download_id, 100)
                return

    for source in sources:
        embed_url = source["embed_url"]
        ok = download_ytdlp(embed_url, dest_path, on_progress=on_progress_ytdlp)
        if ok and is_valid_video(dest_path):
            update_progress(db_path, download_id, 100)
            return

    print("all download methods failed", file=sys.stderr)
    sys.exit(1)


def main():
    if len(sys.argv) < 2:
        print("usage: main.py <scrape|download> [args...]", file=sys.stderr)
        sys.exit(1)

    command = sys.argv[1]

    if command == "scrape":
        if len(sys.argv) < 3:
            print("usage: main.py scrape <url>", file=sys.stderr)
            sys.exit(1)
        cmd_scrape(sys.argv[2])
        close_browser()
        return

    if command == "download":
        if len(sys.argv) < 6:
            print("usage: main.py download <episode_url> <dest_path> <download_id> <db_path>", file=sys.stderr)
            sys.exit(1)
        cmd_download(sys.argv[2], sys.argv[3], int(sys.argv[4]), sys.argv[5])
        close_browser()
        return

    print(f"unknown command: {command}", file=sys.stderr)
    sys.exit(1)


if __name__ == "__main__":
    main()
