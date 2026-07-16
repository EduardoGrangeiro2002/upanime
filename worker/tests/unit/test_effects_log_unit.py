from __future__ import annotations

import json
from pathlib import Path
from types import SimpleNamespace

import numpy as np
import torch

from upanime_worker.effects import EffectsComp
from upanime_worker.models import WorkerJobRequest
from upanime_worker.quality_pipeline import QualityUpscalePipeline
from upanime_worker.tagger import EffectTagger


class StubComp:
    def __init__(self) -> None:
        self.resets = 0

    def reset(self) -> None:
        self.resets += 1


class StubTagger:
    def __init__(self, scores: list[tuple[list[str], float]]) -> None:
        self._scores = scores
        self.calls = 0

    def available(self) -> bool:
        return True

    def shot_effect_scores(self, frame: object) -> tuple[list[str], float]:
        result = self._scores[min(self.calls, len(self._scores) - 1)]
        self.calls += 1
        return result


def make_pipeline(tagger: object | None = None) -> QualityUpscalePipeline:
    return QualityUpscalePipeline(
        model_path=Path("/nonexistent/model.pth"),
        hurrdeblur_model_path=Path("/nonexistent/deblur.pth"),
        target_height=1080,
        encode_preset="medium",
        enable_torch_compile=False,
        tagger=tagger,
    )


def cpu_runtime() -> SimpleNamespace:
    return SimpleNamespace(numpy=np, torch=torch, device=torch.device("cpu"))


def test_shot_gate_returns_tags_and_prob_on_hit():
    tagger = StubTagger([(["fire"], 0.61)])
    pipeline = make_pipeline(tagger)
    comp = StubComp()

    gated, tags, max_prob = pipeline._shot_gate([object()], 0, comp, None, None)

    assert gated is True
    assert tags == ["fire"]
    assert max_prob == 0.61
    assert comp.resets == 1


def test_shot_gate_reports_max_prob_across_samples_on_miss():
    tagger = StubTagger([([], 0.05), ([], 0.22), ([], 0.11)])
    pipeline = make_pipeline(tagger)

    gated, tags, max_prob = pipeline._shot_gate([object(), object(), object()], 3, StubComp(), None, None)

    assert gated is False
    assert tags == []
    assert max_prob == 0.22
    assert tagger.calls == 3


def test_shot_gate_stops_sampling_after_first_hit():
    tagger = StubTagger([([], 0.1), (["energy"], 0.4), ([], 0.9)])
    pipeline = make_pipeline(tagger)

    gated, tags, max_prob = pipeline._shot_gate([object(), object(), object()], 7, StubComp(), None, None)

    assert gated is True
    assert tags == ["energy"]
    assert max_prob == 0.4
    assert tagger.calls == 2


def test_shot_gate_passes_all_when_tagger_missing():
    pipeline = make_pipeline(tagger=None)

    gated, tags, max_prob = pipeline._shot_gate([object()], 0, StubComp(), None, None)

    assert gated is True
    assert tags == ["gate-off"]
    assert max_prob == 1.0


def test_tagger_unavailable_scores_gate_off(tmp_path):
    tagger = EffectTagger(tmp_path / "missing.onnx", tmp_path / "missing.csv")

    tags, max_prob = tagger.shot_effect_scores(object())

    assert tags == ["gate-off"]
    assert max_prob == 1.0


def test_log_gate_appends_rounded_entry():
    pipeline = make_pipeline()
    log: list[dict] = []

    pipeline._log_gate(log, 12.34567, 5, "cut", True, ["fire"], 0.61234)
    pipeline._log_gate(None, 1.0, 0, "cut", False, [], 0.1)

    assert log == [{
        "t": 12.346,
        "shot": 5,
        "event": "cut",
        "gated": True,
        "tags": ["fire"],
        "max_prob": 0.6123,
    }]


def test_write_effects_log_creates_json(tmp_path):
    pipeline = make_pipeline()
    entries = [{"t": 0.0, "shot": 0, "event": "cut", "gated": False, "tags": [], "max_prob": 0.02}]

    pipeline._write_effects_log(tmp_path / "dataset", 23.976, "stream", entries)

    payload = json.loads((tmp_path / "dataset" / "effects_log.json").read_text())
    assert payload["mode"] == "stream"
    assert payload["fps"] == 23.976
    assert payload["entries"] == entries


def test_write_effects_log_skips_when_entries_none(tmp_path):
    pipeline = make_pipeline()

    pipeline._write_effects_log(tmp_path / "dataset", 24.0, "stream", None)

    assert not (tmp_path / "dataset").exists()


def test_comp_only_batch_passthrough_without_comp():
    pipeline = make_pipeline()
    frames = [np.full((48, 64, 3), 120, dtype=np.uint8)]

    out = pipeline._comp_only_batch(frames, cpu_runtime(), None)

    assert len(out) == 1
    assert out[0].shape == (48, 64, 3)
    assert np.array_equal(out[0], frames[0])


def test_comp_only_batch_keeps_source_resolution_and_applies_comp():
    pipeline = make_pipeline()
    comp = EffectsComp(torch, torch.device("cpu"))
    frame = np.full((96, 128, 3), 38, dtype=np.uint8)
    frame[30:66, 40:88, 2] = 242
    frame[30:66, 40:88, 1] = 191
    frame[42:54, 56:72] = 247

    out = pipeline._comp_only_batch([frame], cpu_runtime(), comp)

    assert len(out) == 1
    assert out[0].shape == (96, 128, 3)
    assert out[0].dtype == np.uint8
    assert not np.array_equal(out[0], frame)


def test_job_request_parses_skip_upscale():
    base = {
        "jobId": 1,
        "sourceUrl": "https://example.com/v.mp4",
        "sourceStorageKey": "a/src.mp4",
        "resultStorageKey": "a/out.mp4",
    }

    assert WorkerJobRequest(**base).skip_upscale is False
    assert WorkerJobRequest(**base, skipUpscale=True).skip_upscale is True
