import numpy as np

from upanime_worker.interpolation import (
    FrameDeduper,
    InterpolateFrame,
    ReuseFrame,
    classify_gaps,
    collapse_plan,
    detect_scene_cuts,
    estimate_shift,
    gap_is_pan,
    gray_diff_threshold,
    plan_output_frames,
)


def gray(value: float, shape=(4, 4)) -> np.ndarray:
    return np.full(shape, value, dtype=np.float32)


def gradient(shift: int = 0, shape=(48, 80)) -> np.ndarray:
    base = np.tile(np.linspace(0.0, 1.0, shape[1], dtype=np.float32), (shape[0], 1))
    noise = np.sin(np.arange(shape[0], dtype=np.float32) * 1.7)[:, None] * 0.2
    return np.roll(base + noise, shift, axis=1)


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


def test_estimate_shift_recovers_translation():
    a = gradient()
    b = np.roll(a, (0, 7), axis=(0, 1))
    dy, dx = estimate_shift(a, b)
    assert (abs(dy), abs(dx)) == (0, 7)


def test_gap_is_pan_accepts_translation():
    a = gradient()
    b = np.roll(a, (2, 6), axis=(0, 1))
    assert gap_is_pan(a, b) is True


def test_gap_is_pan_rejects_static_frames():
    a = gradient()
    assert gap_is_pan(a, a.copy()) is False


def test_gap_is_pan_rejects_local_change():
    rng = np.random.default_rng(7)
    a = rng.random((48, 80), dtype=np.float32)
    b = a.copy()
    b[10:35, 20:60] = rng.random((25, 40), dtype=np.float32)
    assert gap_is_pan(a, b) is False


def test_classify_gaps_blocks_scene_cuts_and_flashes():
    pan_a = gradient()
    pan_b = np.roll(pan_a, (0, 5), axis=(0, 1))
    flash = np.clip(pan_b + 0.5, 0.0, 1.5)
    grays = [pan_a, pan_b, flash]
    allow = classify_gaps(grays, [False, False])
    assert allow[0] is True
    assert allow[1] is False


def test_classify_gaps_allows_full_rate_pan():
    frames = [np.roll(gradient(), (0, i * 3), axis=(0, 1)) for i in range(10)]
    allow = classify_gaps(frames, [False] * 9)
    assert allow == [True] * 9


def test_classify_gaps_blocks_marked_scene_cut():
    frames = [np.roll(gradient(), (0, i * 4), axis=(0, 1)) for i in range(3)]
    allow = classify_gaps(frames, [False, True])
    assert allow == [True, False]


def test_plan_preserves_duration_at_target_fps():
    fps = 24.0
    timestamps = [i / fps for i in range(24)]
    plan = plan_output_frames(timestamps, 1.0, 60.0, [True] * 23)
    assert len(plan) == 60


def test_plan_reuses_endpoints_and_interpolates_middles():
    plan = plan_output_frames([0.0, 0.5], 1.0, 4.0, [True])

    assert plan[0] == ReuseFrame(0)
    middle = plan[1]
    assert isinstance(middle, InterpolateFrame)
    assert middle.left_index == 0
    assert 0.0 < middle.timestep < 1.0
    assert plan[2] == ReuseFrame(1)


def test_plan_timestep_is_proportional_inside_gap():
    plan = plan_output_frames([0.0, 1.0], 2.0, 10.0, [True])
    interpolated = [op for op in plan if isinstance(op, InterpolateFrame)]
    timesteps = [op.timestep for op in interpolated]

    assert timesteps == sorted(timesteps)
    for expected, actual in zip([0.1 * i for i in range(1, 10) if 0.02 < 0.1 * i < 0.98], timesteps):
        assert abs(expected - actual) < 1e-9


def test_plan_never_interpolates_blocked_gap():
    plan = plan_output_frames([0.0, 0.5], 1.0, 8.0, [False])

    assert all(isinstance(op, ReuseFrame) for op in plan)
    assert [op.index for op in plan[:2]] == [0, 0]
    assert all(op.index == 1 for op in plan[2:])


def test_plan_reuses_last_frame_for_tail_segment():
    plan = plan_output_frames([0.0], 1.0, 4.0, [])
    assert plan == [ReuseFrame(0)] * 4


def test_plan_endpoint_margin_reuses_instead_of_interpolating():
    plan = plan_output_frames([0.0, 1.0], 2.0, 100.0, [True])

    assert plan[1] == ReuseFrame(0)
    assert plan[99] == ReuseFrame(1)


def test_plan_empty_input():
    assert plan_output_frames([], 0.0, 60.0, []) == []


def test_collapse_plan_groups_consecutive_reuses():
    plan = [
        ReuseFrame(0),
        ReuseFrame(0),
        ReuseFrame(0),
        InterpolateFrame(0, 0.5),
        ReuseFrame(1),
        ReuseFrame(1),
    ]
    assert collapse_plan(plan) == [
        (ReuseFrame(0), 3),
        (InterpolateFrame(0, 0.5), 1),
        (ReuseFrame(1), 2),
    ]
