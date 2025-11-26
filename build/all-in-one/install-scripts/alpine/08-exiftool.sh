#!/bin/sh
# Install exiftool for metadata manipulation on Alpine

set -e

echo "Installing exiftool..."

apk add --no-cache \
    exiftool

echo "exiftool installed successfully"
