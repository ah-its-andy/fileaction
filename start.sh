#!/bin/bash

# FileAction Startup Script
# This script helps you quickly start the FileAction server

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}"
echo "╔═══════════════════════════════════════╗"
echo "║         FileAction Startup            ║"
echo "╚═══════════════════════════════════════╝"
echo -e "${NC}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    echo "Please install Go 1.21 or higher from https://golang.org/"
    exit 1
fi

echo -e "${GREEN}✓${NC} Go is installed: $(go version)"

# Check if binary exists
if [ ! -f "./fileaction" ]; then
    echo -e "${YELLOW}Binary not found. Building...${NC}"
    make build
    echo -e "${GREEN}✓${NC} Build complete"
else
    echo -e "${GREEN}✓${NC} Binary found"
fi

# Create necessary directories
echo -e "${YELLOW}Creating directories...${NC}"
mkdir -p data/logs
mkdir -p images
echo -e "${GREEN}✓${NC} Directories ready"

# Check for config file
if [ ! -f "./config/config.yaml" ]; then
    echo -e "${RED}Error: Config file not found${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Configuration file found"

# Check if port 8080 is available
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
    echo -e "${RED}Error: Port 8080 is already in use${NC}"
    echo "Please stop the process using port 8080 or change the port in config.yaml"
    exit 1
fi

echo -e "${GREEN}✓${NC} Port 8080 is available"

# Start the server
echo ""
echo -e "${GREEN}Starting FileAction server...${NC}"
echo ""
echo -e "Access the application at: ${YELLOW}http://localhost:8080${NC}"
echo -e "Press ${YELLOW}Ctrl+C${NC} to stop the server"
echo ""

./fileaction
