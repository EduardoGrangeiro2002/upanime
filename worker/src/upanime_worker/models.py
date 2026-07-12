from __future__ import annotations

from pydantic import BaseModel, ConfigDict, Field, HttpUrl, field_validator

VALID_TARGET_HEIGHTS = {1080, 1440, 2160}


class WorkerJobRequest(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    job_id: int = Field(alias="jobId")
    source_url: HttpUrl = Field(alias="sourceUrl")
    source_storage_key: str = Field(alias="sourceStorageKey")
    result_storage_key: str = Field(alias="resultStorageKey")
    target_height: int = Field(default=1080, alias="targetHeight")
    batch_size: int | None = Field(default=None, alias="batchSize")
    sharpen: float | None = Field(default=None)
    saturation: float | None = Field(default=None)
    contrast: float | None = Field(default=None)
    callback_url: HttpUrl | None = Field(default=None, alias="callbackUrl")

    @field_validator("target_height")
    @classmethod
    def validate_target_height(cls, value: int) -> int:
        if value not in VALID_TARGET_HEIGHTS:
            raise ValueError(f"target_height must be one of {sorted(VALID_TARGET_HEIGHTS)}")
        return value


class WorkerCallbackPayload(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    job_id: int = Field(alias="jobId")
    status: str
    error: str = ""
    result_storage_key: str = Field(default="", alias="resultStorageKey")
    file_name: str = Field(default="", alias="fileName")
