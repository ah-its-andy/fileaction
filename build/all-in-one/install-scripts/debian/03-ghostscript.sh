#!/bin/bash
# Install Ghostscript for PDF processing on Debian

set -e

echo "Installing Ghostscript..."

apt-get update
apt-get install -y --no-install-recommends \
    ghostscript \
    gsfonts

rm -rf /var/lib/apt/lists/*

echo "Ghostscript installed successfully"
