#!/bin/sh
# Install FFmpeg for video/audio processing on Alpine

set -e

echo "Installing FFmpeg..."

apk add --no-cache \
    ffmpeg \
    ffmpeg-dev

echo "FFmpeg installed successfully"
