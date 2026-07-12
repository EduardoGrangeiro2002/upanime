from __future__ import annotations

MASK_V_MIN = 0.78
MASK_S_MIN = 0.6
MASK_HOT_V = 0.93
MASK_HOT_NEAR_SAT = 15
MASK_OPEN_KERNEL = 7
GROW_S_MIN = 0.55
GROW_V_MIN = 0.45
GROW_ITERATIONS = 16
GROW_WEIGHT = 0.65
MASK_MIN_AREA = 0.004
MASK_EMA = 0.55
BLOOM_SCALES = (4, 8, 16)
BLOOM_WEIGHTS = (0.45, 0.35, 0.25)
EFFECT_SATURATION = 1.35
EFFECT_GAIN = 1.08
CORE_V = 0.95
LIGHT_WRAP_SCALE = 8
LIGHT_WRAP_GAIN = 0.4
HEAT_AMPLITUDE = 2.4
NOISE_TILE = 256
NOISE_AMPLITUDE = 0.18
SHOCK_TRIGGER = 0.02
SHOCK_FRAMES = 12
SHOCK_GAIN = 0.25
SHAKE_PIXELS = 7.0
SHAKE_DECAY = 0.8


def _max_pool(torch, mask: object, kernel: int) -> object:
    return torch.nn.functional.max_pool2d(mask, kernel, stride=1, padding=kernel // 2)


def _erode(torch, mask: object, kernel: int) -> object:
    return 1.0 - _max_pool(torch, 1.0 - mask, kernel)


def _downblur(torch, image: object, scale: int) -> object:
    height, width = image.shape[-2:]
    small = torch.nn.functional.interpolate(image, scale_factor=1.0 / scale, mode="area")
    small = torch.nn.functional.avg_pool2d(small, 5, stride=1, padding=2)
    return torch.nn.functional.interpolate(small, size=(height, width), mode="bilinear", align_corners=False)


class EffectsComp:
    def __init__(
        self,
        torch: object,
        device: object,
        strength: float = 1.0,
        sensitivity: float = 1.0,
    ) -> None:
        self._torch = torch
        self._device = device
        self._strength = max(0.0, min(1.5, strength))
        self._sensitivity = max(0.5, min(1.5, sensitivity))
        generator = torch.Generator(device="cpu").manual_seed(7)
        noise = torch.rand((1, 1, NOISE_TILE, NOISE_TILE), generator=generator)
        noise = torch.nn.functional.avg_pool2d(noise, 9, stride=1, padding=4)
        self._noise = (noise - noise.mean()).to(device)
        self._shake_generator = torch.Generator(device="cpu").manual_seed(11)
        self.reset()

    def reset(self) -> None:
        self._mask_ema = None
        self._prev_area = 0.0
        self._shake_energy = 0.0
        self._shock_t = -1
        self._frame_index = 0

    def process(self, frame: object) -> object:
        torch = self._torch
        mask = self._mask(frame)
        if self._mask_ema is None:
            self._mask_ema = mask
        else:
            self._mask_ema = self._mask_ema * MASK_EMA + mask * (1 - MASK_EMA)
        area = float(self._mask_ema.mean())

        out = frame
        if area >= MASK_MIN_AREA:
            m = self._mask_ema
            out = self._heat(out, m)
            out = self._energy_noise(out, m)
            out = self._regrade(out, m)
            out = self._bloom(out, m)
            out = self._light_wrap(out, m)
            out = self._shockwave(out, m)
            if self._strength < 1.0:
                out = frame + (out - frame) * self._strength
            out = out.clamp(0, 1)

        if area - self._prev_area > SHOCK_TRIGGER:
            self._shake_energy = 1.0
            self._shock_t = 0
        elif self._shock_t >= 0:
            self._shock_t += 1
            if self._shock_t > SHOCK_FRAMES:
                self._shock_t = -1
        self._prev_area = area

        if self._shake_energy > 0.05:
            out = self._shake(out)
            self._shake_energy *= SHAKE_DECAY

        self._frame_index += 1
        return out

    def _mask(self, frame: object) -> object:
        torch = self._torch
        v = frame.amax(dim=1, keepdim=True)
        mn = frame.amin(dim=1, keepdim=True)
        s = (v - mn) / v.clamp(min=1e-6)
        s_min = MASK_S_MIN / self._sensitivity
        grow_s_min = GROW_S_MIN / self._sensitivity
        saturated = ((v > MASK_V_MIN) & (s > s_min)).float()
        near = _max_pool(torch, saturated, MASK_HOT_NEAR_SAT)
        hot = (v > MASK_HOT_V).float() * near
        seed = (saturated + hot).clamp(0, 1)
        seed = _max_pool(torch, _erode(torch, seed, MASK_OPEN_KERNEL), MASK_OPEN_KERNEL)

        weak = ((s > grow_s_min) & (v > GROW_V_MIN)).float()
        grown = seed
        for _ in range(GROW_ITERATIONS):
            grown = _max_pool(torch, grown, 5) * weak

        mask = seed + (grown - seed).clamp(min=0) * GROW_WEIGHT
        mask = _max_pool(torch, mask, 5)
        return torch.nn.functional.avg_pool2d(mask, 9, stride=1, padding=4)

    def _regrade(self, frame: object, mask: object) -> object:
        gray = frame.mean(dim=1, keepdim=True)
        graded = (gray + (frame - gray) * EFFECT_SATURATION) * EFFECT_GAIN
        graded = graded.clamp(0, 1)
        v = graded.amax(dim=1, keepdim=True)
        core = ((v - CORE_V) / (1.0 - CORE_V)).clamp(0, 1) * 0.6
        graded = graded * (1 - core) + core
        return frame * (1 - mask) + graded * mask

    def _bloom(self, frame: object, mask: object) -> object:
        torch = self._torch
        bright = frame * mask
        bloom = torch.zeros_like(frame)
        for scale, weight in zip(BLOOM_SCALES, BLOOM_WEIGHTS):
            bloom = bloom + _downblur(torch, bright, scale) * weight
        bloom = (bloom * self._strength).clamp(0, 1)
        return 1.0 - (1.0 - frame) * (1.0 - bloom)

    def _light_wrap(self, frame: object, mask: object) -> object:
        torch = self._torch
        spill = _downblur(torch, frame * mask, LIGHT_WRAP_SCALE)
        return (frame + spill * (1 - mask) * LIGHT_WRAP_GAIN * self._strength).clamp(0, 1)

    def _heat(self, frame: object, mask: object) -> object:
        torch = self._torch
        _, _, height, width = frame.shape
        strength = torch.nn.functional.avg_pool2d(mask, 9, stride=1, padding=4) * HEAT_AMPLITUDE
        ys = torch.arange(height, device=self._device, dtype=frame.dtype)
        ripple = torch.sin(ys / 6.5 + self._frame_index * 1.1)[None, None, :, None] * strength
        base_y = torch.linspace(-1, 1, height, device=self._device, dtype=frame.dtype)
        base_x = torch.linspace(-1, 1, width, device=self._device, dtype=frame.dtype)
        grid_y, grid_x = torch.meshgrid(base_y, base_x, indexing="ij")
        grid_x = grid_x[None] + ripple[:, 0] * (2.0 / max(width - 1, 1))
        grid = torch.stack([grid_x, grid_y[None].expand_as(grid_x)], dim=-1)
        return torch.nn.functional.grid_sample(frame, grid, mode="bilinear", padding_mode="reflection", align_corners=True)

    def _energy_noise(self, frame: object, mask: object) -> object:
        torch = self._torch
        _, _, height, width = frame.shape
        shift_y = (self._frame_index * 3) % NOISE_TILE
        shift_x = (self._frame_index * 5) % NOISE_TILE
        tile = torch.roll(self._noise, shifts=(shift_y, shift_x), dims=(2, 3))
        noise = torch.nn.functional.interpolate(tile, size=(height, width), mode="bilinear", align_corners=False)
        modulation = 1.0 + noise * NOISE_AMPLITUDE * self._strength * mask
        return (frame * modulation).clamp(0, 1)

    def _shockwave(self, frame: object, mask: object) -> object:
        if self._shock_t < 0:
            return frame
        torch = self._torch
        _, _, height, width = frame.shape
        flat = mask[0, 0]
        total = flat.sum()
        if float(total) <= 0:
            return frame
        ys = torch.arange(height, device=self._device, dtype=frame.dtype)
        xs = torch.arange(width, device=self._device, dtype=frame.dtype)
        cy = float((flat.sum(dim=1) * ys).sum() / total)
        cx = float((flat.sum(dim=0) * xs).sum() / total)
        grid_y, grid_x = torch.meshgrid(ys - cy, xs - cx, indexing="ij")
        distance = (grid_y.pow(2) + grid_x.pow(2)).sqrt()
        progress = self._shock_t / SHOCK_FRAMES
        radius = progress * max(height, width) * 0.6
        ring = torch.exp(-((distance - radius) / (0.05 * max(height, width))).pow(2))
        gain = SHOCK_GAIN * (1.0 - progress) * self._strength
        return (frame + ring[None, None] * gain).clamp(0, 1)

    def _shake(self, frame: object) -> object:
        torch = self._torch
        amplitude = SHAKE_PIXELS * self._shake_energy * self._strength
        if amplitude <= 0:
            return frame
        offsets = torch.normal(0.0, amplitude, (2,), generator=self._shake_generator)
        dx, dy = int(offsets[0]), int(offsets[1])
        if dx == 0 and dy == 0:
            return frame
        pad = max(abs(dx), abs(dy))
        padded = torch.nn.functional.pad(frame, (pad, pad, pad, pad), mode="reflect")
        _, _, height, width = frame.shape
        return padded[:, :, pad + dy : pad + dy + height, pad + dx : pad + dx + width]
