from pathlib import Path
from unittest.mock import MagicMock

import upanime_worker.service as service_module
from upanime_worker.downscale import build_downscale_command
from upanime_worker.models import WorkerJobRequest
from upanime_worker.service import UpscaleJobRunner


def test_build_downscale_command_scales_to_height():
    cmd = build_downscale_command(Path("/in.mp4"), Path("/out.mp4"), 1440, "medium")

    assert cmd[0] == "ffmpeg"
    assert "scale=-2:1440:flags=lanczos" in cmd
    assert "libx264" in cmd
    assert "+faststart" in cmd
    assert cmd[-1] == "/out.mp4"


def _make_runner(storage=None) -> UpscaleJobRunner:
    return UpscaleJobRunner(
        pipeline=MagicMock(),
        storage=storage or MagicMock(),
        callbacks=MagicMock(),
        temp_root=Path("/tmp/worker-test"),
        request_timeout_seconds=10,
        encode_preset="fast",
    )


def _make_job(**overrides) -> WorkerJobRequest:
    base = {
        "jobId": 7,
        "sourceUrl": "https://example.com/source.mp4",
        "sourceStorageKey": "animes/test/s01e01.mp4",
        "resultStorageKey": "animes/test/s01e01_upscaled.mp4",
        "targetHeight": 2160,
        "variants": [
            {"height": 1440, "storageKey": "animes/test/s01e01_upscaled_1440p.mp4"},
            {"height": 1080, "storageKey": "animes/test/s01e01_upscaled_1080p.mp4"},
        ],
    }
    base.update(overrides)
    return WorkerJobRequest(**base)


def test_process_variants_uploads_each_variant(monkeypatch, tmp_path):
    calls = []
    monkeypatch.setattr(service_module, "downscale_video", lambda *a, **k: calls.append(a))
    storage = MagicMock()
    storage.exists.return_value = True
    runner = _make_runner(storage)

    uploaded = runner._process_variants(_make_job(), tmp_path / "output.mp4", tmp_path)

    assert uploaded == [1440, 1080]
    assert len(calls) == 2
    uploaded_keys = [c.args[1] for c in storage.upload_file.call_args_list]
    assert uploaded_keys == [
        "animes/test/s01e01_upscaled_1440p.mp4",
        "animes/test/s01e01_upscaled_1080p.mp4",
    ]


def test_process_variants_skips_when_skip_upscale(monkeypatch, tmp_path):
    monkeypatch.setattr(service_module, "downscale_video", MagicMock())
    runner = _make_runner()

    uploaded = runner._process_variants(
        _make_job(skipUpscale=True), tmp_path / "output.mp4", tmp_path
    )

    assert uploaded == []


def test_process_variants_continues_after_failure(monkeypatch, tmp_path):
    def failing_first(_input, _output, height, _preset):
        if height == 1440:
            raise RuntimeError("encode failed")

    monkeypatch.setattr(service_module, "downscale_video", failing_first)
    storage = MagicMock()
    storage.exists.return_value = True
    runner = _make_runner(storage)

    uploaded = runner._process_variants(_make_job(), tmp_path / "output.mp4", tmp_path)

    assert uploaded == [1080]
