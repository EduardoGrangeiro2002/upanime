FROM runpod/pytorch:2.8.0-py3.11-cuda12.8.1-cudnn-devel-ubuntu22.04

ENV DEBIAN_FRONTEND=noninteractive \
    PIP_NO_CACHE_DIR=1 \
    PYTHONUNBUFFERED=1 \
    WORKER_TEMP_ROOT=/data/tmp \
    WORKER_MODEL_PATH=/data/models/realesr-animevideov3.pth \
    WORKER_HURRDEBLUR_MODEL_PATH=/data/models/1x-HurrDeblur-SuperUltraCompact.pth \
    WORKER_RIFE_DIR=/data/models/Practical-RIFE \
    WORKER_TARGET_HEIGHT=1080 \
    WORKER_ENCODE_PRESET=medium \
    WORKER_REQUEST_TIMEOUT_SECONDS=1800 \
    WORKER_CALLBACK_TIMEOUT_SECONDS=30 \
    WORKER_ENABLE_TORCH_COMPILE=1

RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    unzip \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app/worker

COPY pyproject.toml ./
COPY src ./src

RUN pip install --no-cache-dir -e .

RUN mkdir -p /data/tmp /data/models \
    && curl -fSL -o /data/models/realesr-animevideov3.pth \
       https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.5.0/realesr-animevideov3.pth \
    && curl -fSL -o /data/models/1x-HurrDeblur-SuperUltraCompact.pth \
       https://objectstorage.us-phoenix-1.oraclecloud.com/n/ax6ygfvpvzka/b/open-modeldb-files/o/1x-HurrDeblur-SuperUltraCompact.pth

RUN git clone --depth 1 https://github.com/hzwer/Practical-RIFE.git /data/models/Practical-RIFE \
    && pip install --no-cache-dir gdown \
    && gdown 1ZKjcbmt1hypiFprJPIKW0Tt0lr_2i7bg -O /tmp/rife_v4.25.zip \
    && unzip -o -q /tmp/rife_v4.25.zip -d /data/models/Practical-RIFE \
    && rm -f /tmp/rife_v4.25.zip \
    && rm -rf /data/models/Practical-RIFE/train_log/__MACOSX \
    && touch /data/models/Practical-RIFE/train_log/__init__.py

ENTRYPOINT ["python", "-m", "upanime_worker.handler"]
