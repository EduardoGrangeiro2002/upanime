from __future__ import annotations

import email
import io
import random
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import cv2
import numpy as np
from PIL import Image

from upanime_teacher.service import TeacherIngestService
from upanime_teacher.sink import TeacherSample, TriageAPISink
from upanime_teacher.teacher import Proposal

from ..unit.test_teacher_unit import StubTagger, make_settings, write_video


class CaptureHandler(BaseHTTPRequestHandler):
    received: list[dict] = []
    fail_next = False

    def do_POST(self) -> None:
        if CaptureHandler.fail_next:
            CaptureHandler.fail_next = False
            self.send_response(500)
            self.end_headers()
            return

        length = int(self.headers["Content-Length"])
        body = self.rfile.read(length)
        message = email.message_from_bytes(
            b"Content-Type: " + self.headers["Content-Type"].encode() + b"\r\n\r\n" + body
        )
        fields = {}
        files = {}
        for part in message.get_payload():
            name = part.get_param("name", header="content-disposition")
            if part.get_filename():
                files[name] = part.get_payload(decode=True)
                continue
            fields[name] = part.get_payload(decode=True).decode()

        CaptureHandler.received.append({
            "auth": self.headers.get("Authorization", ""),
            "fields": fields,
            "files": files,
        })
        self.send_response(201)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(b'{"id":"1","status":"pending"}')

    def log_message(self, format: str, *args: object) -> None:
        return


def start_capture_server() -> tuple[ThreadingHTTPServer, str]:
    CaptureHandler.received = []
    server = ThreadingHTTPServer(("127.0.0.1", 0), CaptureHandler)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    host, port = server.server_address
    return server, f"http://{host}:{port}"


def test_sink_posts_multipart_with_token():
    server, base = start_capture_server()
    try:
        sink = TriageAPISink(base, "meu-token", timeout_seconds=5)
        frame = np.full((48, 64, 3), 120, dtype=np.uint8)
        mask = np.zeros((48, 64), dtype=bool)
        mask[10:20, 10:20] = True

        sink.send(TeacherSample(
            class_name="fire", frame_bgr=frame, mask=mask,
            anime_title="Slayers", episode="S1E04",
            timestamp_s=54.3, teacher_prob=0.42, source="teacher",
        ))

        assert len(CaptureHandler.received) == 1
        request = CaptureHandler.received[0]
        assert request["auth"] == "Bearer meu-token"
        assert request["fields"]["class"] == "fire"
        assert request["fields"]["animeTitle"] == "Slayers"
        assert request["fields"]["timestampS"] == "54.300"
        assert request["fields"]["teacherProb"] == "0.4200"

        decoded_mask = np.array(Image.open(io.BytesIO(request["files"]["mask"])))
        assert decoded_mask[15, 15, 3] == 255
        assert decoded_mask[0, 0, 3] == 0
        decoded_frame = cv2.imdecode(
            np.frombuffer(request["files"]["frame"], np.uint8), cv2.IMREAD_COLOR
        )
        assert decoded_frame.shape == (48, 64, 3)
    finally:
        server.shutdown()


def test_sink_raises_on_server_error():
    server, base = start_capture_server()
    try:
        CaptureHandler.fail_next = True
        sink = TriageAPISink(base, "t", timeout_seconds=5)
        frame = np.full((16, 16, 3), 10, dtype=np.uint8)

        try:
            sink.send(TeacherSample("fire", frame, None, "A", "E", 0.0, 0.0, "teacher"))
            raised = False
        except Exception:
            raised = True
        assert raised
    finally:
        server.shutdown()


class BlobTeacher:
    def propose(self, frame_bgr: np.ndarray) -> list[Proposal]:
        bright = frame_bgr[:, :, 2] > 200
        if not bright.any():
            return []
        return [Proposal(class_name="fire", mask=bright, score=0.9, origin="teacher")]


def write_effect_video(path, frame_count: int) -> None:
    writer = cv2.VideoWriter(str(path), cv2.VideoWriter_fourcc(*"mp4v"), 8.0, (64, 48))
    for i in range(frame_count):
        frame = np.full((48, 64, 3), 30, dtype=np.uint8)
        if i >= frame_count // 2:
            frame[10:30, 20:50] = (20, 20, 255)
        writer.write(frame)
    writer.release()


def test_service_end_to_end_posts_samples_and_negatives(tmp_path):
    server, base = start_capture_server()
    try:
        video = tmp_path / "clip.mp4"
        write_effect_video(video, 16)

        settings = make_settings(sample_fps=8.0, wd14_threshold=0.0, negative_keep=1.0)
        service = TeacherIngestService(
            teacher=BlobTeacher(),
            tagger=StubTagger([1.0]),
            sink=TriageAPISink(base, "token", timeout_seconds=5),
            settings=settings,
            rng=random.Random(3),
        )

        stats = service.run(video, "Slayers", "S1E04")

        assert stats["candidates"] == 16
        assert stats["sent"] == len(CaptureHandler.received)
        assert stats["negatives"] == stats["by_class"].get("none", 0)
        classes = {r["fields"]["class"] for r in CaptureHandler.received}
        assert classes == {"fire", "none"}
    finally:
        server.shutdown()


def test_service_respects_max_samples(tmp_path):
    server, base = start_capture_server()
    try:
        video = tmp_path / "clip.mp4"
        write_video(video, 16)

        settings = make_settings(sample_fps=8.0, wd14_threshold=0.0, negative_keep=1.0, max_samples=3)
        service = TeacherIngestService(
            teacher=BlobTeacher(),
            tagger=StubTagger([1.0]),
            sink=TriageAPISink(base, "token", timeout_seconds=5),
            settings=settings,
            rng=random.Random(3),
        )

        stats = service.run(video, "Slayers", "S1E04")

        assert stats["sent"] == 3
    finally:
        server.shutdown()
