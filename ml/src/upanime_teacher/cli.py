from __future__ import annotations

import argparse
import json
import logging
import sys
import tempfile
from pathlib import Path

import requests

from .config import load_settings, resolve_device
from .service import TeacherIngestService
from .sink import TriageAPISink
from .tagger import EffectTagger
from .teacher import ComposedTeacher

logging.basicConfig(level=logging.INFO)


def download_video(url: str, timeout_seconds: int) -> Path:
    destination = Path(tempfile.mkdtemp()) / "episode.mp4"
    logging.info("downloading %s", url.split("?")[0])
    response = requests.get(url, stream=True, timeout=timeout_seconds)
    response.raise_for_status()
    with destination.open("wb") as handle:
        for chunk in response.iter_content(chunk_size=1024 * 1024):
            if not chunk:
                continue
            handle.write(chunk)
    return destination


def parse_timestamps(raw: str) -> tuple[float, ...]:
    if not raw:
        return ()
    return tuple(float(part) for part in raw.split(",") if part.strip())


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="upanime-teacher")
    parser.add_argument("video", help="caminho local ou URL http(s) do episódio")
    parser.add_argument("--anime", required=True)
    parser.add_argument("--episode", required=True)
    parser.add_argument("--timestamps", default="", help="timestamps manuais em segundos, separados por vírgula")
    args = parser.parse_args(argv)

    settings = load_settings()
    device = resolve_device(settings.device)
    logging.info("device: %s", device)

    video_path = Path(args.video)
    if args.video.startswith(("http://", "https://")):
        video_path = download_video(args.video, settings.request_timeout_seconds)

    if not video_path.exists():
        logging.error("video not found: %s", video_path)
        return 1

    service = TeacherIngestService(
        teacher=ComposedTeacher(device, settings.dino_threshold),
        tagger=EffectTagger(settings.tagger_model_path, settings.tagger_tags_path),
        sink=TriageAPISink(settings.api_base, settings.api_token, settings.request_timeout_seconds),
        settings=settings,
    )
    stats = service.run(video_path, args.anime, args.episode, parse_timestamps(args.timestamps))
    print(json.dumps(stats, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
