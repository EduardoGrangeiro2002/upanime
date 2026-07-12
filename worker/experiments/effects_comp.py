from __future__ import annotations

import argparse
from collections import deque
from pathlib import Path

import cv2
import numpy as np

MASK_V_MIN = 0.78
MASK_S_MIN = 0.6
MASK_HOT_V = 0.93
MASK_HOT_NEAR_SAT = 15
MASK_OPEN_KERNEL = 7
GROW_S_MIN = 0.55
GROW_V_MIN = 0.45
GROW_ITERATIONS = 16
GROW_WEIGHT = 0.65
SCENE_CUT_DIFF = 0.3
MASK_MIN_AREA = 0.004
MASK_EMA = 0.55
BLOOM_SIGMAS = (6, 18, 48)
BLOOM_WEIGHTS = (0.45, 0.35, 0.25)
EFFECT_SATURATION = 1.35
EFFECT_GAIN = 1.08
CORE_V = 0.95
LIGHT_WRAP_SIGMA = 61
LIGHT_WRAP_GAIN = 0.4
PARTICLES_PER_FRAME = 45
PARTICLE_LIFE = 10
PARTICLE_SPAWN_MIN_AREA = 0.012
PARTICLE_SPAWN_CORE = 0.8
PARTICLE_STARVE_FRAMES = 6
PULSE_WINDOW = 8
PULSE_MIN_VARIATION = 0.15
PARTICLE_DRIFT = -1.4
FLOW_SCALE = 0.25
HEAT_AMPLITUDE = 2.4
SHAKE_TRIGGER = 0.02
SHAKE_PIXELS = 7.0
SHAKE_DECAY = 0.8


def effect_mask(frame: np.ndarray) -> np.ndarray:
    hsv = cv2.cvtColor(frame, cv2.COLOR_BGR2HSV)
    s, v = hsv[..., 1], hsv[..., 2]
    saturated = ((v > MASK_V_MIN) & (s > MASK_S_MIN)).astype(np.float32)
    near_saturated = cv2.dilate(saturated, np.ones((MASK_HOT_NEAR_SAT, MASK_HOT_NEAR_SAT), np.uint8))
    hot = ((v > MASK_HOT_V).astype(np.float32)) * near_saturated
    seed = np.clip(saturated + hot, 0, 1)
    kernel = np.ones((MASK_OPEN_KERNEL, MASK_OPEN_KERNEL), np.uint8)
    seed = cv2.morphologyEx(seed, cv2.MORPH_OPEN, kernel)

    weak = ((s > GROW_S_MIN) & (v > GROW_V_MIN)).astype(np.float32)
    grown = seed.copy()
    grow_kernel = np.ones((5, 5), np.uint8)
    for _ in range(GROW_ITERATIONS):
        expanded = cv2.dilate(grown, grow_kernel) * weak
        if np.array_equal(expanded, grown):
            break
        grown = expanded

    raw = np.clip(seed + (grown - seed) * GROW_WEIGHT, 0, 1)
    raw = cv2.dilate(raw, np.ones((5, 5), np.uint8))
    return cv2.GaussianBlur(raw, (0, 0), 4)


def regrade_effect(frame: np.ndarray, mask: np.ndarray) -> np.ndarray:
    hsv = cv2.cvtColor(frame, cv2.COLOR_BGR2HSV)
    hsv[..., 1] = np.clip(hsv[..., 1] * EFFECT_SATURATION, 0, 1)
    hsv[..., 2] = np.clip(hsv[..., 2] * EFFECT_GAIN, 0, 1)
    graded = cv2.cvtColor(hsv, cv2.COLOR_HSV2BGR)
    core = np.clip((hsv[..., 2] - CORE_V) / (1.0 - CORE_V), 0, 1)[..., None] * 0.6
    graded = graded * (1 - core) + core
    m = mask[..., None]
    return frame * (1 - m) + graded * m


def apply_bloom(frame: np.ndarray, mask: np.ndarray) -> np.ndarray:
    bright = frame * mask[..., None]
    bloom = np.zeros_like(frame)
    for sigma, weight in zip(BLOOM_SIGMAS, BLOOM_WEIGHTS):
        bloom += cv2.GaussianBlur(bright, (0, 0), sigma) * weight
    return 1.0 - (1.0 - frame) * (1.0 - np.clip(bloom, 0, 1))


def light_wrap(frame: np.ndarray, mask: np.ndarray) -> np.ndarray:
    spill = cv2.GaussianBlur(frame * mask[..., None], (0, 0), LIGHT_WRAP_SIGMA)
    return np.clip(frame + spill * (1 - mask[..., None]) * LIGHT_WRAP_GAIN, 0, 1)


def heat_distortion(frame: np.ndarray, mask: np.ndarray, t: int) -> np.ndarray:
    height, width = mask.shape
    strength = cv2.GaussianBlur(mask, (0, 0), 9) * HEAT_AMPLITUDE
    ys, xs = np.mgrid[0:height, 0:width].astype(np.float32)
    ripple = np.sin(ys / 6.5 + t * 1.1) * strength
    return cv2.remap(frame, xs + ripple, ys, cv2.INTER_LINEAR, borderMode=cv2.BORDER_REFLECT)


class ParticleField:
    def __init__(self, rng: np.random.Generator) -> None:
        self.rng = rng
        self.pos = np.zeros((0, 2), dtype=np.float32)
        self.vel = np.zeros((0, 2), dtype=np.float32)
        self.life = np.zeros(0, dtype=np.float32)
        self.color = np.zeros((0, 3), dtype=np.float32)

    def clear(self) -> None:
        self.pos = np.zeros((0, 2), dtype=np.float32)
        self.vel = np.zeros((0, 2), dtype=np.float32)
        self.life = np.zeros(0, dtype=np.float32)
        self.color = np.zeros((0, 3), dtype=np.float32)

    def spawn(self, frame: np.ndarray, mask: np.ndarray) -> None:
        core = (mask > PARTICLE_SPAWN_CORE).astype(np.float32)
        edge = np.clip(core - cv2.erode(core, np.ones((9, 9), np.uint8)), 0, 1)
        candidates = np.argwhere(edge > 0.5)
        if len(candidates) == 0:
            return
        picks = candidates[self.rng.integers(0, len(candidates), PARTICLES_PER_FRAME)]
        pos = picks[:, ::-1].astype(np.float32)
        vel = self.rng.normal(0, 0.8, (len(picks), 2)).astype(np.float32)
        vel[:, 1] += PARTICLE_DRIFT
        colors = frame[picks[:, 0], picks[:, 1]] * 1.2
        self.pos = np.vstack([self.pos, pos])
        self.vel = np.vstack([self.vel, vel])
        self.life = np.concatenate([self.life, np.full(len(picks), float(PARTICLE_LIFE))])
        self.color = np.vstack([self.color, np.clip(colors, 0, 1)])

    def step(self, flow: np.ndarray, shape: tuple[int, int]) -> None:
        if len(self.pos) == 0:
            return
        height, width = shape
        fx = np.clip((self.pos[:, 0] * FLOW_SCALE).astype(int), 0, flow.shape[1] - 1)
        fy = np.clip((self.pos[:, 1] * FLOW_SCALE).astype(int), 0, flow.shape[0] - 1)
        self.pos += self.vel + flow[fy, fx] / FLOW_SCALE * 0.5
        self.life -= 1
        keep = (
            (self.life > 0)
            & (self.pos[:, 0] > 1) & (self.pos[:, 0] < width - 2)
            & (self.pos[:, 1] > 1) & (self.pos[:, 1] < height - 2)
        )
        self.pos, self.vel = self.pos[keep], self.vel[keep]
        self.life, self.color = self.life[keep], self.color[keep]

    def render(self, frame: np.ndarray) -> np.ndarray:
        if len(self.pos) == 0:
            return frame
        overlay = np.zeros_like(frame)
        fade = (self.life / PARTICLE_LIFE)[:, None]
        for (x, y), color in zip(self.pos.astype(int), self.color * fade):
            cv2.circle(overlay, (int(x), int(y)), 1, color.tolist(), -1)
        overlay = cv2.GaussianBlur(overlay, (0, 0), 1.2)
        return np.clip(frame + overlay, 0, 1)


def effect_is_pulsing(core_areas: deque[float]) -> bool:
    if len(core_areas) < PULSE_WINDOW:
        return False
    peak = max(core_areas)
    if peak <= 0:
        return False
    return (peak - min(core_areas)) / peak > PULSE_MIN_VARIATION


def label(frame: np.ndarray, text: str) -> np.ndarray:
    out = (frame * 255).astype(np.uint8)
    cv2.putText(out, text, (12, 28), cv2.FONT_HERSHEY_SIMPLEX, 0.8, (0, 0, 0), 4)
    cv2.putText(out, text, (12, 28), cv2.FONT_HERSHEY_SIMPLEX, 0.8, (255, 255, 255), 1)
    return out


def process(input_path: Path, output_path: Path) -> None:
    capture = cv2.VideoCapture(str(input_path))
    if not capture.isOpened():
        raise SystemExit(f"não abriu: {input_path}")
    fps = capture.get(cv2.CAP_PROP_FPS) or 24.0
    width = int(capture.get(cv2.CAP_PROP_FRAME_WIDTH))
    height = int(capture.get(cv2.CAP_PROP_FRAME_HEIGHT))
    writer = cv2.VideoWriter(
        str(output_path), cv2.VideoWriter_fourcc(*"mp4v"), fps, (width * 2, height)
    )

    rng = np.random.default_rng(7)
    particles = ParticleField(rng)
    mask_ema: np.ndarray | None = None
    prev_small: np.ndarray | None = None
    prev_area = 0.0
    shake_energy = 0.0
    frame_index = 0
    effect_frames = 0
    starved_frames = 0
    core_areas: deque[float] = deque(maxlen=PULSE_WINDOW)

    while True:
        ok, raw = capture.read()
        if not ok:
            break
        frame = raw.astype(np.float32) / 255.0
        mask = effect_mask(frame)
        mask_ema = mask if mask_ema is None else mask_ema * MASK_EMA + mask * (1 - MASK_EMA)
        area = float(mask_ema.mean())
        core_areas.append(float((mask_ema > PARTICLE_SPAWN_CORE).mean()))

        small = cv2.resize(cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY), None, fx=FLOW_SCALE, fy=FLOW_SCALE)
        flow = np.zeros((*small.shape, 2), dtype=np.float32)
        if prev_small is not None:
            if float(np.abs(small - prev_small).mean()) > SCENE_CUT_DIFF:
                particles.clear()
                mask_ema = mask
                shake_energy = 0.0
                prev_area = float(mask.mean())
                core_areas.clear()
            else:
                flow = cv2.calcOpticalFlowFarneback(prev_small, small, None, 0.5, 3, 15, 3, 5, 1.2, 0)
        prev_small = small

        if area < MASK_MIN_AREA:
            starved_frames += 1
            if starved_frames >= PARTICLE_STARVE_FRAMES:
                particles.clear()
            out = frame
            particles.step(flow, (height, width))
            out = particles.render(out)
        else:
            starved_frames = 0
            effect_frames += 1
            out = heat_distortion(frame, mask_ema, frame_index)
            out = regrade_effect(out, mask_ema)
            out = apply_bloom(out, mask_ema)
            out = light_wrap(out, mask_ema)
            if area >= PARTICLE_SPAWN_MIN_AREA and effect_is_pulsing(core_areas):
                particles.spawn(frame, mask_ema)
            particles.step(flow, (height, width))
            out = particles.render(out)

        if area - prev_area > SHAKE_TRIGGER:
            shake_energy = 1.0
        prev_area = area
        if shake_energy > 0.05:
            dx, dy = rng.normal(0, SHAKE_PIXELS * shake_energy, 2)
            matrix = np.float32([[1, 0, dx], [0, 1, dy]])
            out = cv2.warpAffine(out, matrix, (width, height), borderMode=cv2.BORDER_REFLECT)
            shake_energy *= SHAKE_DECAY

        writer.write(np.hstack([label(frame, "original"), label(out, "comp")]))
        frame_index += 1
        if frame_index % 100 == 0:
            print(f"{frame_index} frames, {effect_frames} com efeito, mascara media {area:.3f}")

    capture.release()
    writer.release()
    print(f"pronto: {output_path} ({frame_index} frames, {effect_frames} com efeito)")


def main() -> None:
    parser = argparse.ArgumentParser(description="Teste dos 5 itens de comp em efeitos de anime")
    parser.add_argument("input", type=Path)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    process(args.input, args.output)


if __name__ == "__main__":
    main()
