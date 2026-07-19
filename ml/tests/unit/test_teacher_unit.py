from __future__ import annotations

import io
import random
from pathlib import Path

import cv2
import numpy as np
import pytest
from PIL import Image

from upanime_teacher.config import Settings
from upanime_teacher.frames import sample_candidates
from upanime_teacher.photometric import bright_points, seed_mask
from upanime_teacher.sink import mask_to_png_bytes
from upanime_teacher.teacher import Proposal, class_for, mask_iou


def make_settings(**overrides) -> Settings:
    base = dict(
        api_base="http://127.0.0.1:9",
        api_token="token",
        device="cpu",
        sample_fps=4.0,
        wd14_threshold=0.05,
        random_keep=0.0,
        negative_keep=1.0,
        max_samples=100,
        dino_threshold=0.2,
        request_timeout_seconds=5,
        tagger_model_path=Path("/nonexistent/model.onnx"),
        tagger_tags_path=Path("/nonexistent/tags.csv"),
    )
    base.update(overrides)
    return Settings(**base)


class StubTagger:
    def __init__(self, probs: list[float]) -> None:
        self._probs = probs
        self.calls = 0

    def available(self) -> bool:
        return True

    def effect_prob(self, frame: object) -> float:
        prob = self._probs[min(self.calls, len(self._probs) - 1)]
        self.calls += 1
        return prob


def bright_blob_frame() -> np.ndarray:
    frame = np.full((96, 128, 3), 40, dtype=np.uint8)
    frame[30:60, 50:90] = (30, 200, 250)
    return frame


def test_seed_mask_finds_bright_saturated_blob():
    mask = seed_mask(bright_blob_frame())
    assert mask[45, 70] == 1
    assert mask[10, 10] == 0


def test_bright_points_returns_blob_centroid():
    points = bright_points(bright_blob_frame())
    assert len(points) == 1
    x, y = points[0]
    assert 60 <= x <= 80
    assert 38 <= y <= 52


def test_bright_points_empty_on_dark_frame():
    frame = np.full((96, 128, 3), 30, dtype=np.uint8)
    assert bright_points(frame) == []


@pytest.mark.parametrize("label,expected", [
    ("fire", "fire"),
    ("burning fire", "fire"),
    ("explosion", "fire"),
    ("lightning bolt", "lightning"),
    ("energy beam", "beam"),
    ("glowing orb", "energy"),
    ("glowing energy", "energy"),
    ("magic aura", "aura"),
    ("magic spell glow", "aura"),
    ("", None),
    ("   ", None),
    ("sword", None),
])
def test_class_for(label, expected):
    assert class_for(label) == expected


def test_mask_iou():
    a = np.zeros((10, 10), dtype=bool)
    b = np.zeros((10, 10), dtype=bool)
    a[:5] = True
    b[:5] = True
    assert mask_iou(a, b) == 1.0
    b[:] = False
    b[5:] = True
    assert mask_iou(a, b) == 0.0
    assert mask_iou(np.zeros((4, 4), bool), np.zeros((4, 4), bool)) == 0.0


def test_mask_to_png_bytes_encodes_alpha():
    mask = np.zeros((8, 8), dtype=bool)
    mask[2:4, 2:4] = True

    png = mask_to_png_bytes(mask, (8, 8))
    decoded = np.array(Image.open(io.BytesIO(png)))

    assert decoded.shape == (8, 8, 4)
    assert decoded[3, 3, 3] == 255
    assert decoded[0, 0, 3] == 0


def test_mask_to_png_bytes_none_is_fully_transparent():
    png = mask_to_png_bytes(None, (4, 4))
    decoded = np.array(Image.open(io.BytesIO(png)))
    assert decoded[:, :, 3].max() == 0


def write_video(path: Path, frame_count: int) -> None:
    writer = cv2.VideoWriter(str(path), cv2.VideoWriter_fourcc(*"mp4v"), 8.0, (64, 48))
    for i in range(frame_count):
        frame = np.full((48, 64, 3), 30 + i, dtype=np.uint8)
        writer.write(frame)
    writer.release()


def test_sample_candidates_origins_and_rate(tmp_path):
    video = tmp_path / "clip.mp4"
    write_video(video, 32)
    tagger = StubTagger([0.9, 0.01, 0.9, 0.01] * 8)

    candidates = list(sample_candidates(
        video, tagger, sample_fps=4.0, wd14_threshold=0.05,
        random_keep=1.0, rng=random.Random(7),
    ))

    origins = [c.origin for c in candidates]
    assert origins == ["wd14", "random", "wd14", "random"] * 4
    assert candidates[0].wd14_prob == 0.9
    assert candidates[1].wd14_prob == 0.01


def test_sample_candidates_manual_timestamps(tmp_path):
    video = tmp_path / "clip.mp4"
    write_video(video, 32)
    tagger = StubTagger([0.0])

    candidates = list(sample_candidates(
        video, tagger, sample_fps=4.0, wd14_threshold=0.5,
        random_keep=0.0, rng=random.Random(1), manual_timestamps=(1.5,),
    ))

    assert len(candidates) == 1
    assert candidates[0].origin == "manual"
    assert candidates[0].timestamp_s == 1.5
