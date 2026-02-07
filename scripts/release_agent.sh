#!/bin/bash

# FluxQuery Agent Release Script
# This script cross-compiles the agent for major platforms.

set -e

VERSION="v2.0.4"
BIN_NAME="fluxquery-agent"
BUILD_DIR="./bin"
CMD_PATH="./cmd/agent/main.go"

mkdir -p $BUILD_DIR

echo "Building FluxQuery Agent $VERSION..."

# Windows (x64)
echo "Compiling for Windows (x64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $BUILD_DIR/${BIN_NAME}-windows-amd64.exe $CMD_PATH

# Linux (x64)
echo "Compiling for Linux (x64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $BUILD_DIR/${BIN_NAME}-linux-amd64 $CMD_PATH

# macOS (Apple Silicon)
echo "Compiling for macOS (ARM64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $BUILD_DIR/${BIN_NAME}-darwin-arm64 $CMD_PATH

# macOS (Intel)
echo "Compiling for macOS (x64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $BUILD_DIR/${BIN_NAME}-darwin-amd64 $CMD_PATH

echo "Release complete. Binaries located in $BUILD_DIR"
