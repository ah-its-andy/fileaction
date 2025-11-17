# Quick Start Guide

This guide will help you get FileAction up and running in 5 minutes.

## Installation

### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/fileaction.git
cd fileaction

# Download dependencies
go mod download

# Build the binary
make build

# Run the server
./fileaction
```

### Option 2: Using Docker

```bash
# Clone the repository
git clone https://github.com/yourusername/fileaction.git
cd fileaction

# Start with Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f
```

The application will be available at http://localhost:8080

## Using the Default Workflow

FileAction automatically creates a **JPEG to HEIC** conversion workflow on first run. This is the quickest way to get started:

### 1. Create the images directory

```bash
mkdir -p ./images
```

### 2. Add test JPEG files

```bash
# Create test images (if you have ImageMagick installed)
convert -size 100x100 xc:blue ./images/test1.jpg
convert -size 100x100 xc:red ./images/test2.jpg

# Or copy your own JPG files
cp /path/to/your/photos/*.jpg ./images/
```

### 3. Access the Web UI

Open http://localhost:8080 in your browser.

### 4. Scan the Default Workflow

1. You'll see the **"convert-jpeg-to-heic"** workflow already created
2. Click the **"🔍 Scan"** button
3. The system will detect your JPEG files

### 5. Monitor Tasks

1. Click **"Tasks"** in the sidebar
2. Watch as tasks are created and executed
3. Click **"👁️ View"** on any task to see real-time logs

The converted HEIC files will be saved in the same directory as the originals.

## First Workflow

### 1. Create a Test Directory

```bash
mkdir -p ./images
```

### 2. Access the Web UI

Open http://localhost:8080 in your browser.

### 3. Create a Workflow

1. Click **"+ New Workflow"**
2. Fill in the form:

**Name:** `test-conversion`

**Description:** `Test image conversion`

**YAML Content:**
```yaml
name: test-conversion
description: Convert JPG to PNG
on:
  paths:
    - ./images
convert:
  from: jpg
  to: png
steps:
  - name: convert-image
    run: convert "${{ input_path }}" "${{ output_path }}"
options:
  concurrency: 2
  include_subdirs: true
  file_glob: "*.jpg"
  skip_on_nochange: true
```

3. Check **"Enabled"**
4. Click **"Save"**

### 4. Add Test Files

```bash
# Create a test image (if you have ImageMagick installed)
convert -size 100x100 xc:blue ./images/test1.jpg
convert -size 100x100 xc:red ./images/test2.jpg

# Or copy your own JPG files
cp /path/to/your/images/*.jpg ./images/
```

### 5. Scan the Workflow

1. In the Workflows view, find your workflow
2. Click **"🔍 Scan"**
3. A message will confirm the scan has started

### 6. Monitor Tasks

1. Click **"Tasks"** in the sidebar
2. You'll see tasks for each detected JPG file
3. Watch the status change from:
   - 🟡 Pending → 🔵 Running → 🟢 Completed

### 7. View Task Logs

1. Click **"👁️ View"** on any task
2. See execution details and real-time logs
3. Check step-by-step output

## Common Workflows

### JPEG to HEIC (macOS/iOS format)

```yaml
name: jpeg-to-heic
on:
  paths:
    - ./photos
convert:
  from: jpeg
  to: heic
steps:
  - name: convert
    run: magick convert "${{ input_path }}" -quality 85 "${{ output_path }}"
options:
  concurrency: 2
  include_subdirs: true
  file_glob: "*.jpg"
  skip_on_nochange: true
```

### PNG to WebP (web optimization)

```yaml
name: png-to-webp
on:
  paths:
    - ./website/images
convert:
  from: png
  to: webp
steps:
  - name: convert
    run: cwebp -q 85 "${{ input_path }}" -o "${{ output_path }}"
options:
  concurrency: 4
  include_subdirs: true
  file_glob: "*.png"
  skip_on_nochange: true
```

### Markdown to PDF

```yaml
name: markdown-to-pdf
on:
  paths:
    - ./documents
convert:
  from: md
  to: pdf
steps:
  - name: pandoc-convert
    run: pandoc "${{ input_path }}" -o "${{ output_path }}" --pdf-engine=xelatex
  - name: verify
    run: test -f "${{ output_path }}"
options:
  concurrency: 2
  include_subdirs: true
  file_glob: "*.md"
  skip_on_nochange: true
```

## Features Overview

### Smart File Tracking
- Automatically indexes files with MD5 hashes
- Only processes new or changed files
- Skips unchanged files to save time

### Concurrent Processing
- Configurable worker pools
- Process multiple files in parallel
- Adjust concurrency per workflow

### Real-time Monitoring
- Live log streaming for running tasks
- Task status updates
- Step-by-step execution details

### Flexible Workflows
- YAML-based definitions
- Variable substitution
- Multi-step pipelines
- Environment variables per step

## Troubleshooting

### Issue: Tasks stay in "pending" status

**Solution:** Check that the executor is running:
```bash
# Check logs
tail -f data/logs/app.log

# Restart the application
make run
```

### Issue: "convert: command not found"

**Solution:** Install ImageMagick:
```bash
# macOS
brew install imagemagick

# Ubuntu/Debian
sudo apt-get install imagemagick

# Alpine (Docker)
apk add imagemagick
```

### Issue: Database locked

**Solution:** Ensure only one instance is running:
```bash
# Stop any running instances
pkill fileaction

# Restart
./fileaction
```

### Issue: Files not detected

**Solution:** 
1. Check the `file_glob` pattern matches your files
2. Verify the path in `on.paths` is correct
3. Ensure `include_subdirs` is set if files are in subdirectories

## Next Steps

- [Read the full documentation](README.md)
- [Explore API endpoints](docs/API.md)
- [Learn about development](docs/DEVELOPMENT.md)
- [Check example workflows](docs/example-workflows/)

## Configuration

### Customize Settings

Edit `config/config.yaml`:

```yaml
# Change server port
server:
  port: 3000

# Increase concurrency
execution:
  default_concurrency: 8

# Adjust timeouts
execution:
  task_timeout: 7200s  # 2 hours
  step_timeout: 3600s  # 1 hour
```

### Use Environment Variables

```bash
# Custom config path
CONFIG_PATH=/etc/fileaction/config.yaml ./fileaction

# Override database path
DB_PATH=/var/lib/fileaction/db.sqlite ./fileaction

# Override log directory
LOG_DIR=/var/log/fileaction ./fileaction
```

## Production Deployment

### Using Docker

```yaml
# docker-compose.yml
version: '3.8'
services:
  fileaction:
    image: fileaction:latest
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
      - ./files:/app/files
    restart: unless-stopped
```

### Using systemd

```ini
# /etc/systemd/system/fileaction.service
[Unit]
Description=FileAction
After=network.target

[Service]
Type=simple
User=fileaction
ExecStart=/opt/fileaction/fileaction
Restart=always

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable fileaction
sudo systemctl start fileaction
```

## Getting Help

- Open an issue on GitHub
- Check the documentation in `docs/`
- Review example workflows in `docs/example-workflows/`

Enjoy using FileAction! 🚀
