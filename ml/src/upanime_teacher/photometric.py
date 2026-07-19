from __future__ import annotations

import cv2
import numpy as np

VALUE_MIN = 199
SATURATION_MIN = 153
MIN_AREA_FRACTION = 0.002
MAX_POINTS = 3


def seed_mask(frame_bgr: np.ndarray) -> np.ndarray:
    hsv = cv2.cvtColor(frame_bgr, cv2.COLOR_BGR2HSV)
    saturation = hsv[:, :, 1]
    value = hsv[:, :, 2]
    mask = ((value >= VALUE_MIN) & (saturation >= SATURATION_MIN)).astype(np.uint8)
    kernel = np.ones((7, 7), np.uint8)
    return cv2.morphologyEx(mask, cv2.MORPH_OPEN, kernel)


def bright_points(frame_bgr: np.ndarray) -> list[tuple[int, int]]:
    mask = seed_mask(frame_bgr)
    total_pixels = mask.shape[0] * mask.shape[1]
    count, _, stats, centroids = cv2.connectedComponentsWithStats(mask)

    blobs = []
    for label in range(1, count):
        area = stats[label, cv2.CC_STAT_AREA]
        if area < total_pixels * MIN_AREA_FRACTION:
            continue
        blobs.append((area, (int(centroids[label][0]), int(centroids[label][1]))))

    blobs.sort(key=lambda item: -item[0])
    return [point for _, point in blobs[:MAX_POINTS]]
