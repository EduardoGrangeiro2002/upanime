from __future__ import annotations

import json
import os
import sys
import time
from pathlib import Path

import numpy as np
import torch
from PIL import Image, ImageDraw

PROMPT = os.getenv("PROBE_PROMPT", "fire. flames. explosion. lightning. energy beam. magic aura. glowing energy.")
BOX_THRESHOLD = float(os.getenv("PROBE_THRESHOLD", "0.25"))
TEXT_THRESHOLD = float(os.getenv("PROBE_TEXT_THRESHOLD", "0.2"))
SUFFIX = os.getenv("PROBE_SUFFIX", "teacher")

CLASS_COLORS = {
    "fire": (249, 115, 22),
    "flames": (249, 115, 22),
    "explosion": (244, 63, 94),
    "lightning": (56, 189, 248),
    "energy beam": (250, 204, 21),
    "glowing energy": (250, 204, 21),
    "magic aura": (167, 139, 250),
}
DEFAULT_COLOR = (148, 163, 184)


def color_for(label: str) -> tuple[int, int, int]:
    for key, color in CLASS_COLORS.items():
        if key in label:
            return color
    return DEFAULT_COLOR


def detect(processor, model, image: Image.Image) -> dict:
    inputs = processor(images=image, text=PROMPT, return_tensors="pt")
    with torch.inference_mode():
        outputs = model(**inputs)
    target_sizes = [image.size[::-1]]
    try:
        results = processor.post_process_grounded_object_detection(
            outputs, inputs.input_ids, box_threshold=BOX_THRESHOLD,
            text_threshold=TEXT_THRESHOLD, target_sizes=target_sizes,
        )
    except TypeError:
        results = processor.post_process_grounded_object_detection(
            outputs, inputs.input_ids, threshold=BOX_THRESHOLD,
            text_threshold=TEXT_THRESHOLD, target_sizes=target_sizes,
        )
    result = results[0]
    labels = result.get("text_labels", result.get("labels"))
    return {"boxes": result["boxes"], "scores": result["scores"], "labels": labels}


def segment(sam_processor, sam_model, image: Image.Image, boxes: list[list[float]]) -> list[np.ndarray]:
    if not boxes:
        return []
    inputs = sam_processor(image, input_boxes=[boxes], return_tensors="pt")
    with torch.inference_mode():
        outputs = sam_model(**inputs)
    masks = sam_processor.image_processor.post_process_masks(
        outputs.pred_masks.cpu(), inputs["original_sizes"].cpu(), inputs["reshaped_input_sizes"].cpu(),
    )[0]
    ious = outputs.iou_scores.cpu()[0]
    best = []
    for i in range(masks.shape[0]):
        channel = int(ious[i].argmax())
        best.append(masks[i, channel].numpy().astype(bool))
    return best


def render(image: Image.Image, detections: list[dict], out_path: Path) -> None:
    canvas = image.convert("RGBA")
    overlay = Image.new("RGBA", canvas.size, (0, 0, 0, 0))
    for det in detections:
        color = color_for(det["label"])
        if det["mask"] is not None:
            mask_img = Image.fromarray((det["mask"] * 140).astype(np.uint8), mode="L")
            solid = Image.new("RGBA", canvas.size, color + (0,))
            solid.putalpha(mask_img)
            overlay = Image.alpha_composite(overlay, solid)
    canvas = Image.alpha_composite(canvas, overlay)
    draw = ImageDraw.Draw(canvas)
    for det in detections:
        color = color_for(det["label"])
        box = det["box"]
        draw.rectangle(box, outline=color + (255,), width=3)
        draw.text((box[0] + 4, max(0, box[1] - 14)), f"{det['label']} {det['score']:.2f}", fill=color + (255,))
    canvas.convert("RGB").save(out_path, quality=88)


def main() -> None:
    from transformers import (
        AutoModelForZeroShotObjectDetection,
        AutoProcessor,
        SamModel,
        SamProcessor,
    )

    frames_dir = Path(__file__).parent / "probe-frames"
    out_dir = Path(__file__).parent / "probe-out"
    out_dir.mkdir(exist_ok=True)

    print("loading grounding-dino-base...", flush=True)
    dino_processor = AutoProcessor.from_pretrained("IDEA-Research/grounding-dino-base")
    dino_model = AutoModelForZeroShotObjectDetection.from_pretrained("IDEA-Research/grounding-dino-base").eval()
    print("loading sam-vit-base...", flush=True)
    sam_processor = SamProcessor.from_pretrained("facebook/sam-vit-base")
    sam_model = SamModel.from_pretrained("facebook/sam-vit-base").eval()

    report = []
    for frame_path in sorted(frames_dir.glob("*.jpg")):
        image = Image.open(frame_path).convert("RGB")
        started = time.time()
        result = detect(dino_processor, dino_model, image)
        dino_seconds = time.time() - started

        boxes = [box.tolist() for box in result["boxes"]]
        started = time.time()
        masks = segment(sam_processor, sam_model, image, boxes)
        sam_seconds = time.time() - started

        detections = []
        for i, box in enumerate(boxes):
            detections.append({
                "label": str(result["labels"][i]),
                "score": float(result["scores"][i]),
                "box": box,
                "mask": masks[i] if i < len(masks) else None,
            })

        render(image, detections, out_dir / f"{frame_path.stem}_{SUFFIX}.jpg")
        report.append({
            "frame": frame_path.name,
            "dino_s": round(dino_seconds, 2),
            "sam_s": round(sam_seconds, 2),
            "detections": [
                {"label": d["label"], "score": round(d["score"], 3),
                 "mask_area_pct": round(float(d["mask"].mean()) * 100, 2) if d["mask"] is not None else 0}
                for d in detections
            ],
        })
        print(f"{frame_path.name}: {len(detections)} detections "
              f"({dino_seconds:.1f}s dino + {sam_seconds:.1f}s sam)", flush=True)

    (out_dir / f"report_{SUFFIX}.json").write_text(json.dumps(report, indent=2))
    print("done", flush=True)


if __name__ == "__main__":
    sys.exit(main())
