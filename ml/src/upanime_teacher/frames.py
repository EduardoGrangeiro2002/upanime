from __future__ import annotations

import random
from dataclasses import dataclass
from pathlib import Path
from typing import Iterator

import cv2
import numpy as np


@dataclass
class FrameCandidate:
    frame_bgr: np.ndarray
    timestamp_s: float
    wd14_prob: float
    origin: str


def sample_candidates(
    video_path: Path,
    tagger: object,
    sample_fps: float,
    wd14_threshold: float,
    random_keep: float,
    rng: random.Random,
    manual_timestamps: tuple[float, ...] = (),
) -> Iterator[FrameCandidate]:
    capture = cv2.VideoCapture(str(video_path))
    if not capture.isOpened():
        raise RuntimeError(f"could not open video: {video_path}")

    fps = capture.get(cv2.CAP_PROP_FPS) or 24.0
    step = max(1, round(fps / sample_fps))

    try:
        index = 0
        while True:
            grabbed = capture.grab()
            if not grabbed:
                break
            if index % step != 0:
                index += 1
                continue
            ok, frame = capture.retrieve()
            if not ok:
                break
            timestamp = index / fps
            prob = tagger.effect_prob(frame)
            if prob >= wd14_threshold:
                yield FrameCandidate(frame, timestamp, prob, "wd14")
            elif rng.random() < random_keep:
                yield FrameCandidate(frame, timestamp, prob, "random")
            index += 1

        for timestamp in manual_timestamps:
            capture.set(cv2.CAP_PROP_POS_MSEC, timestamp * 1000)
            ok, frame = capture.read()
            if not ok:
                continue
            yield FrameCandidate(frame, timestamp, tagger.effect_prob(frame), "manual")
    finally:
        capture.release()
