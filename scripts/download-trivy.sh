#!/bin/bash
# Download Trivy binaries for goreleaser bundled release
# Usage: scripts/download-trivy.sh [version]
# Downloads Trivy for linux/darwin amd64/arm64 into dist/trivy/

set -euo pipefail

TRIVY_VERSION="${1:-0.58.2}"
DEST_DIR="dist/trivy"

mkdir -p "$DEST_DIR"

download_trivy() {
    local os=$1
    local arch=$2
    local trivy_os trivy_arch ext

    # Map Go OS/Arch names to Trivy release names
    case "$os" in
        linux)  trivy_os="Linux" ;;
        darwin) trivy_os="macOS" ;;
        *)      echo "Skipping unsupported OS: $os"; return ;;
    esac

    case "$arch" in
        amd64) trivy_arch="64bit" ;;
        arm64) trivy_arch="ARM64" ;;
        *)     echo "Skipping unsupported arch: $arch"; return ;;
    esac

    ext="tar.gz"
    local filename="trivy_${TRIVY_VERSION}_${trivy_os}-${trivy_arch}.${ext}"
    local url="https://github.com/aquasecurity/trivy/releases/download/v${TRIVY_VERSION}/${filename}"
    local out_dir="${DEST_DIR}/${os}_${arch}"

    mkdir -p "$out_dir"

    echo "Downloading Trivy ${TRIVY_VERSION} for ${os}/${arch}..."
    if curl -sSL -o "${out_dir}/${filename}" "$url"; then
        echo "Extracting ${filename}..."
        tar -xzf "${out_dir}/${filename}" -C "$out_dir" trivy
        rm -f "${out_dir}/${filename}"
        chmod +x "${out_dir}/trivy"
        echo "Done: ${out_dir}/trivy"
    else
        echo "Warning: Failed to download Trivy for ${os}/${arch}"
    fi
}

# Download for all supported platforms
download_trivy linux amd64
download_trivy linux arm64
download_trivy darwin amd64
download_trivy darwin arm64

echo "Trivy downloads complete."
