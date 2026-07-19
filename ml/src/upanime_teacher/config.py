from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class Settings:
    api_base: str
    api_token: str
    device: str
    sample_fps: float
    wd14_threshold: float
    random_keep: float
    negative_keep: float
    max_samples: int
    dino_threshold: float
    request_timeout_seconds: int
    tagger_model_path: Path
    tagger_tags_path: Path


def load_settings() -> Settings:
    api_base = os.getenv("TEACHER_API_BASE", "")
    if not api_base:
        raise RuntimeError("TEACHER_API_BASE is required")
    api_token = os.getenv("TEACHER_API_TOKEN", "")
    if not api_token:
        raise RuntimeError("TEACHER_API_TOKEN is required")

    return Settings(
        api_base=api_base.rstrip("/"),
        api_token=api_token,
        device=os.getenv("TEACHER_DEVICE", "auto"),
        sample_fps=float(os.getenv("TEACHER_SAMPLE_FPS", "1.0")),
        wd14_threshold=float(os.getenv("TEACHER_WD14_THRESHOLD", "0.05")),
        random_keep=float(os.getenv("TEACHER_RANDOM_KEEP", "0.075")),
        negative_keep=float(os.getenv("TEACHER_NEGATIVE_KEEP", "0.33")),
        max_samples=int(os.getenv("TEACHER_MAX_SAMPLES", "400")),
        dino_threshold=float(os.getenv("TEACHER_DINO_THRESHOLD", "0.2")),
        request_timeout_seconds=int(os.getenv("TEACHER_REQUEST_TIMEOUT_SECONDS", "120")),
        tagger_model_path=Path(os.getenv("TEACHER_TAGGER_MODEL_PATH", "./checkpoints/wd-vit-tagger-v3.onnx")),
        tagger_tags_path=Path(os.getenv("TEACHER_TAGGER_TAGS_PATH", "./checkpoints/wd-vit-tagger-v3-tags.csv")),
    )


def resolve_device(device: str) -> str:
    if device != "auto":
        return device
    import torch

    if torch.cuda.is_available():
        return "cuda"
    return "cpu"
