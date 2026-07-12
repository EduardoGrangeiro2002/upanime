from __future__ import annotations

import csv
import logging
import threading
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
DEFAULT_THRESHOLD = 0.35


class EffectTagger:
    def __init__(self, model_path: Path, tags_path: Path, threshold: float = DEFAULT_THRESHOLD) -> None:
        self._model_path = model_path
        self._tags_path = tags_path
        self._threshold = threshold
        self._session = None
        self._effect_indices: list[int] = []
        self._effect_names: list[str] = []
        self._lock = threading.Lock()
        self._available = model_path.exists() and tags_path.exists()
        if not self._available:
            logging.warning("effect tagger unavailable (model or tags missing) — comp gate passes all shots")

    def available(self) -> bool:
        return self._available

    def _ensure_session(self) -> None:
        with self._lock:
            if self._session is not None:
                return
            import onnxruntime

            self._session = onnxruntime.InferenceSession(
                str(self._model_path), providers=["CPUExecutionProvider"]
            )
            with self._tags_path.open() as handle:
                rows = list(csv.DictReader(handle))
            for index, row in enumerate(rows):
                if row["name"] in EFFECT_TAGS:
                    self._effect_indices.append(index)
                    self._effect_names.append(row["name"])
            logging.info("effect tagger loaded: %d effect tags mapped", len(self._effect_indices))

    def shot_effect_tags(self, frame_bgr: object) -> list[str]:
        if not self._available:
            return ["gate-off"]
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

        found = []
        for index, name in zip(self._effect_indices, self._effect_names):
            if float(probs[index]) >= self._threshold:
                found.append(name)
        return found

    def shot_has_effect(self, frame_bgr: object) -> bool:
        return len(self.shot_effect_tags(frame_bgr)) > 0
