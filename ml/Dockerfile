FROM runpod/pytorch:2.8.0-py3.11-cuda12.8.1-cudnn-devel-ubuntu22.04

ENV DEBIAN_FRONTEND=noninteractive \
    PIP_NO_CACHE_DIR=1 \
    PYTHONUNBUFFERED=1 \
    HF_HOME=/data/models/hf \
    TEACHER_TAGGER_MODEL_PATH=/data/models/wd-vit-tagger-v3.onnx \
    TEACHER_TAGGER_TAGS_PATH=/data/models/wd-vit-tagger-v3-tags.csv

RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app/teacher

COPY pyproject.toml ./
COPY src ./src

RUN pip install --no-cache-dir -e .

RUN mkdir -p /data/models \
    && curl -fSL -o /data/models/wd-vit-tagger-v3.onnx \
       https://huggingface.co/SmilingWolf/wd-vit-tagger-v3/resolve/main/model.onnx \
    && curl -fSL -o /data/models/wd-vit-tagger-v3-tags.csv \
       https://huggingface.co/SmilingWolf/wd-vit-tagger-v3/resolve/main/selected_tags.csv

RUN python -c "from huggingface_hub import snapshot_download; snapshot_download('IDEA-Research/grounding-dino-base'); snapshot_download('facebook/sam-vit-base')"

ENTRYPOINT ["python", "-m", "upanime_teacher.cli"]
