#!/bin/bash
# Build k13d-llm bundle with llama.cpp server and model
# This script creates a complete bundle for running k13d with embedded LLM

set -e

VERSION="${1:-dev}"
LLAMA_CPP_VERSION="${LLAMA_CPP_VERSION:-b4547}"
MODEL_NAME="qwen2.5-0.5b-instruct-q4_k_m.gguf"
MODEL_URL="https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/${MODEL_NAME}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Building k13d-llm bundle v${VERSION}${NC}"
echo "llama.cpp version: ${LLAMA_CPP_VERSION}"
echo ""

# Create build directory
BUILD_DIR="dist/llm-bundle"
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# Determine OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "${ARCH}" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
esac

echo "Building for: ${OS}/${ARCH}"

# Download llama.cpp release
LLAMA_RELEASE_URL="https://github.com/ggerganov/llama.cpp/releases/download/${LLAMA_CPP_VERSION}"

case "${OS}" in
    darwin)
        if [ "${ARCH}" = "arm64" ]; then
            LLAMA_ARCHIVE="llama-${LLAMA_CPP_VERSION}-bin-macos-arm64.zip"
        else
            LLAMA_ARCHIVE="llama-${LLAMA_CPP_VERSION}-bin-macos-x64.zip"
        fi
        ;;
    linux)
        if [ "${ARCH}" = "arm64" ]; then
            LLAMA_ARCHIVE="llama-${LLAMA_CPP_VERSION}-bin-ubuntu-arm64.zip"
        else
            LLAMA_ARCHIVE="llama-${LLAMA_CPP_VERSION}-bin-ubuntu-x64.zip"
        fi
        ;;
    *)
        echo -e "${RED}Unsupported OS: ${OS}${NC}"
        exit 1
        ;;
esac

echo ""
echo -e "${YELLOW}Downloading llama.cpp...${NC}"
LLAMA_URL="${LLAMA_RELEASE_URL}/${LLAMA_ARCHIVE}"
echo "URL: ${LLAMA_URL}"

TEMP_DIR=$(mktemp -d)
curl -L -o "${TEMP_DIR}/${LLAMA_ARCHIVE}" "${LLAMA_URL}"
unzip -q "${TEMP_DIR}/${LLAMA_ARCHIVE}" -d "${TEMP_DIR}/llama"

# Create bin directory and copy llama-server
mkdir -p "${BUILD_DIR}/llm/bin"
if [ -f "${TEMP_DIR}/llama/build/bin/llama-server" ]; then
    cp "${TEMP_DIR}/llama/build/bin/llama-server" "${BUILD_DIR}/llm/bin/"
elif [ -f "${TEMP_DIR}/llama/llama-server" ]; then
    cp "${TEMP_DIR}/llama/llama-server" "${BUILD_DIR}/llm/bin/"
else
    # Search for llama-server
    LLAMA_SERVER=$(find "${TEMP_DIR}/llama" -name "llama-server" -type f | head -1)
    if [ -n "${LLAMA_SERVER}" ]; then
        cp "${LLAMA_SERVER}" "${BUILD_DIR}/llm/bin/"
    else
        echo -e "${RED}Could not find llama-server binary${NC}"
        exit 1
    fi
fi

chmod +x "${BUILD_DIR}/llm/bin/llama-server"
echo -e "${GREEN}llama-server binary copied${NC}"

# Download model
echo ""
echo -e "${YELLOW}Downloading model (this may take a while)...${NC}"
echo "Model: ${MODEL_NAME}"
mkdir -p "${BUILD_DIR}/llm/models"
curl -L --progress-bar -o "${BUILD_DIR}/llm/models/${MODEL_NAME}" "${MODEL_URL}"
echo -e "${GREEN}Model downloaded${NC}"

# Build k13d
echo ""
echo -e "${YELLOW}Building k13d...${NC}"
go build -ldflags "-s -w -X main.Version=${VERSION}" -o "${BUILD_DIR}/k13d" ./cmd/kube-ai-dashboard-cli/main.go
echo -e "${GREEN}k13d built${NC}"

# Copy documentation
cp README.md "${BUILD_DIR}/" 2>/dev/null || true
cp LICENSE "${BUILD_DIR}/" 2>/dev/null || true

# Create install script
cat > "${BUILD_DIR}/install.sh" << 'INSTALL_EOF'
#!/bin/bash
# k13d-llm installer
set -e

INSTALL_DIR="${HOME}/.local/share/k13d"

echo "Installing k13d-llm..."

# Create directories
mkdir -p "${INSTALL_DIR}/llm/bin"
mkdir -p "${INSTALL_DIR}/llm/models"
mkdir -p "${HOME}/.local/bin"

# Copy files
cp llm/bin/llama-server "${INSTALL_DIR}/llm/bin/"
chmod +x "${INSTALL_DIR}/llm/bin/llama-server"

cp llm/models/*.gguf "${INSTALL_DIR}/llm/models/"

cp k13d "${HOME}/.local/bin/"
chmod +x "${HOME}/.local/bin/k13d"

echo ""
echo "Installation complete!"
echo ""
echo "Make sure ~/.local/bin is in your PATH:"
echo '  export PATH="$HOME/.local/bin:$PATH"'
echo ""
echo "To start k13d with embedded LLM:"
echo "  k13d --embedded-llm -web"
echo ""
echo "To check status:"
echo "  k13d --embedded-llm-status"
INSTALL_EOF
chmod +x "${BUILD_DIR}/install.sh"

# Create archive
echo ""
echo -e "${YELLOW}Creating archive...${NC}"
ARCHIVE_NAME="k13d-llm_v${VERSION}_${OS}_${ARCH}.tar.gz"
cd dist
tar -czvf "${ARCHIVE_NAME}" -C llm-bundle .
cd ..

# Calculate checksum
sha256sum "dist/${ARCHIVE_NAME}" > "dist/${ARCHIVE_NAME}.sha256"

# Cleanup
rm -rf "${TEMP_DIR}"

echo ""
echo -e "${GREEN}Build complete!${NC}"
echo ""
echo "Archive: dist/${ARCHIVE_NAME}"
echo "Size: $(du -h "dist/${ARCHIVE_NAME}" | cut -f1)"
echo ""
echo "To install:"
echo "  tar -xzf ${ARCHIVE_NAME}"
echo "  cd k13d-llm_v${VERSION}_${OS}_${ARCH}"
echo "  ./install.sh"
