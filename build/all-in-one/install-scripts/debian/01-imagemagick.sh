#!/bin/bash
# Install ImageMagick for image processing on Debian

set -e

echo "Installing ImageMagick..."

apt-get update
apt-get install -y --no-install-recommends \
    imagemagick \
    libmagickwand-dev

rm -rf /var/lib/apt/lists/*

echo "ImageMagick installed successfully"
