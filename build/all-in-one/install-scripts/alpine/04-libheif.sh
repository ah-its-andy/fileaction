#!/bin/sh
# Install libheif for HEIF/HEIC image format support on Alpine

set -e

echo "Installing libheif..."

apk add --no-cache \
    libheif \
    libheif-tools

echo "libheif installed successfully"
