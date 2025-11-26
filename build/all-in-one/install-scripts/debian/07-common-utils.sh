#!/bin/bash
# Install common utilities on Debian

set -e

echo "Installing common utilities..."

apt-get update
apt-get install -y --no-install-recommends \
    curl \
    wget \
    ca-certificates \
    tzdata \
    bash

rm -rf /var/lib/apt/lists/*

echo "Common utilities installed successfully"
