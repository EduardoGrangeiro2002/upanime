from __future__ import annotations

import json
import logging
import math
import queue
import subprocess
import sys
import threading
import time
import types
from dataclasses import dataclass
from pathlib import Path

import requests

MODEL_URL = "https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.5.0/realesr-animevideov3.pth"
HURRDEBLUR_URL = "https://objectstorage.us-phoenix-1.oraclecloud.com/n/ax6ygfvpvzka/b/open-modeldb-files/o/1x-HurrDeblur-SuperUltraCompact.pth"
MODEL_SCALE = 4
BATCH_SIZE = 2
SHARPEN_GPU_AMOUNT = 0.5
SHARPEN_GPU_KERNEL_SIZE = 3
SHARPEN_GPU_SIGMA = 1.0
SATURATION = 1.20
CONTRAST = 1.05
BRIGHTNESS = 0.0
PREPROCESS_VF = "yadif,hqdn3d=4:3:6:4"
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
    ) -> None:
        self._model_path = model_path
        self._hurrdeblur_model_path = hurrdeblur_model_path
        self._target_height = target_height
        self._encode_preset = encode_preset
        self._enable_torch_compile = enable_torch_compile
        self._runtime: PipelineRuntime | None = None
        self._runtime_lock = threading.Lock()

    def process(
        self,
        input_path: Path,
        output_path: Path,
        target_height: int | None = None,
        batch_size: int | None = None,
        sharpen: float | None = None,
        saturation: float | None = None,
        contrast: float | None = None,
    ) -> None:
        encode_params = EncodeParams(
            sharpen=max(0.0, min(2.0, sharpen)) if sharpen is not None else SHARPEN_GPU_AMOUNT,
            saturation=saturation if saturation is not None else SATURATION,
            contrast=contrast if contrast is not None else CONTRAST,
        )
        effective_height = target_height or self._target_height
        effective_batch = max(1, min(16, batch_size)) if batch_size is not None else BATCH_SIZE
        logging.info(
            "gpu_optimized pipeline: target=%dp batch=%d sharpen=%.2f saturation=%.2f contrast=%.2f",
            effective_height, effective_batch, encode_params.sharpen, encode_params.saturation, encode_params.contrast,
        )
        metadata = self._probe_video(input_path, effective_height)
        runtime = self._load_runtime()
        self._run_stream(input_path, output_path, metadata, runtime, encode_params, effective_batch)

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
            if path.exists():
                continue
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

    def _run_stream(
        self,
        input_path: Path,
        output_path: Path,
        metadata: VideoMetadata,
        runtime: PipelineRuntime,
        encode_params: EncodeParams | None = None,
        batch_size: int = BATCH_SIZE,
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

        decoder.start()
        encoder.start()

        while True:
            item = decode_queue.get()
            if item is sentinel:
                break

            pending_images.append(item)
            handled += 1

            if len(pending_images) < batch_size:
                continue

            self._write_batch(encode_queue, pending_images, runtime, metadata, encode_params)
            pending_images = []
            self._log_progress(handled, expected_frames, started)

        if pending_images:
            self._write_batch(encode_queue, pending_images, runtime, metadata, encode_params)

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
    ) -> None:
        frames = self._upscale_batch(pending_images, runtime, metadata, encode_params)
        for frame in frames:
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
    ) -> list[object]:
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
        return self._decode_gpu_optimized_frames(output, sizes, runtime, metadata, sharpen_amount)

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
    ) -> list[str]:
        params = encode_params or EncodeParams(saturation=SATURATION, contrast=CONTRAST)
        encode_vf = (
            f"eq=saturation={params.saturation}:contrast={params.contrast}:brightness={BRIGHTNESS}"
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
            f"{metadata.fps:.6f}",
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
