import math
import sys
from pathlib import Path

import cv2
import numpy as np

SP = Path(__file__).parent


def label_of(t: float, extra: str) -> str:
    m, s = divmod(t, 60)
    return f"{int(m):02d}:{s:04.1f} {extra}"


def build(name: str, items: list[tuple[float, str]], src_dir: str, prefix: str, cols: int) -> None:
    tiles = []
    for t, extra in items:
        path = SP / src_dir / f"{prefix}_{t}.jpg"
        img = cv2.imread(str(path))
        if img is None:
            print(f"missing {path}")
            continue
        h, w = img.shape[:2]
        scale = 480 / w
        img = cv2.resize(img, (480, int(h * scale)))
        cv2.rectangle(img, (0, 0), (480, 26), (0, 0, 0), -1)
        cv2.putText(img, label_of(t, extra), (6, 19), cv2.FONT_HERSHEY_SIMPLEX, 0.55, (0, 255, 255), 1, cv2.LINE_AA)
        tiles.append(img)
    max_h = max(t.shape[0] for t in tiles)
    tiles = [cv2.copyMakeBorder(t, 0, max_h - t.shape[0], 0, 0, cv2.BORDER_CONSTANT) for t in tiles]
    rows = math.ceil(len(tiles) / cols)
    while len(tiles) < rows * cols:
        tiles.append(np.zeros_like(tiles[0]))
    grid = np.vstack([np.hstack(tiles[r * cols:(r + 1) * cols]) for r in range(rows)])
    out = SP / f"{name}.jpg"
    cv2.imwrite(str(out), grid, [cv2.IMWRITE_JPEG_QUALITY, 85])
    print(out, grid.shape)


GATED = [
    (15.015, "0.46 explosion"),
    (22.522, "0.84 magic"),
    (54.346, "0.42 electr+lightn"),
    (73.073, "0.89 magic+eball"),
    (80.205, "0.33 explosion"),
    (95.679, "0.36 fire"),
    (101.435, "0.78 fire"),
    (102.936, "0.65 fire CUT"),
    (252.794, "0.28 fire"),
    (840.882, "0.29 magic"),
    (1097.972, "0.69 magic"),
    (1102.226, "0.94 magic"),
    (1102.393, "0.80 magic CUT"),
    (1144.935, "0.25 magic"),
    (1203.869, "0.31 magic"),
    (1285.075, "0.30 fire"),
]

NEARMISS = [
    (1202.868, "0.24"),
    (251.793, "0.23"),
    (290.999, "0.21"),
    (229.688, "0.20"),
    (43.835, "0.20"),
    (250.792, "0.18"),
    (1171.837, "0.17"),
    (249.791, "0.17"),
    (4.004, "0.14"),
    (248.790, "0.14"),
    (486.027, "0.13"),
    (16.308, "0.13"),
    (3.003, "0.12"),
    (847.722, "0.12 CUT"),
    (1170.836, "0.12"),
    (1098.222, "0.11 CUT"),
    (78.203, "0.11 CUT"),
    (228.687, "0.11"),
    (988.821, "0.11 CUT"),
]

build("gated_grid", GATED, "gated", "g", 4)
build("nearmiss_grid", NEARMISS, "nearmiss", "n", 4)
