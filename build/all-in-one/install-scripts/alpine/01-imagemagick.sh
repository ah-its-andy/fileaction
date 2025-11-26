#!/bin/sh
# Install ImageMagick for image processing on Alpine

set -e

echo "Installing ImageMagick..."

apk add --no-cache \
    imagemagick \
    imagemagick-dev

echo "ImageMagick installed successfully"
