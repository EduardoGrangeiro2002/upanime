import numpy as np

from upanime_worker.temporal import TemporalSmoother


def frame(value: int, shape=(4, 4, 3)) -> np.ndarray:
    return np.full(shape, value, dtype=np.uint8)


def run_through(smoother: TemporalSmoother, frames: list[np.ndarray]) -> list[np.ndarray]:
    out: list[np.ndarray] = []
    for item in frames:
        out.extend(smoother.feed(item))
    out.extend(smoother.flush())
    return out


def test_frame_count_is_preserved():
    frames = [frame(v) for v in [10, 20, 30, 40, 50]]
    out = run_through(TemporalSmoother(), frames)
    assert len(out) == len(frames)


def test_single_frame_passes_through():
    out = run_through(TemporalSmoother(), [frame(42)])
    assert len(out) == 1
    assert out[0][0, 0, 0] == 42


def test_two_frames_pass_through_in_order():
    out = run_through(TemporalSmoother(), [frame(10), frame(200)])
    assert [f[0, 0, 0] for f in out] == [10, 200]


def test_constant_stream_is_unchanged():
    frames = [frame(100) for _ in range(5)]
    out = run_through(TemporalSmoother(), frames)
    assert all(int(f[0, 0, 0]) == 100 for f in out)


def test_middle_frame_is_blended_with_neighbors():
    out = run_through(TemporalSmoother(), [frame(0), frame(60), frame(0)])

    expected = round(0.15 * 0 + 0.7 * 60 + 0.15 * 0)
    assert int(out[1][0, 0, 0]) == expected
    assert int(out[0][0, 0, 0]) == 0
    assert int(out[2][0, 0, 0]) == 0


def test_scene_cut_is_not_blended():
    dark = frame(10)
    bright = frame(240)
    out = run_through(TemporalSmoother(), [dark, dark, bright, bright, bright])

    assert int(out[1][0, 0, 0]) == 10
    assert int(out[2][0, 0, 0]) == 240


def test_boundary_frames_are_never_blended():
    frames = [frame(v) for v in [50, 60, 70, 80]]
    out = run_through(TemporalSmoother(), frames)

    assert int(out[0][0, 0, 0]) == 50
    assert int(out[-1][0, 0, 0]) == 80
