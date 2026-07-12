from __future__ import annotations

import mimetypes
from pathlib import Path

from botocore.exceptions import ClientError


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
        content_type, _ = mimetypes.guess_type(storage_key)
        if not content_type:
            content_type = "application/octet-stream"
        self._client.upload_file(
            str(source_path),
            self._bucket_name,
            storage_key,
            ExtraArgs={"ContentType": content_type},
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
