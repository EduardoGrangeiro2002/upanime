from __future__ import annotations

import numpy as np

SMOOTH_WEIGHTS = (0.15, 0.7, 0.15)
SCENE_CUT_GRAY_DIFF = 0.3


class TemporalSmoother:
    def __init__(
        self,
        weights: tuple[float, float, float] = SMOOTH_WEIGHTS,
        scene_cut_diff: float = SCENE_CUT_GRAY_DIFF,
    ) -> None:
        self._weights = weights
        self._scene_cut_diff = scene_cut_diff
        self._frames: list[np.ndarray] = []
        self._grays: list[np.ndarray] = []

    def feed(self, frame: np.ndarray) -> list[np.ndarray]:
        self._frames.append(frame)
        self._grays.append(frame.mean(axis=2, dtype=np.float32) / 255.0)

        if len(self._frames) == 2:
            return [self._frames[0]]
        if len(self._frames) < 3:
            return []

        emitted = self._smooth_middle()
        self._frames.pop(0)
        self._grays.pop(0)
        return [emitted]

    def flush(self) -> list[np.ndarray]:
        if not self._frames:
            return []
        if len(self._frames) == 1:
            remaining = [self._frames[0]]
        else:
            remaining = [self._frames[-1]]
        self._frames = []
        self._grays = []
        return remaining

    def _smooth_middle(self) -> np.ndarray:
        prev_frame, frame, next_frame = self._frames
        prev_gray, gray, next_gray = self._grays

        if self._is_cut(prev_gray, gray) or self._is_cut(gray, next_gray):
            return frame

        w_prev, w_cur, w_next = self._weights
        blended = (
            prev_frame.astype(np.float32) * w_prev
            + frame.astype(np.float32) * w_cur
            + next_frame.astype(np.float32) * w_next
        )
        return np.clip(blended, 0, 255).astype(np.uint8)

    def _is_cut(self, gray_a: np.ndarray, gray_b: np.ndarray) -> bool:
        return float(np.abs(gray_b - gray_a).mean()) > self._scene_cut_diff
