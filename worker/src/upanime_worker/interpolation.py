from __future__ import annotations

from dataclasses import dataclass

import numpy as np

DEDUP_SSIM_THRESHOLD = 0.996
SCENE_CUT_SSIM_THRESHOLD = 0.4
ENDPOINT_REUSE_MARGIN = 0.02
MIN_SEGMENT_SECONDS = 1e-6
FLASH_LUMA_DELTA = 0.10
PAN_RESIDUAL_RATIO = 0.5
PAN_MIN_RAW_DIFF = 0.002


def gray_diff_threshold(ssim_threshold: float) -> float:
    return (1.0 - ssim_threshold) * 0.5


@dataclass(frozen=True)
class ReuseFrame:
    index: int


@dataclass(frozen=True)
class InterpolateFrame:
    left_index: int
    timestep: float


class FrameDeduper:
    def __init__(self, ssim_threshold: float = DEDUP_SSIM_THRESHOLD) -> None:
        self._threshold = gray_diff_threshold(ssim_threshold)
        self._last_gray = None

    def is_unique(self, gray) -> bool:
        if self._last_gray is None:
            self._last_gray = gray
            return True
        diff = float(abs(gray - self._last_gray).mean())
        if diff <= self._threshold:
            return False
        self._last_gray = gray
        return True


def detect_scene_cuts(grays, ssim_threshold: float = SCENE_CUT_SSIM_THRESHOLD) -> list[bool]:
    threshold = gray_diff_threshold(ssim_threshold)
    cuts = []
    previous = None
    for gray in grays:
        if previous is not None:
            diff = float(abs(gray - previous).mean())
            cuts.append(diff > threshold)
        previous = gray
    return cuts


def estimate_shift(a: np.ndarray, b: np.ndarray) -> tuple[int, int]:
    fa = np.fft.rfft2(a)
    fb = np.fft.rfft2(b)
    cross = fa * np.conj(fb)
    cross = cross / np.maximum(np.abs(cross), 1e-9)
    corr = np.fft.irfft2(cross, a.shape)
    dy, dx = np.unravel_index(int(np.argmax(corr)), corr.shape)
    if dy > a.shape[0] // 2:
        dy -= a.shape[0]
    if dx > a.shape[1] // 2:
        dx -= a.shape[1]
    return int(dy), int(dx)


def _cropped_diff(a: np.ndarray, b: np.ndarray, margin_y: int, margin_x: int) -> float:
    h, w = a.shape
    a_crop = a[margin_y : h - margin_y, margin_x : w - margin_x]
    b_crop = b[margin_y : h - margin_y, margin_x : w - margin_x]
    if a_crop.size == 0:
        return float(np.abs(b - a).mean())
    return float(np.abs(b_crop - a_crop).mean())


def gap_is_pan(a: np.ndarray, b: np.ndarray, residual_ratio: float = PAN_RESIDUAL_RATIO) -> bool:
    raw = float(np.abs(b - a).mean())
    if raw < PAN_MIN_RAW_DIFF:
        return False
    dy, dx = estimate_shift(a, b)
    if dy == 0 and dx == 0:
        return False
    if abs(dy) > a.shape[0] // 4 or abs(dx) > a.shape[1] // 4:
        return False
    margin_y = max(abs(dy), 4)
    margin_x = max(abs(dx), 4)
    raw_cropped = _cropped_diff(a, b, margin_y, margin_x)
    best = raw_cropped
    for sy, sx in ((dy, dx), (-dy, -dx)):
        shifted = np.roll(a, (sy, sx), axis=(0, 1))
        best = min(best, _cropped_diff(shifted, b, margin_y, margin_x))
    return best <= raw_cropped * residual_ratio


def classify_gaps(
    grays: list[np.ndarray],
    is_scene_cut: list[bool],
    pan_residual_ratio: float = PAN_RESIDUAL_RATIO,
) -> list[bool]:
    allow = []
    for gap in range(len(is_scene_cut)):
        if is_scene_cut[gap]:
            allow.append(False)
            continue
        a, b = grays[gap], grays[gap + 1]
        if abs(float(b.mean()) - float(a.mean())) > FLASH_LUMA_DELTA:
            allow.append(False)
            continue
        allow.append(gap_is_pan(a, b, pan_residual_ratio))
    return allow


def plan_output_frames(
    unique_timestamps: list[float],
    total_duration: float,
    target_fps: float,
    allow_interpolation: list[bool],
) -> list[ReuseFrame | InterpolateFrame]:
    num_unique = len(unique_timestamps)
    if num_unique == 0:
        return []

    boundaries = list(unique_timestamps) + [total_duration]
    total_target_frames = int(round(total_duration * target_fps))
    target_frame_duration = 1.0 / target_fps

    plan: list[ReuseFrame | InterpolateFrame] = []
    seg_idx = 0
    for frame_idx in range(total_target_frames):
        out_ts = frame_idx * target_frame_duration
        while seg_idx < num_unique - 1 and boundaries[seg_idx + 1] <= out_ts:
            seg_idx += 1
        plan.append(_plan_single(out_ts, seg_idx, boundaries, num_unique, allow_interpolation))
    return plan


def _plan_single(
    out_ts: float,
    seg_idx: int,
    boundaries: list[float],
    num_unique: int,
    allow_interpolation: list[bool],
) -> ReuseFrame | InterpolateFrame:
    if seg_idx >= num_unique - 1:
        return ReuseFrame(num_unique - 1)

    seg_start = boundaries[seg_idx]
    seg_end = boundaries[seg_idx + 1]
    seg_len = seg_end - seg_start
    if seg_len < MIN_SEGMENT_SECONDS:
        return ReuseFrame(seg_idx)

    timestep = (out_ts - seg_start) / seg_len
    if timestep < ENDPOINT_REUSE_MARGIN:
        return ReuseFrame(seg_idx)
    if timestep > 1.0 - ENDPOINT_REUSE_MARGIN:
        return ReuseFrame(seg_idx + 1)
    if not allow_interpolation[seg_idx]:
        return ReuseFrame(seg_idx if timestep < 0.5 else seg_idx + 1)
    return InterpolateFrame(seg_idx, timestep)


def collapse_plan(
    plan: list[ReuseFrame | InterpolateFrame],
) -> list[tuple[ReuseFrame | InterpolateFrame, int]]:
    collapsed: list[tuple[ReuseFrame | InterpolateFrame, int]] = []
    for op in plan:
        if collapsed and isinstance(op, ReuseFrame) and collapsed[-1][0] == op:
            collapsed[-1] = (op, collapsed[-1][1] + 1)
            continue
        collapsed.append((op, 1))
    return collapsed
