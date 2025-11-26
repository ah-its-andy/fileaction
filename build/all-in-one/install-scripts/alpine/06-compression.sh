#!/bin/sh
# Install compression and archiving tools on Alpine

set -e

echo "Installing compression and archiving tools..."

apk add --no-cache \
    zip \
    unzip \
    tar \
    gzip \
    bzip2 \
    xz \
    p7zip

echo "Compression tools installed successfully"
