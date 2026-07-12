from __future__ import annotations

import mimetypes
from pathlib import Path, PurePosixPath

from botocore.exceptions import ClientError

_KNOWN_CONTENT_TYPES = {
    ".mp4": "video/mp4",
    ".webm": "video/webm",
    ".mkv": "video/x-matroska",
    ".avi": "video/x-msvideo",
    ".mov": "video/quicktime",
    ".jpg": "image/jpeg",
    ".jpeg": "image/jpeg",
    ".png": "image/png",
    ".webp": "image/webp",
    ".gif": "image/gif",
}


def _content_type_for_key(key: str) -> str:
    ext = PurePosixPath(key).suffix.lower()
    if ext in _KNOWN_CONTENT_TYPES:
        return _KNOWN_CONTENT_TYPES[ext]
    guessed, _ = mimetypes.guess_type(key)
    return guessed or "application/octet-stream"


class R2StorageClient:
    def __init__(self, account_id: str, access_key_id: str, access_secret: str, bucket_name: str) -> None:
        import boto3

        endpoint_url = f"https://{account_id}.r2.cloudflarestorage.com"
        self._bucket_name = bucket_name
        self._client = boto3.client(
            "s3",
            endpoint_url=endpoint_url,
            aws_access_key_id=access_key_id,
            aws_secret_access_key=access_secret,
            region_name="auto",
        )

    def upload_file(self, source_path: Path, storage_key: str) -> None:
        self._client.upload_file(
            str(source_path),
            self._bucket_name,
            storage_key,
            ExtraArgs={"ContentType": _content_type_for_key(storage_key)},
        )

    def exists(self, storage_key: str) -> bool:
        try:
            self._client.head_object(Bucket=self._bucket_name, Key=storage_key)
        except ClientError as exc:
            error_code = exc.response.get("Error", {}).get("Code", "")
            if error_code in {"404", "NoSuchKey", "NotFound"}:
                return False
            raise
        return True
