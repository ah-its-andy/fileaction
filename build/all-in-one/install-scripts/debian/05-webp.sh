#!/bin/bash
# Install WebP tools for WebP image format on Debian

set -e

echo "Installing WebP tools..."

apt-get update
apt-get install -y --no-install-recommends \
    webp \
    libwebp-dev

rm -rf /var/lib/apt/lists/*

echo "WebP tools installed successfully"
