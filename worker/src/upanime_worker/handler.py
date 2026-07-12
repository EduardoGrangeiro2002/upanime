from __future__ import annotations

import logging
from pathlib import Path

from .callbacks import CallbackClient
from .config import load_settings
from .models import WorkerJobRequest
from .service import UpscaleJobRunner

logging.basicConfig(level=logging.INFO)


def _build_runner() -> UpscaleJobRunner:
    from .quality_pipeline import QualityUpscalePipeline
    from .r2_storage import R2StorageClient

    settings = load_settings()

    pipeline = QualityUpscalePipeline(
        model_path=settings.model_path,
        hurrdeblur_model_path=settings.hurrdeblur_model_path,
        target_height=settings.target_height,
        encode_preset=settings.encode_preset,
        enable_torch_compile=settings.enable_torch_compile,
        rife_dir=settings.rife_dir,
    )
    storage = R2StorageClient(
        account_id=settings.r2_account_id,
        access_key_id=settings.r2_access_key_id,
        access_secret=settings.r2_access_secret,
        bucket_name=settings.r2_bucket_name,
    )
    callbacks = CallbackClient(timeout_seconds=settings.callback_timeout_seconds)

    return UpscaleJobRunner(
        pipeline=pipeline,
        storage=storage,
        callbacks=callbacks,
        temp_root=settings.temp_root,
        request_timeout_seconds=settings.request_timeout_seconds,
        force_interpolate=settings.force_interpolate,
    )


_runner: UpscaleJobRunner | None = None


def _get_runner() -> UpscaleJobRunner:
    global _runner
    if _runner is not None:
        return _runner
    _runner = _build_runner()
    return _runner


def handler(job: dict) -> dict:
    job_input = job["input"]
    request = WorkerJobRequest(**job_input)
    _get_runner().run(request)
    return {"status": "completed", "resultStorageKey": request.result_storage_key}


if __name__ == "__main__":
    import runpod

    _runner = _build_runner()
    runpod.serverless.start({"handler": handler})
