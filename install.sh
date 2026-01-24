#!/bin/sh
set -e

REPO="mark3labs/iteratr"
BINARY="iteratr"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Determine OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

# Determine architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Determine version (use argument, env var, or fetch latest)
VERSION="${1:-${ITERATR_VERSION:-}}"
if [ -z "$VERSION" ]; then
    VERSION=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        echo "Failed to determine latest version" >&2
        exit 1
    fi
fi
# Strip leading 'v' if present
VERSION="${VERSION#v}"

echo "Installing ${BINARY} v${VERSION} (${OS}/${ARCH})..."

# Determine archive extension
EXT="tar.gz"
if [ "$OS" = "windows" ]; then
    EXT="zip"
fi

# Download
FILENAME="${BINARY}_${VERSION}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${FILENAME}"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading ${URL}..."
curl -sL "$URL" -o "${TMPDIR}/${FILENAME}"

# Verify checksum if available
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/v${VERSION}/checksums.txt"
if curl -sL "$CHECKSUMS_URL" -o "${TMPDIR}/checksums.txt" 2>/dev/null; then
    EXPECTED=$(grep "${FILENAME}" "${TMPDIR}/checksums.txt" | awk '{print $1}')
    if [ -n "$EXPECTED" ]; then
        if command -v sha256sum >/dev/null 2>&1; then
            ACTUAL=$(sha256sum "${TMPDIR}/${FILENAME}" | awk '{print $1}')
        elif command -v shasum >/dev/null 2>&1; then
            ACTUAL=$(shasum -a 256 "${TMPDIR}/${FILENAME}" | awk '{print $1}')
        else
            ACTUAL=""
        fi
        if [ -n "$ACTUAL" ] && [ "$ACTUAL" != "$EXPECTED" ]; then
            echo "Checksum verification failed!" >&2
            echo "  Expected: ${EXPECTED}" >&2
            echo "  Got:      ${ACTUAL}" >&2
            exit 1
        fi
        echo "Checksum verified."
    fi
fi

# Extract
if [ "$EXT" = "tar.gz" ]; then
    tar -xzf "${TMPDIR}/${FILENAME}" -C "$TMPDIR"
else
    unzip -q "${TMPDIR}/${FILENAME}" -d "$TMPDIR"
fi

# Install
if [ -w "$INSTALL_DIR" ]; then
    mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
    echo "Elevated permissions required to install to ${INSTALL_DIR}"
    sudo mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi
chmod +x "${INSTALL_DIR}/${BINARY}"

echo "Successfully installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
echo "Run '${BINARY} --help' to get started."
