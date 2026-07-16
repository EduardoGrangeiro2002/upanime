from __future__ import annotations

import logging
import shutil
from pathlib import Path
from typing import Protocol

import requests

from .callbacks import CallbackClient, build_failure_callback, build_success_callback
from .downscale import downscale_video
from .models import WorkerJobRequest


class VideoPipeline(Protocol):
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
        dataset_dir: Path | None = None,
    ) -> None: ...


class ObjectStorage(Protocol):
    def upload_file(self, source_path: Path, storage_key: str) -> None: ...
    def exists(self, storage_key: str) -> bool: ...


class UpscaleJobRunner:
    def __init__(
        self,
        pipeline: VideoPipeline,
        storage: ObjectStorage,
        callbacks: CallbackClient,
        temp_root: Path,
        request_timeout_seconds: int,
        force_interpolate: bool = False,
        encode_preset: str = "medium",
    ) -> None:
        self._pipeline = pipeline
        self._storage = storage
        self._callbacks = callbacks
        self._temp_root = temp_root
        self._request_timeout_seconds = request_timeout_seconds
        self._force_interpolate = force_interpolate
        self._encode_preset = encode_preset

    def run(self, job: WorkerJobRequest) -> list[int]:
        work_dir = self._temp_root / str(job.job_id)
        input_path = work_dir / "input.mp4"
        output_path = work_dir / "output.mp4"
        dataset_dir = work_dir / "dataset" if job.effects else None

        try:
            self._prepare_work_dir(work_dir)
            self._download_source(str(job.source_url), input_path)
            self._pipeline.process(
                input_path,
                output_path,
                target_height=job.target_height,
                batch_size=job.batch_size,
                sharpen=job.sharpen,
                saturation=job.saturation,
                contrast=job.contrast,
                interpolate=job.interpolate or self._force_interpolate,
                pan_residual_ratio=job.pan_ratio,
                effects=job.effects,
                effects_strength=job.effects_strength,
                effects_sensitivity=job.effects_sensitivity,
                skip_upscale=job.skip_upscale,
                dataset_dir=dataset_dir,
            )
            self._storage.upload_file(output_path, job.result_storage_key)
            self._ensure_uploaded(job.result_storage_key)
            self._upload_dataset(dataset_dir, job.job_id)
            uploaded_heights = self._process_variants(job, output_path, work_dir)
        except Exception as exc:
            logging.exception("worker job %s failed", job.job_id)
            self._notify_failure(job, str(exc))
            return []
        finally:
            self._cleanup(work_dir)

        self._notify_success(job)
        return uploaded_heights

    def _process_variants(self, job: WorkerJobRequest, output_path: Path, work_dir: Path) -> list[int]:
        if job.skip_upscale:
            return []
        uploaded: list[int] = []
        for variant in job.variants:
            variant_path = work_dir / f"variant_{variant.height}p.mp4"
            try:
                downscale_video(output_path, variant_path, variant.height, self._encode_preset)
                self._storage.upload_file(variant_path, variant.storage_key)
                self._ensure_uploaded(variant.storage_key)
            except Exception:
                logging.exception("variant %dp failed for job %d (main result unaffected)", variant.height, job.job_id)
                continue
            uploaded.append(variant.height)
        return uploaded

    def _prepare_work_dir(self, work_dir: Path) -> None:
        if work_dir.exists():
            shutil.rmtree(work_dir)
        work_dir.mkdir(parents=True, exist_ok=True)

    def _download_source(self, source_url: str, input_path: Path) -> None:
        response = requests.get(source_url, stream=True, timeout=self._request_timeout_seconds)
        response.raise_for_status()
        with input_path.open("wb") as destination:
            for chunk in response.iter_content(chunk_size=1024 * 1024):
                if not chunk:
                    continue
                destination.write(chunk)

    def _ensure_uploaded(self, storage_key: str) -> None:
        if self._storage.exists(storage_key):
            return
        raise RuntimeError(f"uploaded file not found in storage: {storage_key}")

    def _upload_dataset(self, dataset_dir: Path | None, job_id: int) -> None:
        if dataset_dir is None or not dataset_dir.exists():
            return
        try:
            for sample in sorted(dataset_dir.iterdir()):
                self._storage.upload_file(sample, f"datasets/effects/{job_id}/{sample.name}")
        except Exception:
            logging.exception("dataset upload failed for job %d (job result unaffected)", job_id)

    def _notify_success(self, job: WorkerJobRequest) -> None:
        if not job.callback_url:
            return
        payload = build_success_callback(job.job_id, job.result_storage_key)
        try:
            self._callbacks.notify(str(job.callback_url), payload)
        except Exception:
            logging.exception("worker job %s uploaded successfully but callback failed", job.job_id)

    def _notify_failure(self, job: WorkerJobRequest, error: str) -> None:
        if not job.callback_url:
            return
        payload = build_failure_callback(job.job_id, error, job.result_storage_key)
        try:
            self._callbacks.notify(str(job.callback_url), payload)
        except Exception:
            logging.exception("worker job %s failed and failure callback could not be delivered", job.job_id)

    def _cleanup(self, work_dir: Path) -> None:
        if not work_dir.exists():
            return
        shutil.rmtree(work_dir)
