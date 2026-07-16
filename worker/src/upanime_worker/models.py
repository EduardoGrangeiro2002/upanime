from __future__ import annotations

from pydantic import BaseModel, ConfigDict, Field, HttpUrl, field_validator

VALID_TARGET_HEIGHTS = {1080, 1440, 2160}


class VariantSpec(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    height: int
    storage_key: str = Field(alias="storageKey")


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
    interpolate: bool = Field(default=False)
    pan_ratio: float | None = Field(default=None, alias="panRatio")
    effects: bool = Field(default=False)
    effects_strength: float | None = Field(default=None, alias="effectsStrength")
    effects_sensitivity: float | None = Field(default=None, alias="effectsSensitivity")
    skip_upscale: bool = Field(default=False, alias="skipUpscale")
    variants: list[VariantSpec] = Field(default_factory=list)
    callback_url: HttpUrl | None = Field(default=None, alias="callbackUrl")

    @field_validator("target_height")
    @classmethod
    def validate_target_height(cls, value: int) -> int:
        if value not in VALID_TARGET_HEIGHTS:
            raise ValueError(f"target_height must be one of {sorted(VALID_TARGET_HEIGHTS)}")
        return value

    @field_validator("pan_ratio")
    @classmethod
    def clamp_pan_ratio(cls, value: float | None) -> float | None:
        if value is None:
            return None
        return min(0.9, max(0.6, value))

    @field_validator("effects_strength")
    @classmethod
    def clamp_effects_strength(cls, value: float | None) -> float | None:
        if value is None:
            return None
        return min(1.5, max(0.0, value))

    @field_validator("effects_sensitivity")
    @classmethod
    def clamp_effects_sensitivity(cls, value: float | None) -> float | None:
        if value is None:
            return None
        return min(1.5, max(0.5, value))


class WorkerCallbackPayload(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    job_id: int = Field(alias="jobId")
    status: str
    error: str = ""
    result_storage_key: str = Field(default="", alias="resultStorageKey")
    file_name: str = Field(default="", alias="fileName")
