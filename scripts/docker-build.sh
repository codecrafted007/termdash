#!/bin/bash
#
# Build termdash using Docker (no local Go installation required)
#
# Usage: ./scripts/docker-build.sh [os] [arch] [version]
#
# Examples:
#   ./scripts/docker-build.sh                    # Build for linux/amd64
#   ./scripts/docker-build.sh linux arm64        # Build for linux/arm64
#   ./scripts/docker-build.sh darwin amd64 v1.0  # Build for macOS amd64
#

set -e

OS=${1:-"linux"}
ARCH=${2:-"amd64"}
VERSION=${3:-"dev"}
OUTPUT_DIR="dist"

# Color output
GREEN='\033[0;32m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

mkdir -p "$OUTPUT_DIR"

OUTPUT_NAME="termdash-${OS}-${ARCH}"
if [ "$OS" == "windows" ]; then
    OUTPUT_NAME="${OUTPUT_NAME}.exe"
fi

log_info "Building termdash for ${OS}/${ARCH} (version: ${VERSION})"
log_info "Using Docker for isolated build environment..."

# Build using Docker
docker build \
    --build-arg TARGETOS=$OS \
    --build-arg TARGETARCH=$ARCH \
    --build-arg VERSION=$VERSION \
    --target builder \
    -t termdash-builder:latest \
    .

# Extract binary from container
CONTAINER_ID=$(docker create termdash-builder:latest)
docker cp "${CONTAINER_ID}:/app/bin/termdash" "${OUTPUT_DIR}/${OUTPUT_NAME}"
docker rm "${CONTAINER_ID}" > /dev/null

log_info "✓ Created ${OUTPUT_DIR}/${OUTPUT_NAME}"
ls -lh "${OUTPUT_DIR}/${OUTPUT_NAME}"
