from __future__ import annotations

import csv
import logging
from pathlib import Path

EFFECT_TAGS = {
    "fire",
    "explosion",
    "energy",
    "electricity",
    "lightning",
    "magic",
    "energy_ball",
    "energy_beam",
    "aura",
    "magic_circle",
}
TAGGER_INPUT_SIZE = 448


class EffectTagger:
    def __init__(self, model_path: Path, tags_path: Path) -> None:
        self._model_path = model_path
        self._tags_path = tags_path
        self._session = None
        self._effect_indices: list[int] = []
        self._available = model_path.exists() and tags_path.exists()
        if not self._available:
            logging.warning("wd14 tagger unavailable — every sampled frame becomes a candidate")

    def available(self) -> bool:
        return self._available

    def _ensure_session(self) -> None:
        if self._session is not None:
            return
        import onnxruntime

        self._session = onnxruntime.InferenceSession(
            str(self._model_path), providers=["CPUExecutionProvider"]
        )
        with self._tags_path.open() as handle:
            rows = list(csv.DictReader(handle))
        self._effect_indices = [i for i, row in enumerate(rows) if row["name"] in EFFECT_TAGS]

    def effect_prob(self, frame_bgr: object) -> float:
        if not self._available:
            return 1.0
        self._ensure_session()

        import cv2
        import numpy as np

        height, width = frame_bgr.shape[:2]
        side = max(height, width)
        canvas = np.full((side, side, 3), 255, dtype=np.uint8)
        top = (side - height) // 2
        left = (side - width) // 2
        canvas[top : top + height, left : left + width] = frame_bgr
        resized = cv2.resize(canvas, (TAGGER_INPUT_SIZE, TAGGER_INPUT_SIZE), interpolation=cv2.INTER_AREA)
        batch = resized.astype(np.float32)[None]

        input_name = self._session.get_inputs()[0].name
        probs = self._session.run(None, {input_name: batch})[0][0]
        return max(float(probs[i]) for i in self._effect_indices)
