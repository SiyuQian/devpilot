#!/bin/sh
set -e

# devpilot installer
# Usage:
#   curl -sSL https://raw.githubusercontent.com/siyuqian/devpilot/main/install.sh | sh
#   curl -sSL ... | sh -s -- --version v0.1.0 --dir ~/.local/bin

REPO="siyuqian/devpilot"
INSTALL_DIR="/usr/local/bin"
VERSION=""

# Parse arguments
while [ $# -gt 0 ]; do
    case "$1" in
        --version) VERSION="$2"; shift 2 ;;
        --dir)     INSTALL_DIR="$2"; shift 2 ;;
        --help)
            echo "Usage: install.sh [--version VERSION] [--dir INSTALL_DIR]"
            echo "  --version  Specific version to install (e.g. v0.1.0). Default: latest"
            echo "  --dir      Installation directory. Default: /usr/local/bin"
            exit 0
            ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
    darwin) ;;
    linux)  ;;
    *)      echo "Error: unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)       echo "Error: unsupported architecture: $ARCH"; exit 1 ;;
esac

BINARY="devpilot-${OS}-${ARCH}"

# Check for supported platform
case "${OS}-${ARCH}" in
    darwin-arm64|darwin-amd64|linux-amd64) ;;
    *) echo "Error: no prebuilt binary for ${OS}-${ARCH}"; exit 1 ;;
esac

# Resolve version
if [ -z "$VERSION" ]; then
    echo "Fetching latest release..."
    VERSION="$(curl -sSL -H "Accept: application/vnd.github+json" \
        "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
    if [ -z "$VERSION" ]; then
        echo "Error: could not determine latest version. Use --version to specify."
        exit 1
    fi
fi

echo "Installing devpilot ${VERSION} (${OS}/${ARCH})..."

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

# Download binary and checksums
curl -sSL -o "${TMPDIR}/${BINARY}" "${BASE_URL}/${BINARY}"
curl -sSL -o "${TMPDIR}/checksums.txt" "${BASE_URL}/checksums.txt"

# Verify checksum
echo "Verifying checksum..."
EXPECTED="$(grep "${BINARY}" "${TMPDIR}/checksums.txt" | awk '{print $1}')"
if [ -z "$EXPECTED" ]; then
    echo "Error: checksum not found for ${BINARY}"
    exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL="$(sha256sum "${TMPDIR}/${BINARY}" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
    ACTUAL="$(shasum -a 256 "${TMPDIR}/${BINARY}" | awk '{print $1}')"
else
    echo "Warning: no sha256 tool found, skipping checksum verification"
    ACTUAL="$EXPECTED"
fi

if [ "$ACTUAL" != "$EXPECTED" ]; then
    echo "Error: checksum mismatch"
    echo "  expected: $EXPECTED"
    echo "  actual:   $ACTUAL"
    exit 1
fi

echo "Checksum verified."

# Install
mkdir -p "$INSTALL_DIR"
if [ -w "$INSTALL_DIR" ]; then
    mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/devpilot"
    chmod +x "${INSTALL_DIR}/devpilot"
else
    echo "Need sudo to install to ${INSTALL_DIR}"
    sudo mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/devpilot"
    sudo chmod +x "${INSTALL_DIR}/devpilot"
fi

echo ""
echo "devpilot ${VERSION} installed to ${INSTALL_DIR}/devpilot"
echo ""
echo "Verify with:"
echo "  devpilot --version"
