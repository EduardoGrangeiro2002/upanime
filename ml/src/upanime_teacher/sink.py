from __future__ import annotations

import io
from dataclasses import dataclass
from typing import Protocol

import cv2
import numpy as np
import requests
from PIL import Image


@dataclass
class TeacherSample:
    class_name: str
    frame_bgr: np.ndarray
    mask: np.ndarray | None
    anime_title: str
    episode: str
    timestamp_s: float
    teacher_prob: float
    source: str


class SampleSink(Protocol):
    def send(self, sample: TeacherSample) -> None: ...


def mask_to_png_bytes(mask: np.ndarray | None, shape: tuple[int, int]) -> bytes:
    height, width = shape
    rgba = np.zeros((height, width, 4), dtype=np.uint8)
    if mask is not None:
        rgba[mask] = (255, 255, 255, 255)
    buffer = io.BytesIO()
    Image.fromarray(rgba, mode="RGBA").save(buffer, format="PNG")
    return buffer.getvalue()


class TriageAPISink:
    def __init__(self, api_base: str, token: str, timeout_seconds: int) -> None:
        self._url = f"{api_base}/api/dataset/samples"
        self._headers = {"Authorization": f"Bearer {token}"}
        self._timeout = timeout_seconds

    def send(self, sample: TeacherSample) -> None:
        ok, frame_jpg = cv2.imencode(".jpg", sample.frame_bgr, [cv2.IMWRITE_JPEG_QUALITY, 90])
        if not ok:
            raise RuntimeError("frame jpg encode failed")
        mask_png = mask_to_png_bytes(sample.mask, sample.frame_bgr.shape[:2])

        response = requests.post(
            self._url,
            headers=self._headers,
            data={
                "class": sample.class_name,
                "source": sample.source,
                "animeTitle": sample.anime_title,
                "episode": sample.episode,
                "timestampS": f"{sample.timestamp_s:.3f}",
                "teacherProb": f"{sample.teacher_prob:.4f}",
            },
            files={
                "frame": ("frame.jpg", frame_jpg.tobytes(), "image/jpeg"),
                "mask": ("mask.png", mask_png, "image/png"),
            },
            timeout=self._timeout,
        )
        response.raise_for_status()
