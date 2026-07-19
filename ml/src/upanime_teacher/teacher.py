from __future__ import annotations

from dataclasses import dataclass

import numpy as np

from .photometric import bright_points

PROMPT = "fire. flames. burning fire. explosion. lightning. electricity. energy beam. energy blast. glowing orb. glowing energy. magic aura. magic spell glow."
TEXT_THRESHOLD = 0.2

CLASS_MAP = (
    ("fire", "fire"),
    ("flame", "fire"),
    ("burning", "fire"),
    ("explosion", "fire"),
    ("lightning", "lightning"),
    ("electricity", "lightning"),
    ("beam", "beam"),
    ("blast", "beam"),
    ("orb", "energy"),
    ("energy", "energy"),
    ("aura", "aura"),
    ("magic", "aura"),
    ("glow", "aura"),
)


def class_for(label: str) -> str | None:
    clean = label.strip().lower()
    if not clean:
        return None
    for key, class_name in CLASS_MAP:
        if key in clean:
            return class_name
    return None


@dataclass
class Proposal:
    class_name: str
    mask: np.ndarray
    score: float
    origin: str


def mask_iou(a: np.ndarray, b: np.ndarray) -> float:
    union = float(np.logical_or(a, b).sum())
    if union == 0:
        return 0.0
    return float(np.logical_and(a, b).sum()) / union


class ComposedTeacher:
    def __init__(self, device: str, dino_threshold: float) -> None:
        self._device = device
        self._dino_threshold = dino_threshold
        self._models = None

    def _ensure_models(self) -> None:
        if self._models is not None:
            return
        import torch
        from transformers import (
            AutoModelForZeroShotObjectDetection,
            AutoProcessor,
            SamModel,
            SamProcessor,
        )

        device = torch.device(self._device)
        self._models = {
            "torch": torch,
            "device": device,
            "dino_processor": AutoProcessor.from_pretrained("IDEA-Research/grounding-dino-base"),
            "dino": AutoModelForZeroShotObjectDetection.from_pretrained("IDEA-Research/grounding-dino-base").eval().to(device),
            "sam_processor": SamProcessor.from_pretrained("facebook/sam-vit-base"),
            "sam": SamModel.from_pretrained("facebook/sam-vit-base").eval().to(device),
        }

    def propose(self, frame_bgr: np.ndarray) -> list[Proposal]:
        self._ensure_models()
        from PIL import Image

        image = Image.fromarray(frame_bgr[:, :, ::-1])
        detections = self._detect(image)
        proposals = self._sam_box_proposals(image, detections)
        proposals.extend(self._hsv_point_proposals(image, frame_bgr, proposals))
        return self._merge_by_class(proposals)

    def _detect(self, image: object) -> list[dict]:
        torch = self._models["torch"]
        processor = self._models["dino_processor"]
        inputs = processor(images=image, text=PROMPT, return_tensors="pt").to(self._models["device"])
        with torch.inference_mode():
            outputs = self._models["dino"](**inputs)
        target_sizes = [image.size[::-1]]
        try:
            results = processor.post_process_grounded_object_detection(
                outputs, inputs.input_ids, box_threshold=self._dino_threshold,
                text_threshold=TEXT_THRESHOLD, target_sizes=target_sizes,
            )
        except TypeError:
            results = processor.post_process_grounded_object_detection(
                outputs, inputs.input_ids, threshold=self._dino_threshold,
                text_threshold=TEXT_THRESHOLD, target_sizes=target_sizes,
            )
        result = results[0]
        labels = result.get("text_labels", result.get("labels"))

        detections = []
        for i in range(len(result["boxes"])):
            class_name = class_for(str(labels[i]))
            if class_name is None:
                continue
            detections.append({
                "class": class_name,
                "score": float(result["scores"][i]),
                "box": result["boxes"][i].tolist(),
            })
        return detections

    def _sam_box_proposals(self, image: object, detections: list[dict]) -> list[Proposal]:
        if not detections:
            return []
        masks = self._run_sam(image, boxes=[d["box"] for d in detections])
        return [
            Proposal(class_name=d["class"], mask=masks[i], score=d["score"], origin="teacher")
            for i, d in enumerate(detections)
        ]

    def _hsv_point_proposals(
        self, image: object, frame_bgr: np.ndarray, existing: list[Proposal]
    ) -> list[Proposal]:
        points = bright_points(frame_bgr)
        if not points:
            return []
        masks = self._run_sam(image, points=points)

        proposals = []
        for mask in masks:
            if any(mask_iou(mask, p.mask) > 0.5 for p in existing):
                continue
            if mask.mean() > 0.5:
                continue
            proposals.append(Proposal(class_name="energy", mask=mask, score=0.0, origin="hsv"))
        return proposals

    def _run_sam(
        self,
        image: object,
        boxes: list[list[float]] | None = None,
        points: list[tuple[int, int]] | None = None,
    ) -> list[np.ndarray]:
        torch = self._models["torch"]
        processor = self._models["sam_processor"]
        kwargs = {}
        if boxes is not None:
            kwargs["input_boxes"] = [boxes]
        if points is not None:
            kwargs["input_points"] = [[[list(p)] for p in points]]
        inputs = processor(image, return_tensors="pt", **kwargs).to(self._models["device"])
        with torch.inference_mode():
            outputs = self._models["sam"](**inputs)
        masks = processor.image_processor.post_process_masks(
            outputs.pred_masks.cpu(), inputs["original_sizes"].cpu(), inputs["reshaped_input_sizes"].cpu(),
        )[0]
        ious = outputs.iou_scores.cpu()[0]

        best = []
        for i in range(masks.shape[0]):
            channel = int(ious[i].argmax())
            best.append(masks[i, channel].numpy().astype(bool))
        return best

    def _merge_by_class(self, proposals: list[Proposal]) -> list[Proposal]:
        merged: dict[str, Proposal] = {}
        for proposal in proposals:
            current = merged.get(proposal.class_name)
            if current is None:
                merged[proposal.class_name] = proposal
                continue
            merged[proposal.class_name] = Proposal(
                class_name=proposal.class_name,
                mask=np.logical_or(current.mask, proposal.mask),
                score=max(current.score, proposal.score),
                origin=current.origin if current.origin == "teacher" else proposal.origin,
            )
        return list(merged.values())
