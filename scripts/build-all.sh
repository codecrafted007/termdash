#!/bin/bash
#
# Build termdash for multiple platforms using Docker
#
# Usage: ./scripts/build-all.sh [version]
#
# Example: ./scripts/build-all.sh v1.0.0
#

set -e

VERSION=${1:-"dev"}
OUTPUT_DIR="dist"
BINARY_NAME="termdash"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Platforms to build for
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "linux/arm/v7"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# Create output directory
mkdir -p "$OUTPUT_DIR"

log_info "Building termdash version: $VERSION"
log_info "Output directory: $OUTPUT_DIR"
echo ""

# Build for each platform
for PLATFORM in "${PLATFORMS[@]}"; do
    OS=$(echo $PLATFORM | cut -d'/' -f1)
    ARCH=$(echo $PLATFORM | cut -d'/' -f2)
    VARIANT=$(echo $PLATFORM | cut -d'/' -f3)

    OUTPUT_NAME="${BINARY_NAME}-${OS}-${ARCH}"
    if [ -n "$VARIANT" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}-${VARIANT}"
    fi
    if [ "$OS" == "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi

    log_info "Building for $PLATFORM..."

    # Build using Go directly (faster than Docker for local builds)
    GOARM=""
    if [ "$VARIANT" == "v7" ]; then
        GOARM="7"
    fi

    CGO_FLAG=0
    if [ "$OS" == "darwin" ]; then
        CGO_FLAG=1
    fi

    CGO_ENABLED=$CGO_FLAG GOOS=$OS GOARCH=$ARCH GOARM=$GOARM go build \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o "${OUTPUT_DIR}/${OUTPUT_NAME}" \
        ./cmd/termdash

    if [ $? -eq 0 ]; then
        log_info "  ✓ Created ${OUTPUT_DIR}/${OUTPUT_NAME}"
    else
        log_error "  ✗ Failed to build for $PLATFORM"
    fi
done

echo ""
log_info "Build complete! Binaries are in the '$OUTPUT_DIR' directory:"
echo ""
ls -lh "$OUTPUT_DIR"

# Create checksums
log_info "Generating checksums..."
cd "$OUTPUT_DIR"
sha256sum * > checksums.txt 2>/dev/null || shasum -a 256 * > checksums.txt
cd ..
log_info "  ✓ Created ${OUTPUT_DIR}/checksums.txt"

echo ""
log_info "Done! 🎉"
