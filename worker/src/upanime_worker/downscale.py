from __future__ import annotations

import subprocess
from pathlib import Path

DOWNSCALE_CRF = "18"


def build_downscale_command(
    input_path: Path,
    output_path: Path,
    height: int,
    encode_preset: str,
) -> list[str]:
    return [
        "ffmpeg",
        "-hide_banner",
        "-loglevel",
        "error",
        "-y",
        "-i",
        str(input_path),
        "-vf",
        f"scale=-2:{height}:flags=lanczos",
        "-c:v",
        "libx264",
        "-preset",
        encode_preset,
        "-crf",
        DOWNSCALE_CRF,
        "-c:a",
        "copy",
        "-movflags",
        "+faststart",
        str(output_path),
    ]


def downscale_video(
    input_path: Path,
    output_path: Path,
    height: int,
    encode_preset: str,
) -> None:
    subprocess.run(
        build_downscale_command(input_path, output_path, height, encode_preset),
        check=True,
    )
