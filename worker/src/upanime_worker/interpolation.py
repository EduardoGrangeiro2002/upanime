from __future__ import annotations

from dataclasses import dataclass

DEDUP_SSIM_THRESHOLD = 0.996
SCENE_CUT_SSIM_THRESHOLD = 0.4
ENDPOINT_REUSE_MARGIN = 0.02
MIN_SEGMENT_SECONDS = 1e-6


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


def plan_output_frames(
    unique_timestamps: list[float],
    total_duration: float,
    target_fps: float,
    is_scene_cut: list[bool],
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
        plan.append(_plan_single(out_ts, seg_idx, boundaries, num_unique, is_scene_cut))
    return plan


def _plan_single(
    out_ts: float,
    seg_idx: int,
    boundaries: list[float],
    num_unique: int,
    is_scene_cut: list[bool],
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
    if is_scene_cut[seg_idx]:
        return ReuseFrame(seg_idx if timestep < 0.5 else seg_idx + 1)
    return InterpolateFrame(seg_idx, timestep)
