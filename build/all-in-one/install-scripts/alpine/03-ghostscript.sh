#!/bin/sh
# Install Ghostscript for PDF processing on Alpine

set -e

echo "Installing Ghostscript..."

apk add --no-cache \
    ghostscript \
    ghostscript-fonts

echo "Ghostscript installed successfully"
