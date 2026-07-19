#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
IMAGE="alkindar/upanime-teacher"
VERSION_FILE="${SCRIPT_DIR}/version.json"

current_version() {
  python3 -c "import json; print(json.load(open('${VERSION_FILE}'))['version'])"
}

bump_version() {
  local version="$1"
  local part="$2"
  local major minor patch
  IFS='.' read -r major minor patch <<< "$version"

  case "$part" in
    major) echo "$((major + 1)).0.0" ;;
    minor) echo "${major}.$((minor + 1)).0" ;;
    patch) echo "${major}.${minor}.$((patch + 1))" ;;
  esac
}

save_version() {
  printf '{"version": "%s"}\n' "$1" > "$VERSION_FILE"
}

BUMP="${1:-}"

if [ -z "$BUMP" ]; then
  echo "Current version: $(current_version)"
  echo ""
  echo "Usage: ./deploy.sh <--major|--minor|--patch>"
  echo "  --major  1.2.3 → 2.0.0"
  echo "  --minor  1.2.3 → 1.3.0"
  echo "  --patch  1.2.3 → 1.2.4"
  exit 1
fi

CURRENT="$(current_version)"

case "$BUMP" in
  --major) VERSION="$(bump_version "$CURRENT" major)" ;;
  --minor) VERSION="$(bump_version "$CURRENT" minor)" ;;
  --patch) VERSION="$(bump_version "$CURRENT" patch)" ;;
  *)
    echo "Unknown flag: $BUMP"
    echo "Use --major, --minor, or --patch"
    exit 1
    ;;
esac

echo "${CURRENT} → ${VERSION}"
echo ""

echo "Building ${IMAGE}:${VERSION} for linux/amd64..."
docker buildx build --platform linux/amd64 -t "${IMAGE}:${VERSION}" -t "${IMAGE}:latest" "${SCRIPT_DIR}"

echo "Pushing ${IMAGE}:${VERSION}..."
docker push "${IMAGE}:${VERSION}"

echo "Pushing ${IMAGE}:latest..."
docker push "${IMAGE}:latest"

save_version "$VERSION"

echo ""
echo "Deployed ${IMAGE}:${VERSION}"
