from __future__ import annotations

from pathlib import Path
from types import SimpleNamespace

import pytest
import torch

from upanime_worker.models import WorkerJobRequest
from upanime_worker.quality_pipeline import MODEL_SCALE, QualityUpscalePipeline


def make_pipeline(apisr_model_path: Path | None = None) -> QualityUpscalePipeline:
    return QualityUpscalePipeline(
        model_path=Path("/nonexistent/model.pth"),
        hurrdeblur_model_path=Path("/nonexistent/deblur.pth"),
        target_height=1080,
        encode_preset="medium",
        enable_torch_compile=False,
        apisr_model_path=apisr_model_path,
    )


def make_job(**overrides) -> WorkerJobRequest:
    base = {
        "jobId": 1,
        "sourceUrl": "https://example.com/source.mp4",
        "sourceStorageKey": "animes/test/s01e01.mp4",
        "resultStorageKey": "animes/test/s01e01_upscaled.mp4",
    }
    return WorkerJobRequest(**{**base, **overrides})


def test_upscaler_accepts_known_values():
    assert make_job().upscaler is None
    assert make_job(upscaler="apisr").upscaler == "apisr"
    assert make_job(upscaler="compact").upscaler == "compact"


def test_upscaler_rejects_unknown_value():
    with pytest.raises(ValueError):
        make_job(upscaler="waifu2x")


def test_load_apisr_requires_configured_path():
    runtime = SimpleNamespace(torch=torch, device=torch.device("cpu"), use_half=False)
    with pytest.raises(RuntimeError, match="WORKER_APISR_MODEL_PATH"):
        make_pipeline()._load_apisr(runtime)


class OomBelowThreshold:
    def __init__(self, max_pixels: int) -> None:
        self.max_pixels = max_pixels

    def __call__(self, tensor: torch.Tensor) -> torch.Tensor:
        _, _, height, width = tensor.shape
        if height * width > self.max_pixels:
            raise torch.cuda.OutOfMemoryError("fake oom")
        return torch.nn.functional.interpolate(tensor, scale_factor=MODEL_SCALE, mode="nearest")


def test_apisr_infer_oom_split_matches_full_frame():
    pipeline = make_pipeline()
    runtime = SimpleNamespace(torch=torch)
    tensor = torch.rand(1, 3, 1024, 512)
    expected = torch.nn.functional.interpolate(tensor, scale_factor=MODEL_SCALE, mode="nearest")

    result = pipeline._apisr_infer(OomBelowThreshold(100000), tensor, runtime)

    assert result.shape == expected.shape
    assert torch.equal(result, expected)


def test_apisr_infer_reraises_when_tile_already_small():
    pipeline = make_pipeline()
    runtime = SimpleNamespace(torch=torch)
    tensor = torch.rand(1, 3, 64, 64)

    with pytest.raises(torch.cuda.OutOfMemoryError):
        pipeline._apisr_infer(OomBelowThreshold(0), tensor, runtime)


class OomAboveBatchOne:
    def __call__(self, tensor: torch.Tensor) -> torch.Tensor:
        if tensor.shape[0] > 1:
            raise torch.cuda.OutOfMemoryError("fake batch oom")
        return torch.nn.functional.interpolate(tensor, scale_factor=MODEL_SCALE, mode="nearest")


def test_apisr_infer_splits_batch_on_oom():
    pipeline = make_pipeline()
    runtime = SimpleNamespace(torch=torch)
    tensor = torch.rand(4, 3, 300, 300)
    expected = torch.nn.functional.interpolate(tensor, scale_factor=MODEL_SCALE, mode="nearest")

    result = pipeline._apisr_infer(OomAboveBatchOne(), tensor, runtime)

    assert torch.equal(result, expected)


def test_timed_accumulates_stage_seconds():
    pipeline = make_pipeline()
    with pipeline._timed("model"):
        pass
    with pipeline._timed("model"):
        pass
    assert pipeline._stage_seconds["model"] >= 0.0
    assert len(pipeline._stage_seconds) == 1
