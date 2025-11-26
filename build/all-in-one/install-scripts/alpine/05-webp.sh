#!/bin/sh
# Install WebP tools for WebP image format on Alpine

set -e

echo "Installing WebP tools..."

apk add --no-cache \
    libwebp \
    libwebp-tools

echo "WebP tools installed successfully"
