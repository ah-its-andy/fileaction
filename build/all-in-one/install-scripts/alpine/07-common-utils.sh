#!/bin/sh
# Install common utilities on Alpine

set -e

echo "Installing common utilities..."

apk add --no-cache \
    curl \
    wget \
    ca-certificates \
    tzdata \
    bash

echo "Common utilities installed successfully"
