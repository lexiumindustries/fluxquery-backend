#!/bin/sh
set -e

# FluxQuery Agent Installer
# Repo: https://github.com/lexiumindustries/fluxquery-backend

REPO="lexiumindustries/fluxquery-backend"
BINARY_NAME="fluxquery-agent"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux) ;;
  darwin) ;;
  mingw*|msys*) OS="windows" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

# Detect Arch
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported Architecture: $ARCH"
    exit 1
    ;;
esac

# Construct Asset Name
ASSET_NAME="${BINARY_NAME}-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
  ASSET_NAME="${ASSET_NAME}.exe"
fi

echo "FluxQuery Agent Installer"
echo "========================="
echo "Platform: $OS / $ARCH"
echo "Fetching latest release from $REPO..."

# URL for "latest" release asset
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ASSET_NAME}"

echo "Downloading from: $DOWNLOAD_URL"

if command -v curl >/dev/null 2>&1; then
  curl -fsSL -o "$ASSET_NAME" "$DOWNLOAD_URL"
elif command -v wget >/dev/null 2>&1; then
  wget -q -O "$ASSET_NAME" "$DOWNLOAD_URL"
else
  echo "Error: neither curl nor wget found."
  exit 1
fi

chmod +x "$ASSET_NAME"

echo ""
echo "Successfully downloaded $ASSET_NAME"
echo "Run it with:"
if [ "$OS" = "windows" ]; then
    echo "  ./$ASSET_NAME"
else
    echo "  ./$ASSET_NAME"
fi
