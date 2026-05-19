#!/bin/bash

set -e

usage() {
    echo "Usage: $0 [PLATFORM...]"
    echo ""
    echo "Supported platforms:"
    echo "  linux/amd64   linux/x86_64    linux"
    echo "  linux/arm64   linux/aarch64"
    echo "  darwin/amd64  darwin/x86_64   mac"
    echo "  darwin/arm64  darwin/aarch64  mac/arm64"
    echo "  windows/amd64 windows/x86_64  win"
    echo ""
    echo "If no platform is specified, all platforms are built."
    exit 1
}

# Resolve a platform string to canonical GOOS/GOARCH
resolve_platform() {
    case "$1" in
        linux/x86_64|linux/amd64|linux)
            echo "linux amd64" ;;
        linux/aarch64|linux/arm64)
            echo "linux arm64" ;;
        darwin/x86_64|darwin/amd64|mac)
            echo "darwin amd64" ;;
        darwin/aarch64|darwin/arm64|mac/arm64)
            echo "darwin arm64" ;;
        windows/x86_64|windows/amd64|win)
            echo "windows amd64" ;;
        *)
            echo "Unknown platform: $1" >&2
            return 1 ;;
    esac
}

# Build directory
BUILD_DIR="build"

# Version info
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS="-s -w -X main.Version=${VERSION}"

# Determine target platforms
if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    usage
fi

if [ $# -eq 0 ]; then
    # Build all platforms
    TARGETS=("linux amd64" "linux arm64" "darwin amd64" "darwin arm64" "windows amd64")
else
    TARGETS=()
    for arg in "$@"; do
        resolved=$(resolve_platform "$arg") || exit 1
        TARGETS+=("$resolved")
    done
fi

echo "Building douyin v${VERSION} at ${BUILD_TIME}"
echo "============================================"

# Clean and recreate build dir
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

for target in "${TARGETS[@]}"; do
    read -r GOOS GOARCH <<< "$target"

    if [ "$GOOS" = "windows" ]; then
        OUTPUT="${BUILD_DIR}/douyin.exe"
    else
        OUTPUT="${BUILD_DIR}/douyin"
    fi

    echo "Building ${GOOS}/${GOARCH}..."
    env CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
        go build -ldflags "$LDFLAGS" -o "$OUTPUT" .

    # Rename to include platform suffix
    if [ "$GOOS" = "windows" ]; then
        mv "${BUILD_DIR}/douyin.exe" "${BUILD_DIR}/douyin_${GOOS}_${GOARCH}.exe"
    else
        mv "${BUILD_DIR}/douyin" "${BUILD_DIR}/douyin_${GOOS}_${GOARCH}"
    fi
done

echo "============================================"
echo "Done. Binaries in ${BUILD_DIR}/"
ls -lh "$BUILD_DIR"
