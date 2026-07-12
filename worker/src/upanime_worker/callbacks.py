from __future__ import annotations

from pathlib import Path

import requests

from .models import WorkerCallbackPayload


def build_success_callback(job_id: int, result_storage_key: str) -> WorkerCallbackPayload:
    return WorkerCallbackPayload(
        job_id=job_id,
        status="completed",
        result_storage_key=result_storage_key,
        file_name=Path(result_storage_key).name,
    )


def build_failure_callback(job_id: int, error: str, result_storage_key: str) -> WorkerCallbackPayload:
    return WorkerCallbackPayload(
        job_id=job_id,
        status="failed",
        error=error,
        result_storage_key=result_storage_key,
        file_name=Path(result_storage_key).name,
    )


class CallbackClient:
    def __init__(self, timeout_seconds: int) -> None:
        self._timeout_seconds = timeout_seconds

    def notify(self, callback_url: str, payload: WorkerCallbackPayload) -> None:
        response = requests.post(
            callback_url,
            json=payload.model_dump(by_alias=True),
            timeout=self._timeout_seconds,
        )
        response.raise_for_status()
