FROM node:22-slim AS client-build

RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /build/client
COPY client/package.json client/pnpm-lock.yaml client/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

COPY client/ ./
RUN pnpm build

FROM golang:1.25-bookworm AS api-build

WORKDIR /build/api
COPY api/go.mod api/go.sum ./
RUN go mod download

COPY api/ ./
RUN CGO_ENABLED=0 go build -o /upanime-api .

FROM python:3.12-slim-bookworm

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

WORKDIR /app

COPY --from=api-build /upanime-api /app/api/upanime-api
COPY --from=client-build /build/client/dist /app/client/dist

COPY scraper/ /app/scraper/
RUN cd /app/scraper && uv sync --frozen --no-dev
RUN cd /app/scraper && uv run playwright install --with-deps chromium

ENV DATABASE_PATH=/app/data/upanime.db
ENV DOWNLOAD_PATH=/app/data/downloads
ENV SCRAPER_DIR=/app/scraper
ENV PORT=7891

VOLUME /app/data

EXPOSE 7891

WORKDIR /app/api
CMD ["./upanime-api"]
