#!/bin/bash
# Install compression and archiving tools on Debian

set -e

echo "Installing compression and archiving tools..."

apt-get update
apt-get install -y --no-install-recommends \
    zip \
    unzip \
    tar \
    gzip \
    bzip2 \
    xz-utils \
    p7zip-full

rm -rf /var/lib/apt/lists/*

echo "Compression tools installed successfully"
