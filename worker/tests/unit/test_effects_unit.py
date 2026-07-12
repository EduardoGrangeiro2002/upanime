import torch

from upanime_worker.effects import MASK_MIN_AREA, EffectsComp


def make_comp(strength: float = 1.0, sensitivity: float = 1.0) -> EffectsComp:
    return EffectsComp(torch, torch.device("cpu"), strength=strength, sensitivity=sensitivity)


def dark_frame(h: int = 96, w: int = 128) -> torch.Tensor:
    return torch.full((1, 3, h, w), 0.15)


def fire_frame(h: int = 96, w: int = 128) -> torch.Tensor:
    frame = dark_frame(h, w)
    frame[:, 2, 30:66, 40:88] = 0.95
    frame[:, 1, 30:66, 40:88] = 0.75
    frame[:, 0, 30:66, 40:88] = 0.15
    frame[:, :, 42:54, 56:72] = 0.97
    return frame


def test_frame_without_effect_passes_untouched():
    comp = make_comp()
    frame = dark_frame()
    out = comp.process(frame.clone())
    assert torch.equal(out, frame)


def test_fire_region_gets_brighter_and_background_stays():
    comp = make_comp()
    frame = fire_frame()
    for _ in range(3):
        out = comp.process(frame.clone())
    fire_before = frame[:, :, 42:54, 56:72].mean()
    fire_after = out[:, :, 42:54, 56:72].mean()
    corner_before = frame[:, :, :20, :20].mean()
    corner_after = out[:, :, :20, :20].mean()
    assert fire_after >= fire_before
    assert abs(float(corner_after - corner_before)) < 0.05


def test_mask_area_detects_fire():
    comp = make_comp()
    comp.process(fire_frame())
    assert float(comp._mask_ema.mean()) >= MASK_MIN_AREA


def test_reset_clears_state():
    comp = make_comp()
    comp.process(fire_frame())
    comp.reset()
    assert comp._mask_ema is None
    assert comp._prev_area == 0.0


def test_strength_zero_keeps_frame_close_to_original():
    comp = make_comp(strength=0.0)
    frame = fire_frame()
    out = comp.process(frame.clone())
    diff = float((out - frame).abs().mean())
    assert diff < 0.02


def test_output_stays_in_range():
    comp = make_comp(strength=1.5)
    frame = fire_frame()
    for _ in range(5):
        out = comp.process(frame.clone())
    assert float(out.min()) >= 0.0
    assert float(out.max()) <= 1.0
