from __future__ import annotations

from pathlib import Path

from upanime_worker.quality_pipeline import (
    EncodeParams,
    QualityUpscalePipeline,
    VideoMetadata,
)


def make_pipeline() -> QualityUpscalePipeline:
    return QualityUpscalePipeline(
        model_path=Path("/nonexistent/model.pth"),
        hurrdeblur_model_path=Path("/nonexistent/deblur.pth"),
        target_height=1080,
        encode_preset="medium",
        enable_torch_compile=False,
    )


def make_metadata() -> VideoMetadata:
    return VideoMetadata(
        width=1920,
        height=1080,
        fps=24.0,
        total_frames=100,
        duration=4.17,
        target_height=2160,
        target_width=3840,
        outscale=2.0,
    )


def encode_vf(cmd: list[str]) -> str:
    return cmd[cmd.index("-vf") + 1]


def test_decode_command_only_deinterlaces_interlaced_frames():
    cmd = make_pipeline()._build_decode_command(Path("/in.mp4"))
    assert "yadif=deint=interlaced" in encode_vf(cmd)


def test_encode_command_outputs_tagged_yuv420p():
    cmd = make_pipeline()._build_encode_command(
        Path("/in.mp4"),
        Path("/out.mp4"),
        make_metadata(),
        EncodeParams(sharpen=0.5, saturation=1.20, contrast=1.05),
    )
    vf = encode_vf(cmd)
    assert "vibrance=intensity=0.300" in vf
    assert "scale=out_color_matrix=bt709,format=yuv420p" in vf
    assert "deband" in vf
    assert "setparams=color_primaries=bt709:color_trc=bt709:colorspace=bt709" in vf
    assert "saturation" not in vf


def test_encode_command_clamps_vibrance():
    cmd = make_pipeline()._build_encode_command(
        Path("/in.mp4"),
        Path("/out.mp4"),
        make_metadata(),
        EncodeParams(sharpen=0.5, saturation=3.0, contrast=1.05),
    )
    assert "vibrance=intensity=2.000" in encode_vf(cmd)


def test_encode_command_defaults_do_not_crash():
    cmd = make_pipeline()._build_encode_command(
        Path("/in.mp4"), Path("/out.mp4"), make_metadata(), None
    )
    assert "vibrance=intensity=0.300" in encode_vf(cmd)
