from __future__ import annotations

import json
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

from upanime_worker.callbacks import CallbackClient
from upanime_worker.handler import handler
from upanime_worker.models import WorkerJobRequest
from upanime_worker.service import UpscaleJobRunner


class CopyPipeline:
    def __init__(self, fail: bool = False) -> None:
        self.fail = fail
        self.heights: list[int] = []

    def process(self, input_path: Path, output_path: Path, **kwargs) -> None:
        self.heights.append(kwargs.get("target_height", 1080))
        self.last_kwargs = kwargs
        if self.fail:
            raise RuntimeError("pipeline failed")
        output_path.write_bytes(input_path.read_bytes())


class FakeStorage:
    def __init__(self) -> None:
        self.files: dict[str, bytes] = {}

    def upload_file(self, source_path: Path, storage_key: str) -> None:
        self.files[storage_key] = source_path.read_bytes()

    def exists(self, storage_key: str) -> bool:
        return storage_key in self.files


class FailingCallbackClient:
    def notify(self, callback_url: str, payload: object) -> None:
        raise RuntimeError("callback offline")


class CallbackServerHandler(BaseHTTPRequestHandler):
    video_bytes = b"fake-mp4"
    callbacks: list[dict[str, object]] = []

    def do_GET(self) -> None:
        if self.path != "/video.mp4":
            self.send_response(404)
            self.end_headers()
            return
        self.send_response(200)
        self.send_header("Content-Type", "video/mp4")
        self.send_header("Content-Length", str(len(self.video_bytes)))
        self.end_headers()
        self.wfile.write(self.video_bytes)

    def do_POST(self) -> None:
        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length)
        self.callbacks.append(json.loads(body))
        self.send_response(204)
        self.end_headers()

    def log_message(self, format: str, *args: object) -> None:
        return


def start_test_server() -> tuple[ThreadingHTTPServer, str]:
    server = ThreadingHTTPServer(("127.0.0.1", 0), CallbackServerHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    host, port = server.server_address
    return server, f"http://{host}:{port}"


def test_upscale_job_runner_success(tmp_path):
    CallbackServerHandler.callbacks = []
    server, base_url = start_test_server()

    try:
        storage = FakeStorage()
        pipeline = CopyPipeline()
        runner = UpscaleJobRunner(
            pipeline=pipeline,
            storage=storage,
            callbacks=CallbackClient(timeout_seconds=5),
            temp_root=tmp_path,
            request_timeout_seconds=5,
        )
        job = WorkerJobRequest(
            jobId=11,
            sourceUrl=f"{base_url}/video.mp4",
            sourceStorageKey="animes/test/source.mp4",
            resultStorageKey="animes/test/source_upscaled.mp4",
            targetHeight=1440,
            callbackUrl=f"{base_url}/callback",
        )

        runner.run(job)

        assert storage.exists("animes/test/source_upscaled.mp4")
        assert pipeline.heights == [1440]
        assert CallbackServerHandler.callbacks == [{
            "jobId": 11,
            "status": "completed",
            "error": "",
            "resultStorageKey": "animes/test/source_upscaled.mp4",
            "fileName": "source_upscaled.mp4",
        }]
    finally:
        server.shutdown()


def test_upscale_job_runner_passes_effects_debug_flags(tmp_path):
    CallbackServerHandler.callbacks = []
    server, base_url = start_test_server()

    try:
        storage = FakeStorage()
        pipeline = CopyPipeline()
        runner = UpscaleJobRunner(
            pipeline=pipeline,
            storage=storage,
            callbacks=CallbackClient(timeout_seconds=5),
            temp_root=tmp_path,
            request_timeout_seconds=5,
        )
        job = WorkerJobRequest(
            jobId=14,
            sourceUrl=f"{base_url}/video.mp4",
            sourceStorageKey="animes/test/source.mp4",
            resultStorageKey="animes/test/source_upscaled.mp4",
            effects=True,
            skipUpscale=True,
            callbackUrl=f"{base_url}/callback",
        )

        runner.run(job)

        assert storage.exists("animes/test/source_upscaled.mp4")
        assert pipeline.last_kwargs["effects"] is True
        assert pipeline.last_kwargs["skip_upscale"] is True
        assert pipeline.last_kwargs["dataset_dir"] == tmp_path / "14" / "dataset"
    finally:
        server.shutdown()


def test_upscale_job_runner_uploads_effects_log_with_dataset(tmp_path):
    CallbackServerHandler.callbacks = []
    server, base_url = start_test_server()

    class LogWritingPipeline(CopyPipeline):
        def process(self, input_path: Path, output_path: Path, **kwargs) -> None:
            super().process(input_path, output_path, **kwargs)
            dataset_dir = kwargs["dataset_dir"]
            dataset_dir.mkdir(parents=True, exist_ok=True)
            (dataset_dir / "effects_log.json").write_text(
                json.dumps({"mode": "stream", "fps": 24.0, "entries": []})
            )

    try:
        storage = FakeStorage()
        runner = UpscaleJobRunner(
            pipeline=LogWritingPipeline(),
            storage=storage,
            callbacks=CallbackClient(timeout_seconds=5),
            temp_root=tmp_path,
            request_timeout_seconds=5,
        )
        job = WorkerJobRequest(
            jobId=15,
            sourceUrl=f"{base_url}/video.mp4",
            sourceStorageKey="animes/test/source.mp4",
            resultStorageKey="animes/test/source_upscaled.mp4",
            effects=True,
            callbackUrl=f"{base_url}/callback",
        )

        runner.run(job)

        assert storage.exists("datasets/effects/15/effects_log.json")
        payload = json.loads(storage.files["datasets/effects/15/effects_log.json"])
        assert payload["mode"] == "stream"
    finally:
        server.shutdown()


def test_upscale_job_runner_failure(tmp_path):
    CallbackServerHandler.callbacks = []
    server, base_url = start_test_server()

    try:
        storage = FakeStorage()
        runner = UpscaleJobRunner(
            pipeline=CopyPipeline(fail=True),
            storage=storage,
            callbacks=CallbackClient(timeout_seconds=5),
            temp_root=tmp_path,
            request_timeout_seconds=5,
        )
        job = WorkerJobRequest(
            jobId=12,
            sourceUrl=f"{base_url}/video.mp4",
            sourceStorageKey="animes/test/source.mp4",
            resultStorageKey="animes/test/source_upscaled.mp4",
            targetHeight=1080,
            callbackUrl=f"{base_url}/callback",
        )

        runner.run(job)

        assert not storage.exists("animes/test/source_upscaled.mp4")
        assert CallbackServerHandler.callbacks == [{
            "jobId": 12,
            "status": "failed",
            "error": "pipeline failed",
            "resultStorageKey": "animes/test/source_upscaled.mp4",
            "fileName": "source_upscaled.mp4",
        }]
    finally:
        server.shutdown()


def test_upscale_job_runner_swallows_failure_callback_errors(tmp_path):
    server, base_url = start_test_server()

    try:
        storage = FakeStorage()
        runner = UpscaleJobRunner(
            pipeline=CopyPipeline(fail=True),
            storage=storage,
            callbacks=FailingCallbackClient(),
            temp_root=tmp_path,
            request_timeout_seconds=5,
        )
        job = WorkerJobRequest(
            jobId=13,
            sourceUrl=f"{base_url}/video.mp4",
            sourceStorageKey="animes/test/source.mp4",
            resultStorageKey="animes/test/source_upscaled.mp4",
            targetHeight=1080,
            callbackUrl=f"{base_url}/callback",
        )

        runner.run(job)

        assert not storage.exists("animes/test/source_upscaled.mp4")
    finally:
        server.shutdown()


def test_handler_end_to_end(tmp_path, monkeypatch):
    CallbackServerHandler.callbacks = []
    server, base_url = start_test_server()

    try:
        storage = FakeStorage()
        pipeline = CopyPipeline()
        runner = UpscaleJobRunner(
            pipeline=pipeline,
            storage=storage,
            callbacks=CallbackClient(timeout_seconds=5),
            temp_root=tmp_path,
            request_timeout_seconds=5,
        )

        import upanime_worker.handler as handler_mod
        monkeypatch.setattr(handler_mod, "_runner", runner)

        result = handler({
            "input": {
                "jobId": 99,
                "sourceUrl": f"{base_url}/video.mp4",
                "sourceStorageKey": "animes/test/source.mp4",
                "resultStorageKey": "animes/test/source_upscaled.mp4",
                "targetHeight": 2160,
                "callbackUrl": f"{base_url}/callback",
            }
        })

        assert result == {
            "status": "completed",
            "resultStorageKey": "animes/test/source_upscaled.mp4",
        }
        assert storage.exists("animes/test/source_upscaled.mp4")
        assert pipeline.heights == [2160]
        assert CallbackServerHandler.callbacks == [{
            "jobId": 99,
            "status": "completed",
            "error": "",
            "resultStorageKey": "animes/test/source_upscaled.mp4",
            "fileName": "source_upscaled.mp4",
        }]
    finally:
        server.shutdown()
