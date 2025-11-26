#!/bin/bash
# Install FFmpeg for video/audio processing on Debian

set -e

echo "Installing FFmpeg..."

apt-get update
apt-get install -y --no-install-recommends \
    ffmpeg \
    libavcodec-dev \
    libavformat-dev \
    libavutil-dev \
    libswscale-dev

rm -rf /var/lib/apt/lists/*

echo "FFmpeg installed successfully"
