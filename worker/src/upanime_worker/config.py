from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class WorkerSettings:
    temp_root: Path
    model_path: Path
    hurrdeblur_model_path: Path
    rife_dir: Path
    target_height: int
    encode_preset: str
    temporal_smooth: bool
    r2_account_id: str
    r2_access_key_id: str
    r2_access_secret: str
    r2_bucket_name: str
    request_timeout_seconds: int
    callback_timeout_seconds: int
    enable_torch_compile: bool


def load_settings() -> WorkerSettings:
    return WorkerSettings(
        temp_root=Path(os.getenv("WORKER_TEMP_ROOT", "/tmp/upanime-worker")),
        model_path=Path(os.getenv("WORKER_MODEL_PATH", "./models/realesr-animevideov3.pth")),
        hurrdeblur_model_path=Path(os.getenv("WORKER_HURRDEBLUR_MODEL_PATH", "./models/1x-HurrDeblur-SuperUltraCompact.pth")),
        rife_dir=Path(os.getenv("WORKER_RIFE_DIR", "./models/Practical-RIFE")),
        target_height=int(os.getenv("WORKER_TARGET_HEIGHT", "1080")),
        encode_preset=os.getenv("WORKER_ENCODE_PRESET", "medium"),
        temporal_smooth=os.getenv("WORKER_TEMPORAL_SMOOTH", "1") == "1",
        r2_account_id=os.getenv("R2_ACCOUNT_ID", ""),
        r2_access_key_id=os.getenv("R2_ACCESS_KEY_ID", ""),
        r2_access_secret=os.getenv("R2_ACCESS_SECRET", ""),
        r2_bucket_name=os.getenv("R2_BUCKET_NAME", ""),
        request_timeout_seconds=int(os.getenv("WORKER_REQUEST_TIMEOUT_SECONDS", "1800")),
        callback_timeout_seconds=int(os.getenv("WORKER_CALLBACK_TIMEOUT_SECONDS", "30")),
        enable_torch_compile=os.getenv("WORKER_ENABLE_TORCH_COMPILE", "1") == "1",
    )
