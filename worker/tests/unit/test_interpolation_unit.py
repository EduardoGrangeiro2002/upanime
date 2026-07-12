import numpy as np

from upanime_worker.interpolation import (
    FrameDeduper,
    InterpolateFrame,
    ReuseFrame,
    detect_scene_cuts,
    gray_diff_threshold,
    plan_output_frames,
)


def gray(value: float, shape=(4, 4)) -> np.ndarray:
    return np.full(shape, value, dtype=np.float32)


def test_deduper_keeps_first_frame():
    deduper = FrameDeduper()
    assert deduper.is_unique(gray(0.5)) is True


def test_deduper_drops_identical_and_keeps_changed():
    deduper = FrameDeduper()
    assert deduper.is_unique(gray(0.5)) is True
    assert deduper.is_unique(gray(0.5)) is False
    assert deduper.is_unique(gray(0.5) + 0.0001) is False
    assert deduper.is_unique(gray(0.9)) is True


def test_deduper_compares_against_last_unique_not_last_seen():
    deduper = FrameDeduper()
    threshold = gray_diff_threshold(0.996)
    step = threshold * 0.8
    assert deduper.is_unique(gray(0.1)) is True
    assert deduper.is_unique(gray(0.1 + step)) is False
    assert deduper.is_unique(gray(0.1 + 2 * step)) is True


def test_detect_scene_cuts_flags_hard_transition():
    grays = [gray(0.1), gray(0.12), gray(0.95)]
    assert detect_scene_cuts(grays) == [False, True]


def test_detect_scene_cuts_accepts_generator():
    grays = (gray(v) for v in [0.1, 0.9])
    assert detect_scene_cuts(grays) == [True]


def test_plan_preserves_duration_at_target_fps():
    fps = 24.0
    timestamps = [i / fps for i in range(24)]
    plan = plan_output_frames(timestamps, 1.0, 60.0, [False] * 23)
    assert len(plan) == 60


def test_plan_reuses_endpoints_and_interpolates_middles():
    plan = plan_output_frames([0.0, 0.5], 1.0, 4.0, [False])

    assert plan[0] == ReuseFrame(0)
    middle = plan[1]
    assert isinstance(middle, InterpolateFrame)
    assert middle.left_index == 0
    assert 0.0 < middle.timestep < 1.0
    assert plan[2] == ReuseFrame(1)


def test_plan_timestep_is_proportional_inside_gap():
    plan = plan_output_frames([0.0, 1.0], 2.0, 10.0, [False])
    interpolated = [op for op in plan if isinstance(op, InterpolateFrame)]
    timesteps = [op.timestep for op in interpolated]

    assert timesteps == sorted(timesteps)
    for expected, actual in zip([0.1 * i for i in range(1, 10) if 0.02 < 0.1 * i < 0.98], timesteps):
        assert abs(expected - actual) < 1e-9


def test_plan_never_interpolates_across_scene_cut():
    plan = plan_output_frames([0.0, 0.5], 1.0, 8.0, [True])

    assert all(isinstance(op, ReuseFrame) for op in plan)
    assert [op.index for op in plan[:2]] == [0, 0]
    assert all(op.index == 1 for op in plan[2:])


def test_plan_reuses_last_frame_for_tail_segment():
    plan = plan_output_frames([0.0], 1.0, 4.0, [])
    assert plan == [ReuseFrame(0)] * 4


def test_plan_endpoint_margin_reuses_instead_of_interpolating():
    plan = plan_output_frames([0.0, 1.0], 2.0, 100.0, [False])

    assert plan[1] == ReuseFrame(0)
    assert plan[99] == ReuseFrame(1)


def test_plan_empty_input():
    assert plan_output_frames([], 0.0, 60.0, []) == []
