from __future__ import annotations

import importlib.util
import json
import logging
import math
import queue
import subprocess
import sys
import threading
import time
import types
from dataclasses import dataclass, replace
from pathlib import Path

import requests

from .effects import EffectsComp
from .interpolation import (
    PAN_RESIDUAL_RATIO,
    FrameDeduper,
    InterpolateFrame,
    classify_gaps,
    collapse_plan,
    detect_scene_cuts,
    plan_output_frames,
)
from .tagger import EffectTagger

MODEL_URL = "https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.5.0/realesr-animevideov3.pth"
HURRDEBLUR_URL = "https://objectstorage.us-phoenix-1.oraclecloud.com/n/ax6ygfvpvzka/b/open-modeldb-files/o/1x-HurrDeblur-SuperUltraCompact.pth"
APISR_URL = "https://github.com/Kiteretsu77/APISR/releases/download/v0.1.0/4x_APISR_GRL_GAN_generator.pth"
APISR_MIN_TILE_PIXELS = 65536
APISR_TILE_OVERLAP = 32
MODEL_SCALE = 4
BATCH_SIZE = 2
TARGET_FPS = 60.0
RIFE_PAD_MULTIPLE = 64
CLASSIFY_GRAY_WIDTH = 320
SHARPEN_GPU_AMOUNT = 0.5
SHARPEN_GPU_KERNEL_SIZE = 3
SHARPEN_GPU_SIGMA = 1.0
SATURATION = 1.20
CONTRAST = 1.05
BRIGHTNESS = 0.0
PREPROCESS_VF = "yadif=deint=interlaced,hqdn3d=4:3:6:4"
DECODE_QUEUE_SIZE = 24
ENCODE_QUEUE_SIZE = 24


@dataclass(frozen=True)
class EncodeParams:
    sharpen: float
    saturation: float
    contrast: float


@dataclass(frozen=True)
class VideoMetadata:
    width: int
    height: int
    fps: float
    total_frames: int | None
    duration: float
    target_height: int
    target_width: int
    outscale: float


@dataclass(frozen=True)
class PipelineRuntime:
    cv2: object
    numpy: object
    torch: object
    device: object
    raw_model: object
    hurrdeblur_model: object
    use_half: bool
    sharpen_kernel: object


class QualityUpscalePipeline:
    def __init__(
        self,
        model_path: Path,
        hurrdeblur_model_path: Path,
        target_height: int,
        encode_preset: str,
        enable_torch_compile: bool,
        rife_dir: Path | None = None,
        tagger: EffectTagger | None = None,
        apisr_model_path: Path | None = None,
    ) -> None:
        self._model_path = model_path
        self._hurrdeblur_model_path = hurrdeblur_model_path
        self._apisr_model_path = apisr_model_path
        self._target_height = target_height
        self._encode_preset = encode_preset
        self._enable_torch_compile = enable_torch_compile
        self._rife_dir = rife_dir
        self._tagger = tagger
        self._runtime: PipelineRuntime | None = None
        self._runtime_lock = threading.Lock()
        self._rife_model: object | None = None
        self._rife_lock = threading.Lock()
        self._apisr_model: object | None = None
        self._apisr_lock = threading.Lock()
        self._upscaler = "compact"

    def process(
        self,
        input_path: Path,
        output_path: Path,
        target_height: int | None = None,
        batch_size: int | None = None,
        sharpen: float | None = None,
        saturation: float | None = None,
        contrast: float | None = None,
        interpolate: bool = False,
        pan_residual_ratio: float | None = None,
        effects: bool = False,
        effects_strength: float | None = None,
        effects_sensitivity: float | None = None,
        skip_upscale: bool = False,
        upscaler: str | None = None,
        dataset_dir: Path | None = None,
    ) -> None:
        self._upscaler = upscaler or "compact"
        encode_params = EncodeParams(
            sharpen=max(0.0, min(2.0, sharpen)) if sharpen is not None else SHARPEN_GPU_AMOUNT,
            saturation=saturation if saturation is not None else SATURATION,
            contrast=contrast if contrast is not None else CONTRAST,
        )
        effective_height = target_height or self._target_height
        effective_batch = max(1, min(16, batch_size)) if batch_size is not None else BATCH_SIZE
        effective_pan_ratio = pan_residual_ratio if pan_residual_ratio is not None else PAN_RESIDUAL_RATIO
        logging.info(
            "gpu_optimized pipeline: target=%dp batch=%d sharpen=%.2f saturation=%.2f contrast=%.2f interpolate=%s pan_ratio=%.2f effects=%s skip_upscale=%s upscaler=%s",
            effective_height, effective_batch, encode_params.sharpen, encode_params.saturation, encode_params.contrast, interpolate, effective_pan_ratio, effects, skip_upscale, self._upscaler,
        )
        metadata = self._probe_video(input_path, effective_height)
        if skip_upscale:
            metadata = replace(metadata, target_height=metadata.height, target_width=metadata.width, outscale=1.0)
        runtime = self._load_runtime()
        comp = None
        if effects:
            comp = EffectsComp(
                runtime.torch,
                runtime.device,
                strength=effects_strength if effects_strength is not None else 1.0,
                sensitivity=effects_sensitivity if effects_sensitivity is not None else 1.0,
            )
        effects_log: list[dict] | None = [] if comp is not None and dataset_dir is not None else None
        if interpolate:
            self._run_chain(input_path, output_path, metadata, runtime, encode_params, effective_batch, effective_pan_ratio, comp, dataset_dir, effects_log, skip_upscale)
        else:
            self._run_stream(input_path, output_path, metadata, runtime, encode_params, effective_batch, comp, dataset_dir, effects_log, skip_upscale)
        self._write_effects_log(dataset_dir, metadata.fps, "chain" if interpolate else "stream", effects_log)

    def _load_runtime(self) -> PipelineRuntime:
        with self._runtime_lock:
            if self._runtime is not None:
                return self._runtime

            self._download_model_if_needed()
            runtime = self._build_runtime()
            self._runtime = runtime
            return runtime

    def _download_model_if_needed(self) -> None:
        for path, url in [
            (self._model_path, MODEL_URL),
            (self._hurrdeblur_model_path, HURRDEBLUR_URL),
        ]:
            self._download_file(path, url)

    def _download_file(self, path: Path, url: str) -> None:
        if path.exists():
            return
        path.parent.mkdir(parents=True, exist_ok=True)
        response = requests.get(url, stream=True, timeout=300)
        response.raise_for_status()
        with path.open("wb") as destination:
            for chunk in response.iter_content(chunk_size=1024 * 1024):
                if not chunk:
                    continue
                destination.write(chunk)

    def _build_runtime(self) -> PipelineRuntime:
        import cv2
        import numpy as np
        import torch
        import torchvision.transforms.functional as F

        module_name = "torchvision.transforms.functional_tensor"
        existing_module = sys.modules.get(module_name)
        if (
            existing_module is None
            or getattr(existing_module, "__spec__", None) is None
        ):
            shim = types.ModuleType(module_name)
            shim.rgb_to_grayscale = F.rgb_to_grayscale
            sys.modules[module_name] = shim

        from realesrgan import RealESRGANer
        from realesrgan.archs.srvgg_arch import SRVGGNetCompact

        if not torch.cuda.is_available():
            raise RuntimeError("CUDA is required for the local upscale worker")

        torch.backends.cudnn.benchmark = True
        if hasattr(torch, "set_float32_matmul_precision"):
            torch.set_float32_matmul_precision("high")

        model_net = SRVGGNetCompact(
            num_in_ch=3,
            num_out_ch=3,
            num_feat=64,
            num_conv=16,
            upscale=MODEL_SCALE,
            act_type="prelu",
        )
        upsampler = RealESRGANer(
            scale=MODEL_SCALE,
            model_path=str(self._model_path),
            model=model_net,
            tile=0,
            tile_pad=10,
            pre_pad=10,
            half=True,
        )

        raw_model = upsampler.model.eval()
        if self._enable_torch_compile and hasattr(torch, "compile"):
            raw_model = raw_model.to(memory_format=torch.channels_last)
            raw_model = torch.compile(raw_model)

        device = torch.device("cuda")
        sharpen_kernel = self._build_sharpen_kernel(torch, device, upsampler.half)

        import spandrel
        hurrdeblur_loaded = spandrel.ModelLoader(device=device).load_from_file(
            str(self._hurrdeblur_model_path)
        )
        hurrdeblur_model = hurrdeblur_loaded.model.eval()
        if upsampler.half:
            hurrdeblur_model = hurrdeblur_model.half()
        if self._enable_torch_compile and hasattr(torch, "compile"):
            hurrdeblur_model = torch.compile(hurrdeblur_model)

        self._warmup(torch, np, raw_model, hurrdeblur_model, upsampler.half)

        return PipelineRuntime(
            cv2=cv2,
            numpy=np,
            torch=torch,
            device=device,
            raw_model=raw_model,
            hurrdeblur_model=hurrdeblur_model,
            use_half=upsampler.half,
            sharpen_kernel=sharpen_kernel,
        )

    def _build_sharpen_kernel(
        self, torch: object, device: object, use_half: bool
    ) -> object:
        coords = torch.arange(
            SHARPEN_GPU_KERNEL_SIZE, device=device, dtype=torch.float32
        ) - SHARPEN_GPU_KERNEL_SIZE // 2
        gauss_1d = torch.exp(-0.5 * (coords / SHARPEN_GPU_SIGMA) ** 2)
        gauss_1d = gauss_1d / gauss_1d.sum()
        gauss_2d = gauss_1d[:, None] * gauss_1d[None, :]
        kernel = gauss_2d.expand(3, 1, SHARPEN_GPU_KERNEL_SIZE, SHARPEN_GPU_KERNEL_SIZE).contiguous()
        if use_half:
            kernel = kernel.half()
        return kernel

    def _warmup(
        self, torch: object, np: object, raw_model: object, hurrdeblur_model: object, use_half: bool
    ) -> None:
        sample = np.random.randint(0, 255, (1, 3, 480, 640), dtype=np.uint8)
        batch = torch.from_numpy(sample).float().div(255.0).to("cuda")
        batch = batch.contiguous(memory_format=torch.channels_last)
        if use_half:
            batch = batch.half()
        with torch.inference_mode():
            raw_model(batch)
        torch.cuda.synchronize()
        deblur_sample = torch.randn(1, 3, 480, 640, device="cuda")
        if use_half:
            deblur_sample = deblur_sample.half()
        with torch.inference_mode():
            hurrdeblur_model(deblur_sample)
        torch.cuda.synchronize()

    def _probe_video(self, input_path: Path, target_height: int) -> VideoMetadata:
        result = subprocess.run(
            [
                "ffprobe",
                "-v",
                "quiet",
                "-print_format",
                "json",
                "-show_streams",
                "-show_format",
                str(input_path),
            ],
            capture_output=True,
            text=True,
            check=True,
        )
        probe = json.loads(result.stdout)
        video_stream = next(
            stream for stream in probe["streams"] if stream["codec_type"] == "video"
        )
        width = int(video_stream["width"])
        height = int(video_stream["height"])
        fps = self._parse_fps(video_stream.get("r_frame_rate", "24/1"))
        total_frames = self._parse_total_frames(video_stream.get("nb_frames"))
        duration = self._parse_duration(
            video_stream.get("duration"), probe.get("format", {}).get("duration")
        )
        outscale = target_height / height
        target_width = int(round(width * outscale))
        return VideoMetadata(
            width=width,
            height=height,
            fps=fps,
            total_frames=total_frames,
            duration=duration,
            target_height=target_height,
            target_width=target_width,
            outscale=outscale,
        )

    def _shot_gate(
        self,
        frames: list[object],
        shot_index: int,
        comp: EffectsComp,
        dataset_dir: Path | None,
        runtime: PipelineRuntime,
    ) -> tuple[bool, list[str], float]:
        comp.reset()
        if self._tagger is None or not self._tagger.available():
            return True, ["gate-off"], 1.0
        max_prob = 0.0
        for frame in frames:
            tags, prob = self._tagger.shot_effect_scores(frame)
            max_prob = max(max_prob, prob)
            if not tags:
                continue
            self._save_dataset_sample(frame, tags, shot_index, dataset_dir, runtime)
            return True, tags, max_prob
        return False, [], max_prob

    def _log_gate(
        self,
        effects_log: list[dict] | None,
        t: float,
        shot_index: int,
        event: str,
        gated: bool,
        tags: list[str],
        max_prob: float,
    ) -> None:
        if effects_log is None:
            return
        effects_log.append({
            "t": round(t, 3),
            "shot": shot_index,
            "event": event,
            "gated": gated,
            "tags": tags,
            "max_prob": round(max_prob, 4),
        })

    def _write_effects_log(
        self,
        dataset_dir: Path | None,
        fps: float,
        mode: str,
        entries: list[dict] | None,
    ) -> None:
        if entries is None or dataset_dir is None:
            return
        try:
            dataset_dir.mkdir(parents=True, exist_ok=True)
            (dataset_dir / "effects_log.json").write_text(
                json.dumps({"mode": mode, "fps": fps, "entries": entries})
            )
        except Exception:
            logging.exception("effects log write failed")

    def _segment_samples(self, frames: list[object], start: int, end: int) -> list[object]:
        length = end - start
        picks = sorted({start, start + length // 3, start + 2 * length // 3})
        return [frames[p] for p in picks if start <= p < end]

    def _save_dataset_sample(
        self,
        frame: object,
        tags: list[str],
        shot_index: int,
        dataset_dir: Path | None,
        runtime: PipelineRuntime,
    ) -> None:
        if dataset_dir is None:
            return
        try:
            dataset_dir.mkdir(parents=True, exist_ok=True)
            runtime.cv2.imwrite(str(dataset_dir / f"shot_{shot_index:04d}.jpg"), frame)
            (dataset_dir / f"shot_{shot_index:04d}.json").write_text(json.dumps({"tags": tags}))
        except Exception:
            logging.exception("dataset sample save failed for shot %d", shot_index)

    def _shot_small_gray(self, frame: object, runtime: PipelineRuntime) -> object:
        gray = runtime.cv2.cvtColor(frame, runtime.cv2.COLOR_BGR2GRAY).astype(runtime.numpy.float32) / 255.0
        small_height = max(2, int(round(gray.shape[0] * CLASSIFY_GRAY_WIDTH / gray.shape[1])))
        return runtime.cv2.resize(gray, (CLASSIFY_GRAY_WIDTH, small_height))

    def _run_stream(
        self,
        input_path: Path,
        output_path: Path,
        metadata: VideoMetadata,
        runtime: PipelineRuntime,
        encode_params: EncodeParams | None = None,
        batch_size: int = BATCH_SIZE,
        comp: EffectsComp | None = None,
        dataset_dir: Path | None = None,
        effects_log: list[dict] | None = None,
        skip_upscale: bool = False,
    ) -> None:
        frame_bytes = metadata.width * metadata.height * 3
        expected_frames = metadata.total_frames or int(
            round(metadata.duration * metadata.fps)
        )

        decode_proc = subprocess.Popen(
            self._build_decode_command(input_path),
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            bufsize=frame_bytes * 4,
        )
        encode_proc = subprocess.Popen(
            self._build_encode_command(input_path, output_path, metadata, encode_params),
            stdin=subprocess.PIPE,
            stderr=subprocess.PIPE,
            bufsize=frame_bytes * 4,
        )

        decode_queue: queue.Queue[object] = queue.Queue(maxsize=DECODE_QUEUE_SIZE)
        encode_queue: queue.Queue[object] = queue.Queue(maxsize=ENCODE_QUEUE_SIZE)
        sentinel = object()
        decoder_state: dict[str, object] = {}
        encoder_state: dict[str, object] = {}

        decoder = threading.Thread(
            target=self._decoder_worker,
            args=(
                decode_proc,
                decode_queue,
                sentinel,
                decoder_state,
                frame_bytes,
                metadata,
                runtime,
            ),
            daemon=True,
        )
        encoder = threading.Thread(
            target=self._encoder_worker,
            args=(encode_proc, encode_queue, sentinel, encoder_state, runtime),
            daemon=True,
        )

        runtime.torch.cuda.empty_cache()
        started = time.time()
        handled = 0
        pending_images: list[object] = []
        cut_threshold = (1.0 - 0.4) * 0.5
        comp_active = False
        prev_small = None
        shot_index = 0
        frames_in_shot = 0

        decoder.start()
        encoder.start()

        while True:
            item = decode_queue.get()
            if item is sentinel:
                break

            if comp is not None:
                small = self._shot_small_gray(item, runtime)
                is_first = prev_small is None
                is_cut = not is_first and float(abs(small - prev_small).mean()) > cut_threshold
                if is_first or is_cut:
                    if pending_images:
                        self._write_batch(encode_queue, pending_images, runtime, metadata, encode_params, comp if comp_active else None, skip_upscale)
                        pending_images = []
                    comp_active, tags, max_prob = self._shot_gate([item], shot_index, comp, dataset_dir, runtime)
                    self._log_gate(effects_log, handled / metadata.fps, shot_index, "cut", comp_active, tags, max_prob)
                    shot_index += 1
                    frames_in_shot = 0
                elif not comp_active:
                    frames_in_shot += 1
                    if frames_in_shot % 24 == 0:
                        gated, tags, max_prob = self._shot_gate([item], shot_index - 1, comp, dataset_dir, runtime)
                        self._log_gate(effects_log, handled / metadata.fps, shot_index - 1, "recheck", gated, tags, max_prob)
                        if gated:
                            if pending_images:
                                self._write_batch(encode_queue, pending_images, runtime, metadata, encode_params, None, skip_upscale)
                                pending_images = []
                            comp_active = True
                prev_small = small

            pending_images.append(item)
            handled += 1

            if len(pending_images) < batch_size:
                continue

            self._write_batch(encode_queue, pending_images, runtime, metadata, encode_params, comp if comp_active else None, skip_upscale)
            pending_images = []
            self._log_progress(handled, expected_frames, started)

        if pending_images:
            self._write_batch(encode_queue, pending_images, runtime, metadata, encode_params, comp if comp_active else None, skip_upscale)

        encode_queue.put(sentinel)
        decoder.join()
        encoder.join()

        self._raise_worker_errors(decoder_state, encoder_state)
        logging.info(
            "gpu_optimized pipeline finished in %.1fs (%d frames)",
            time.time() - started,
            handled,
        )

    def _write_batch(
        self,
        encode_queue: queue.Queue[object],
        pending_images: list[object],
        runtime: PipelineRuntime,
        metadata: VideoMetadata,
        encode_params: EncodeParams | None = None,
        comp: EffectsComp | None = None,
        skip_upscale: bool = False,
    ) -> None:
        frames = self._upscale_batch(pending_images, runtime, metadata, encode_params, comp, skip_upscale)
        for frame in frames:
            encode_queue.put(frame)

    def _load_apisr(self, runtime: PipelineRuntime) -> object:
        with self._apisr_lock:
            if self._apisr_model is not None:
                return self._apisr_model

            if self._apisr_model_path is None:
                raise RuntimeError("upscaler=apisr requested but WORKER_APISR_MODEL_PATH is not configured")

            self._download_file(self._apisr_model_path, APISR_URL)

            import spandrel
            descriptor = spandrel.ModelLoader(device=runtime.device).load_from_file(
                str(self._apisr_model_path)
            )
            if descriptor.scale != MODEL_SCALE:
                raise RuntimeError(f"APISR model scale {descriptor.scale} != {MODEL_SCALE}")
            model = descriptor.model.eval()
            if runtime.use_half:
                model = model.half()
            self._apisr_model = model
            return model

    def _apisr_infer(self, model: object, tensor: object, runtime: PipelineRuntime) -> object:
        _, _, height, width = tensor.shape
        try:
            with runtime.torch.inference_mode():
                return model(tensor)
        except runtime.torch.cuda.OutOfMemoryError:
            if height * width <= APISR_MIN_TILE_PIXELS:
                raise
            runtime.torch.cuda.empty_cache()

        overlap = APISR_TILE_OVERLAP
        if height >= width:
            mid = height // 2
            top = self._apisr_infer(model, tensor[:, :, : mid + overlap, :], runtime)
            bottom = self._apisr_infer(model, tensor[:, :, mid - overlap :, :], runtime)
            return runtime.torch.cat(
                [top[:, :, : mid * MODEL_SCALE, :], bottom[:, :, overlap * MODEL_SCALE :, :]], dim=2
            )
        mid = width // 2
        left = self._apisr_infer(model, tensor[:, :, :, : mid + overlap], runtime)
        right = self._apisr_infer(model, tensor[:, :, :, mid - overlap :], runtime)
        return runtime.torch.cat(
            [left[:, :, :, : mid * MODEL_SCALE], right[:, :, :, overlap * MODEL_SCALE :]], dim=3
        )

    def _apisr_upscale_batch(
        self,
        images: list[object],
        runtime: PipelineRuntime,
        metadata: VideoMetadata,
        encode_params: EncodeParams | None,
        comp: EffectsComp | None,
    ) -> list[object]:
        model = self._load_apisr(runtime)
        sizes = [(image.shape[0], image.shape[1]) for image in images]
        outputs = []
        for image in images:
            tensor = (
                runtime.torch.from_numpy(runtime.numpy.array(image, copy=True))
                .permute(2, 0, 1)
                .float()
                .div(255.0)
                .unsqueeze(0)
                .to(runtime.device)
            )
            if runtime.use_half:
                tensor = tensor.half()
            outputs.append(self._apisr_infer(model, tensor, runtime))
        output = runtime.torch.cat(outputs, dim=0).clamp_(0, 1)
        sharpen_amount = encode_params.sharpen if encode_params else SHARPEN_GPU_AMOUNT
        return self._decode_gpu_optimized_frames(output, sizes, runtime, metadata, sharpen_amount, comp)

    def _load_rife(self, runtime: PipelineRuntime) -> object:
        with self._rife_lock:
            if self._rife_model is not None:
                return self._rife_model

            if self._rife_dir is None:
                raise RuntimeError("interpolate requested but WORKER_RIFE_DIR is not configured")

            train_log = self._rife_dir / "train_log"
            model_py = train_log / "RIFE_HDv3.py"
            if not model_py.exists() or not (train_log / "flownet.pkl").exists():
                raise RuntimeError(f"RIFE model files not found in {train_log}")

            rife_root = str(self._rife_dir)
            if rife_root not in sys.path:
                sys.path.insert(0, rife_root)

            spec = importlib.util.spec_from_file_location("RIFE_HDv3", str(model_py))
            module = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(module)

            model = module.Model()
            if not hasattr(model, "version"):
                model.version = 0
            model.load_model(str(train_log), -1)
            model.eval()
            model.device()
            self._rife_model = model
            return model

    def _to_rife_tensor(self, frame: object, runtime: PipelineRuntime) -> object:
        rgb = runtime.cv2.cvtColor(frame, runtime.cv2.COLOR_BGR2RGB)
        tensor = (
            runtime.torch.from_numpy(rgb.copy())
            .permute(2, 0, 1)
            .float()
            .div(255.0)
            .unsqueeze(0)
            .to(runtime.device)
        )
        _, _, height, width = tensor.shape
        pad_h = (RIFE_PAD_MULTIPLE - height % RIFE_PAD_MULTIPLE) % RIFE_PAD_MULTIPLE
        pad_w = (RIFE_PAD_MULTIPLE - width % RIFE_PAD_MULTIPLE) % RIFE_PAD_MULTIPLE
        if pad_h > 0 or pad_w > 0:
            tensor = runtime.torch.nn.functional.pad(tensor, (0, pad_w, 0, pad_h), mode="reflect")
        return tensor

    def _from_rife_tensor(self, tensor: object, runtime: PipelineRuntime, metadata: VideoMetadata) -> object:
        tensor = tensor[:, :, : metadata.height, : metadata.width]
        frame = tensor.squeeze(0).mul(255.0).clamp_(0, 255).byte().permute(1, 2, 0).cpu().numpy()
        return runtime.cv2.cvtColor(frame, runtime.cv2.COLOR_RGB2BGR)

    def _decode_unique_frames(
        self, input_path: Path, metadata: VideoMetadata, runtime: PipelineRuntime
    ) -> tuple[list[object], list[int], list[object], int]:
        np = runtime.numpy
        cv2 = runtime.cv2
        frame_bytes = metadata.width * metadata.height * 3
        small_height = max(2, int(round(metadata.height * CLASSIFY_GRAY_WIDTH / metadata.width)))

        decode_proc = subprocess.Popen(
            self._build_decode_command(input_path),
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            bufsize=frame_bytes * 4,
        )

        deduper = FrameDeduper()
        unique_frames: list[object] = []
        unique_indices: list[int] = []
        small_grays: list[object] = []
        total_decoded = 0

        try:
            while True:
                payload = self._read_exact(decode_proc.stdout, frame_bytes)
                if payload is None:
                    break
                frame = np.frombuffer(payload, dtype=np.uint8).reshape(
                    metadata.height, metadata.width, 3
                ).copy()
                gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY).astype(np.float32) / 255.0
                if deduper.is_unique(gray):
                    unique_frames.append(frame)
                    unique_indices.append(total_decoded)
                    small_grays.append(cv2.resize(gray, (CLASSIFY_GRAY_WIDTH, small_height)))
                total_decoded += 1
        finally:
            decode_proc.stdout.close()

        stderr = decode_proc.stderr.read().decode("utf-8", errors="replace")
        if decode_proc.wait() != 0:
            raise RuntimeError(stderr)
        if not unique_frames:
            raise RuntimeError("no frames decoded from source video")

        return unique_frames, unique_indices, small_grays, total_decoded

    def _run_chain(
        self,
        input_path: Path,
        output_path: Path,
        metadata: VideoMetadata,
        runtime: PipelineRuntime,
        encode_params: EncodeParams,
        batch_size: int,
        pan_residual_ratio: float = PAN_RESIDUAL_RATIO,
        comp: EffectsComp | None = None,
        dataset_dir: Path | None = None,
        effects_log: list[dict] | None = None,
        skip_upscale: bool = False,
    ) -> None:
        rife = self._load_rife(runtime)

        started = time.time()
        unique_frames, unique_indices, small_grays, total_decoded = self._decode_unique_frames(
            input_path, metadata, runtime
        )
        logging.info(
            "chain pass 1: %d frames -> %d unique in %.1fs",
            total_decoded, len(unique_frames), time.time() - started,
        )

        total_duration = total_decoded / metadata.fps
        unique_timestamps = [index / metadata.fps for index in unique_indices]
        is_scene_cut = detect_scene_cuts(small_grays)
        allow = classify_gaps(small_grays, is_scene_cut, pan_residual_ratio)
        plan = plan_output_frames(unique_timestamps, total_duration, TARGET_FPS, allow)
        interp_count = sum(1 for op in plan if isinstance(op, InterpolateFrame))
        logging.info(
            "chain plan: %d output frames, %d interpolated (%.1f%%), %d/%d gaps approved for interpolation",
            len(plan), interp_count, 100.0 * interp_count / max(len(plan), 1), sum(allow), max(len(allow), 1),
        )
        collapsed = collapse_plan(plan)

        segment_ids = [0] * len(unique_frames)
        for i in range(1, len(unique_frames)):
            segment_ids[i] = segment_ids[i - 1] + (1 if is_scene_cut[i - 1] else 0)
        segment_comp: dict[int, bool] = {}
        if comp is not None:
            segment_starts: dict[int, int] = {}
            for i in range(len(unique_frames)):
                segment_starts.setdefault(segment_ids[i], i)
            for sid, start in segment_starts.items():
                end = len(unique_frames)
                for j in range(start, len(unique_frames)):
                    if segment_ids[j] != sid:
                        end = j
                        break
                samples = self._segment_samples(unique_frames, start, end)
                gated, tags, max_prob = self._shot_gate(samples, sid, comp, dataset_dir, runtime)
                segment_comp[sid] = gated
                if effects_log is not None:
                    end_s = unique_timestamps[end] if end < len(unique_timestamps) else total_duration
                    effects_log.append({
                        "segment": sid,
                        "start_s": round(unique_timestamps[start], 3),
                        "end_s": round(end_s, 3),
                        "gated": gated,
                        "tags": tags,
                        "max_prob": round(max_prob, 4),
                    })
            logging.info(
                "chain comp gate: %d/%d segments approved",
                sum(1 for flag in segment_comp.values() if flag), len(segment_comp),
            )

        encode_proc = subprocess.Popen(
            self._build_encode_command(
                input_path, output_path, metadata, encode_params, fps_override=TARGET_FPS
            ),
            stdin=subprocess.PIPE,
            stderr=subprocess.PIPE,
            bufsize=metadata.width * metadata.height * 3 * 4,
        )
        encode_queue: queue.Queue[object] = queue.Queue(maxsize=ENCODE_QUEUE_SIZE)
        sentinel = object()
        encoder_state: dict[str, object] = {}
        encoder = threading.Thread(
            target=self._encoder_worker,
            args=(encode_proc, encode_queue, sentinel, encoder_state, runtime),
            daemon=True,
        )
        encoder.start()

        runtime.torch.cuda.empty_cache()
        started_pass2 = time.time()
        handled = 0
        pending: list[tuple[object, int]] = []
        cached_left_index = -1
        cached_left = None
        cached_right_index = -1
        cached_right = None
        current_segment = -1
        comp_active = False

        for op, count in collapsed:
            drawing_index = op.left_index if isinstance(op, InterpolateFrame) else op.index
            if comp is not None and segment_ids[drawing_index] != current_segment:
                if pending:
                    self._write_batch_counted(encode_queue, pending, runtime, metadata, encode_params, comp if comp_active else None, skip_upscale)
                    pending = []
                current_segment = segment_ids[drawing_index]
                comp.reset()
                comp_active = segment_comp.get(current_segment, False)

            if isinstance(op, InterpolateFrame):
                if cached_left_index != op.left_index:
                    cached_left = self._to_rife_tensor(unique_frames[op.left_index], runtime)
                    cached_left_index = op.left_index
                if cached_right_index != op.left_index + 1:
                    cached_right = self._to_rife_tensor(unique_frames[op.left_index + 1], runtime)
                    cached_right_index = op.left_index + 1
                with runtime.torch.inference_mode():
                    mid = rife.inference(cached_left, cached_right, timestep=op.timestep)
                frame = self._from_rife_tensor(mid, runtime, metadata)
            else:
                frame = unique_frames[op.index]

            pending.append((frame, count))
            handled += count
            if len(pending) < batch_size:
                continue

            self._write_batch_counted(encode_queue, pending, runtime, metadata, encode_params, comp if comp_active else None, skip_upscale)
            pending = []
            self._log_progress(handled, len(plan), started_pass2)

        if pending:
            self._write_batch_counted(encode_queue, pending, runtime, metadata, encode_params, comp if comp_active else None, skip_upscale)

        encode_queue.put(sentinel)
        encoder.join()

        self._raise_worker_errors({}, encoder_state)
        logging.info(
            "chain pipeline finished in %.1fs (%d in -> %d out frames)",
            time.time() - started,
            total_decoded,
            handled,
        )

    def _write_batch_counted(
        self,
        encode_queue: queue.Queue[object],
        pending: list[tuple[object, int]],
        runtime: PipelineRuntime,
        metadata: VideoMetadata,
        encode_params: EncodeParams | None = None,
        comp: EffectsComp | None = None,
        skip_upscale: bool = False,
    ) -> None:
        frames = self._upscale_batch([frame for frame, _ in pending], runtime, metadata, encode_params, comp, skip_upscale)
        for frame, (_, count) in zip(frames, pending):
            for _ in range(count):
                encode_queue.put(frame)

    def _decoder_worker(
        self,
        decode_proc: subprocess.Popen[bytes],
        decode_queue: queue.Queue[object],
        sentinel: object,
        decoder_state: dict[str, object],
        frame_bytes: int,
        metadata: VideoMetadata,
        runtime: PipelineRuntime,
    ) -> None:
        try:
            while True:
                payload = self._read_exact(decode_proc.stdout, frame_bytes)
                if payload is None:
                    break
                frame = runtime.numpy.frombuffer(payload, dtype=runtime.numpy.uint8)
                decode_queue.put(frame.reshape(metadata.height, metadata.width, 3))
        except Exception as exc:
            decoder_state["error"] = exc
        finally:
            if decode_proc.stdout is not None:
                decode_proc.stdout.close()
            decoder_state["stderr"] = decode_proc.stderr.read().decode(
                "utf-8", errors="replace"
            )
            decoder_state["returncode"] = decode_proc.wait()
            decode_queue.put(sentinel)

    def _encoder_worker(
        self,
        encode_proc: subprocess.Popen[bytes],
        encode_queue: queue.Queue[object],
        sentinel: object,
        encoder_state: dict[str, object],
        runtime: PipelineRuntime,
    ) -> None:
        try:
            while True:
                item = encode_queue.get()
                if item is sentinel:
                    break
                encode_proc.stdin.write(runtime.numpy.ascontiguousarray(item).tobytes())
        except Exception as exc:
            encoder_state["error"] = exc
        finally:
            if encode_proc.stdin is not None:
                encode_proc.stdin.close()
            encoder_state["stderr"] = encode_proc.stderr.read().decode(
                "utf-8", errors="replace"
            )
            encoder_state["returncode"] = encode_proc.wait()

    def _upscale_batch(
        self,
        images: list[object],
        runtime: PipelineRuntime,
        metadata: VideoMetadata,
        encode_params: EncodeParams | None = None,
        comp: EffectsComp | None = None,
        skip_upscale: bool = False,
    ) -> list[object]:
        if skip_upscale:
            return self._comp_only_batch(images, runtime, comp)
        if self._upscaler == "apisr":
            return self._apisr_upscale_batch(images, runtime, metadata, encode_params, comp)
        batch, sizes = self._prepare_batch(images, runtime)

        batch = batch.to(runtime.device, non_blocking=True)
        batch = batch.contiguous(memory_format=runtime.torch.channels_last)
        if runtime.use_half:
            batch = batch.half()

        runtime.torch.cuda.synchronize()
        with runtime.torch.inference_mode():
            output = runtime.raw_model(batch).clamp_(0, 1)
        runtime.torch.cuda.synchronize()

        sharpen_amount = encode_params.sharpen if encode_params else SHARPEN_GPU_AMOUNT
        return self._decode_gpu_optimized_frames(output, sizes, runtime, metadata, sharpen_amount, comp)

    def _comp_only_batch(
        self,
        images: list[object],
        runtime: PipelineRuntime,
        comp: EffectsComp | None,
    ) -> list[object]:
        if comp is None:
            return [runtime.numpy.ascontiguousarray(image) for image in images]
        results = []
        for image in images:
            frame = (
                runtime.torch.from_numpy(runtime.numpy.array(image, copy=True))
                .permute(2, 0, 1)
                .float()
                .div(255.0)
                .unsqueeze(0)
                .to(runtime.device)
            )
            with runtime.torch.inference_mode():
                frame = comp.process(frame).clamp(0, 1)
            frame = (
                frame.mul(255.0)
                .clamp_(0, 255)
                .round_()
                .byte()
                .squeeze(0)
                .permute(1, 2, 0)
                .cpu()
                .numpy()
            )
            results.append(runtime.numpy.ascontiguousarray(frame))
        return results

    def _prepare_batch(
        self, images: list[object], runtime: PipelineRuntime
    ) -> tuple[object, list[tuple[int, int]]]:
        sizes = [(img.shape[0], img.shape[1]) for img in images]
        max_h = max(height for height, _ in sizes)
        max_w = max(width for _, width in sizes)
        pad_h = int(math.ceil(max_h / 4) * 4)
        pad_w = int(math.ceil(max_w / 4) * 4)

        tensors = []
        for image in images:
            tensor = (
                runtime.torch.from_numpy(runtime.numpy.array(image, copy=True))
                .permute(2, 0, 1)
                .float()
                .div(255.0)
            )
            height, width = tensor.shape[1], tensor.shape[2]
            if height < pad_h or width < pad_w:
                tensor = runtime.torch.nn.functional.pad(
                    tensor,
                    (0, pad_w - width, 0, pad_h - height),
                    mode="reflect",
                )
            tensors.append(tensor)

        return runtime.torch.stack(tensors).pin_memory(), sizes

    def _gpu_unsharp_mask(
        self, frames: object, runtime: PipelineRuntime, amount: float,
    ) -> object:
        if amount <= 0.0:
            return frames
        blurred = runtime.torch.nn.functional.conv2d(
            frames, runtime.sharpen_kernel,
            padding=SHARPEN_GPU_KERNEL_SIZE // 2, groups=3,
        )
        return (frames + amount * (frames - blurred)).clamp_(0, 1)

    def _run_hurrdeblur(self, frame: object, runtime: PipelineRuntime) -> object:
        with runtime.torch.inference_mode():
            return runtime.hurrdeblur_model(frame).clamp_(0, 1)

    def _decode_gpu_optimized_frames(
        self,
        output: object,
        sizes: list[tuple[int, int]],
        runtime: PipelineRuntime,
        metadata: VideoMetadata,
        sharpen_amount: float = SHARPEN_GPU_AMOUNT,
        comp: EffectsComp | None = None,
    ) -> list[object]:
        results = []
        for index, (orig_h, orig_w) in enumerate(sizes):
            frame = output[
                index : index + 1, :, : orig_h * MODEL_SCALE, : orig_w * MODEL_SCALE
            ]

            frame = self._gpu_unsharp_mask(frame, runtime, sharpen_amount)

            target_h = int(round(orig_h * metadata.outscale))
            target_w = int(round(orig_w * metadata.outscale))
            frame = runtime.torch.nn.functional.interpolate(
                frame, size=(target_h, target_w), mode="area",
            )

            frame = self._run_hurrdeblur(frame, runtime)

            if comp is not None:
                with runtime.torch.inference_mode():
                    frame = comp.process(frame.float()).clamp(0, 1)

            frame = (
                frame.clone().mul_(255.0)
                .clamp_(0, 255)
                .round_()
                .byte()
                .squeeze(0)
                .permute(1, 2, 0)
                .cpu()
                .numpy()
            )
            results.append(runtime.numpy.ascontiguousarray(frame))
        return results

    def _build_decode_command(self, input_path: Path) -> list[str]:
        return [
            "ffmpeg",
            "-hide_banner",
            "-loglevel",
            "error",
            "-i",
            str(input_path),
            "-vf",
            PREPROCESS_VF,
            "-f",
            "rawvideo",
            "-pix_fmt",
            "bgr24",
            "-",
        ]

    def _build_encode_command(
        self,
        input_path: Path,
        output_path: Path,
        metadata: VideoMetadata,
        encode_params: EncodeParams | None = None,
        fps_override: float | None = None,
    ) -> list[str]:
        params = encode_params or EncodeParams(
            sharpen=SHARPEN_GPU_AMOUNT, saturation=SATURATION, contrast=CONTRAST
        )
        vibrance = max(-2.0, min(2.0, (params.saturation - 1.0) * 1.5))
        encode_vf = (
            f"vibrance=intensity={vibrance:.3f},"
            "scale=out_color_matrix=bt709,format=yuv420p,"
            f"eq=contrast={params.contrast}:brightness={BRIGHTNESS},"
            "deband,"
            "setparams=color_primaries=bt709:color_trc=bt709:colorspace=bt709"
        )
        return [
            "ffmpeg",
            "-hide_banner",
            "-loglevel",
            "error",
            "-y",
            "-f",
            "rawvideo",
            "-pix_fmt",
            "bgr24",
            "-s",
            f"{metadata.target_width}x{metadata.target_height}",
            "-r",
            f"{fps_override or metadata.fps:.6f}",
            "-i",
            "-",
            "-i",
            str(input_path),
            "-map",
            "0:v",
            "-map",
            "1:a?",
            "-vf",
            encode_vf,
            "-c:v",
            "libx264",
            "-preset",
            self._encode_preset,
            "-crf",
            "18",
            "-c:a",
            "copy",
            str(output_path),
        ]

    def _read_exact(self, stream: object, size: int) -> bytes | None:
        buffer = bytearray()
        while len(buffer) < size:
            chunk = stream.read(size - len(buffer))
            if not chunk:
                break
            buffer.extend(chunk)
        if not buffer:
            return None
        if len(buffer) != size:
            raise EOFError(f"incomplete frame: expected {size}, got {len(buffer)}")
        return bytes(buffer)

    def _raise_worker_errors(
        self, decoder_state: dict[str, object], encoder_state: dict[str, object]
    ) -> None:
        if "error" in decoder_state:
            raise decoder_state["error"]  # type: ignore[misc]
        if "error" in encoder_state:
            raise encoder_state["error"]  # type: ignore[misc]
        if decoder_state.get("returncode", 0) != 0:
            raise RuntimeError(str(decoder_state.get("stderr", "")))
        if encoder_state.get("returncode", 0) != 0:
            raise RuntimeError(str(encoder_state.get("stderr", "")))

    def _log_progress(self, handled: int, expected_frames: int, started: float) -> None:
        if handled % 100 != 0:
            return
        fps = handled / max(time.time() - started, 0.01)
        logging.info(
            "processed %s/%s frames at %.1f fps", handled, expected_frames, fps
        )

    def _parse_fps(self, fps_value: str) -> float:
        num, den = fps_value.split("/")
        return float(num) / float(den)

    def _parse_total_frames(self, frame_value: str | None) -> int | None:
        if frame_value in (None, "N/A"):
            return None
        return int(frame_value)

    def _parse_duration(
        self, stream_duration: str | None, format_duration: str | None
    ) -> float:
        if stream_duration not in (None, "N/A"):
            return float(stream_duration)
        if format_duration not in (None, "N/A"):
            return float(format_duration)
        return 0.0
