#!/bin/bash
# Install libheif for HEIF/HEIC image format support on Debian

set -e

echo "Installing libheif..."

apt-get update
apt-get install -y --no-install-recommends \
    libheif1 \
    libheif-dev \
    heif-gdk-pixbuf \
    heif-thumbnailer

rm -rf /var/lib/apt/lists/*

echo "libheif installed successfully"
