#!/bin/bash

# vman project build script
# This script provides cross-platform build capabilities

set -e

# Project information
PROJECT_NAME="vman"
PROJECT_MODULE="github.com/songzhibin97/vman"
BUILD_DIR="build"
CMD_DIR="cmd/vman"

# Version and build info
VERSION=${VERSION:-"dev"}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS="-X ${PROJECT_MODULE}/internal/version.Version=${VERSION} \
         -X ${PROJECT_MODULE}/internal/version.Commit=${COMMIT} \
         -X ${PROJECT_MODULE}/internal/version.BuildTime=${BUILD_TIME}"

# Supported platforms
PLATFORMS=${PLATFORMS:-"linux/amd64 darwin/amd64 darwin/arm64 windows/amd64"}

# Functions
build_single() {
    local platform=$1
    local os=$(echo $platform | cut -d'/' -f1)
    local arch=$(echo $platform | cut -d'/' -f2)
    local output_name=${PROJECT_NAME}
    
    if [ "$os" = "windows" ]; then
        output_name="${PROJECT_NAME}.exe"
    fi
    
    local output_path="${BUILD_DIR}/${os}-${arch}/${output_name}"
    
    echo "Building for ${os}/${arch}..."
    
    mkdir -p "$(dirname $output_path)"
    
    GOOS=$os GOARCH=$arch go build \
        -ldflags "$LDFLAGS" \
        -o "$output_path" \
        ./${CMD_DIR}
    
    echo "Built: $output_path"
}

clean() {
    echo "Cleaning build directory..."
    rm -rf ${BUILD_DIR}
}

build_all() {
    echo "Building ${PROJECT_NAME} for multiple platforms..."
    echo "Version: $VERSION"
    echo "Commit: $COMMIT"
    echo "Build Time: $BUILD_TIME"
    echo ""
    
    clean
    
    for platform in $PLATFORMS; do
        build_single $platform
    done
    
    echo ""
    echo "Build completed! Artifacts in ${BUILD_DIR}/"
}

build_local() {
    echo "Building ${PROJECT_NAME} for local platform..."
    go build -ldflags "$LDFLAGS" -o ${BUILD_DIR}/${PROJECT_NAME} ./${CMD_DIR}
    echo "Built: ${BUILD_DIR}/${PROJECT_NAME}"
}

# Main execution
case "${1:-all}" in
    "clean")
        clean
        ;;
    "local")
        build_local
        ;;
    "all")
        build_all
        ;;
    *)
        echo "Usage: $0 {all|local|clean}"
        echo "  all   - Build for all supported platforms"
        echo "  local - Build for current platform only"
        echo "  clean - Clean build directory"
        exit 1
        ;;
esac