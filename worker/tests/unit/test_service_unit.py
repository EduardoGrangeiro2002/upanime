from unittest.mock import MagicMock

import pytest

from upanime_worker.handler import handler
from upanime_worker.models import WorkerJobRequest


class RunnerSpy:
    def __init__(self, uploaded_heights: list[int] | None = None) -> None:
        self.jobs: list[WorkerJobRequest] = []
        self.uploaded_heights = uploaded_heights or []

    def run(self, job: WorkerJobRequest) -> list[int]:
        self.jobs.append(job)
        return self.uploaded_heights


def _make_job_input(**overrides) -> dict:
    base = {
        "jobId": 1,
        "sourceUrl": "https://example.com/source.mp4",
        "sourceStorageKey": "animes/test/source.mp4",
        "resultStorageKey": "animes/test/source_upscaled.mp4",
        "targetHeight": 1080,
        "callbackUrl": "https://example.com/callback",
    }
    base.update(overrides)
    return base


def _patch_runner(monkeypatch, runner):
    import upanime_worker.handler as mod
    monkeypatch.setattr(mod, "_runner", runner)


def test_handler_returns_completed_on_success(monkeypatch):
    spy = RunnerSpy()
    _patch_runner(monkeypatch, spy)

    result = handler({"input": _make_job_input()})

    assert result == {
        "status": "completed",
        "resultStorageKey": "animes/test/source_upscaled.mp4",
        "variantHeights": "",
    }
    assert len(spy.jobs) == 1
    assert spy.jobs[0].job_id == 1
    assert spy.jobs[0].target_height == 1080


def test_handler_returns_uploaded_variant_heights(monkeypatch):
    spy = RunnerSpy(uploaded_heights=[1440, 1080])
    _patch_runner(monkeypatch, spy)

    result = handler({"input": _make_job_input(targetHeight=2160)})

    assert result["variantHeights"] == "1440,1080"


def test_job_request_parses_variants():
    job = WorkerJobRequest(**_make_job_input(
        targetHeight=2160,
        variants=[
            {"height": 1440, "storageKey": "animes/test/source_upscaled_1440p.mp4"},
            {"height": 1080, "storageKey": "animes/test/source_upscaled_1080p.mp4"},
        ],
    ))

    assert [v.height for v in job.variants] == [1440, 1080]
    assert job.variants[0].storage_key == "animes/test/source_upscaled_1440p.mp4"


def test_job_request_defaults_to_no_variants():
    job = WorkerJobRequest(**_make_job_input())

    assert job.variants == []


def test_handler_delegates_to_runner_with_correct_fields(monkeypatch):
    spy = RunnerSpy()
    _patch_runner(monkeypatch, spy)

    handler({"input": _make_job_input(jobId=42, targetHeight=1440)})

    job = spy.jobs[0]
    assert job.job_id == 42
    assert str(job.source_url) == "https://example.com/source.mp4"
    assert job.source_storage_key == "animes/test/source.mp4"
    assert job.result_storage_key == "animes/test/source_upscaled.mp4"
    assert job.target_height == 1440
    assert str(job.callback_url) == "https://example.com/callback"


def test_handler_raises_on_invalid_input(monkeypatch):
    spy = RunnerSpy()
    _patch_runner(monkeypatch, spy)

    with pytest.raises(Exception):
        handler({"input": {"jobId": 1}})

    assert len(spy.jobs) == 0


def test_handler_propagates_runner_exception(monkeypatch):
    failing_runner = MagicMock()
    failing_runner.run.side_effect = RuntimeError("boom")
    _patch_runner(monkeypatch, failing_runner)

    with pytest.raises(RuntimeError, match="boom"):
        handler({"input": _make_job_input()})


def test_handler_rejects_invalid_target_height(monkeypatch):
    spy = RunnerSpy()
    _patch_runner(monkeypatch, spy)

    with pytest.raises(Exception):
        handler({"input": _make_job_input(targetHeight=999)})

    assert len(spy.jobs) == 0


def test_handler_accepts_all_valid_target_heights(monkeypatch):
    spy = RunnerSpy()
    _patch_runner(monkeypatch, spy)

    for height in [1080, 1440, 2160]:
        handler({"input": _make_job_input(targetHeight=height)})

    assert [j.target_height for j in spy.jobs] == [1080, 1440, 2160]
