#!/bin/bash
# Install exiftool for metadata manipulation on Debian

set -e

echo "Installing exiftool..."

apt-get update
apt-get install -y --no-install-recommends \
    libimage-exiftool-perl

rm -rf /var/lib/apt/lists/*

echo "exiftool installed successfully"
