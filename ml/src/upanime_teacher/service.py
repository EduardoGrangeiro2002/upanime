from __future__ import annotations

import logging
import random
from pathlib import Path

from .config import Settings
from .frames import sample_candidates
from .sink import SampleSink, TeacherSample


class TeacherIngestService:
    def __init__(
        self,
        teacher: object,
        tagger: object,
        sink: SampleSink,
        settings: Settings,
        rng: random.Random | None = None,
    ) -> None:
        self._teacher = teacher
        self._tagger = tagger
        self._sink = sink
        self._settings = settings
        self._rng = rng or random.Random()

    def run(
        self,
        video_path: Path,
        anime_title: str,
        episode: str,
        manual_timestamps: tuple[float, ...] = (),
    ) -> dict:
        stats = {"candidates": 0, "sent": 0, "negatives": 0, "by_class": {}}

        candidates = sample_candidates(
            video_path,
            self._tagger,
            self._settings.sample_fps,
            self._settings.wd14_threshold,
            self._settings.random_keep,
            self._rng,
            manual_timestamps,
        )

        for candidate in candidates:
            if stats["sent"] >= self._settings.max_samples:
                logging.warning("max_samples reached (%d) — stopping early", self._settings.max_samples)
                break
            stats["candidates"] += 1

            proposals = self._teacher.propose(candidate.frame_bgr)
            if not proposals:
                self._maybe_send_negative(candidate, anime_title, episode, stats)
                continue

            for proposal in proposals:
                self._sink.send(TeacherSample(
                    class_name=proposal.class_name,
                    frame_bgr=candidate.frame_bgr,
                    mask=proposal.mask,
                    anime_title=anime_title,
                    episode=episode,
                    timestamp_s=candidate.timestamp_s,
                    teacher_prob=proposal.score,
                    source=proposal.origin,
                ))
                stats["sent"] += 1
                stats["by_class"][proposal.class_name] = stats["by_class"].get(proposal.class_name, 0) + 1

            if stats["candidates"] % 25 == 0:
                logging.info("progress: %d candidates, %d sent", stats["candidates"], stats["sent"])

        return stats

    def _maybe_send_negative(self, candidate: object, anime_title: str, episode: str, stats: dict) -> None:
        if self._rng.random() >= self._settings.negative_keep:
            return
        self._sink.send(TeacherSample(
            class_name="none",
            frame_bgr=candidate.frame_bgr,
            mask=None,
            anime_title=anime_title,
            episode=episode,
            timestamp_s=candidate.timestamp_s,
            teacher_prob=candidate.wd14_prob,
            source="sampler",
        ))
        stats["sent"] += 1
        stats["negatives"] += 1
        stats["by_class"]["none"] = stats["by_class"].get("none", 0) + 1
