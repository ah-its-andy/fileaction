# FileAction

FileAction is a lightweight workflow automation engine inspired by GitHub Actions, designed specifically for file format conversions and batch processing. It provides a simple YAML-based workflow definition, automatic file scanning with MD5-based change detection, and a clean web UI for monitoring tasks.

## Features

- **YAML-Based Workflows**: Define conversion workflows with simple YAML syntax
- **Workflow Management**: Create, edit, delete, enable/disable workflows via web UI
- **Automatic File Scanning**: Recursively scan directories and track file changes with MD5 hashing
- **Smart Task Creation**: Only process new or modified files (skip_on_nochange)
- **Concurrent Execution**: Configurable worker pools for parallel task processing
- **Real-time Monitoring**: Web UI with live log streaming for running tasks
- **Pure Go Backend**: No CGO dependencies, cross-platform compatible
- **Single Binary Deployment**: Everything bundled in one executable
- **SQLite Storage**: Embedded database using modernc.org/sqlite (no CGO)
- **Default Workflow**: Pre-configured JPEG to HEIC conversion workflow included

## Architecture

### Backend (Go + Fiber)
- **Fiber Framework**: High-performance HTTP server with middleware support
- **SQLite Database**: Pure Go SQLite implementation for data persistence
- **Workflow Engine**: YAML parser, variable substitution, and task scheduler
- **File Scanner**: MD5-based change detection and file indexing
- **Task Executor**: Worker pool with timeout control and log management

### Frontend (Vanilla JS SPA)
- **Native JavaScript**: No frameworks, pure DOM manipulation
- **Real-time Updates**: Polling-based log streaming for running tasks
- **GitHub Actions Style**: Clean, familiar UI design
- **Hash-based Routing**: Single-page navigation without page reloads

## Quick Start

### Prerequisites

- Go 1.21 or higher
- ImageMagick (for image conversion workflows)
- Pandoc (optional, for document conversion)

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/fileaction.git
cd fileaction

# Build the binary
go mod download
go build -o fileaction .

# Run the server
./fileaction
```

The server will start on `http://localhost:8080` by default.

### Default Workflow

FileAction comes with a pre-configured **JPEG to HEIC** conversion workflow that is automatically created on first run:

- **Name**: `convert-jpeg-to-heic`
- **Purpose**: Convert JPEG images to HEIC format for better compression
- **Location**: Watches `./images` directory
- **Features**: 
  - Uses ImageMagick for conversion
  - Quality set to 85
  - Processes only new or changed files
  - 2 concurrent workers

To use the default workflow:
1. Create an `./images` directory
2. Add your JPEG files (`.jpg` extension)
3. Open the web UI at `http://localhost:8080`
4. Click "🔍 Scan" on the workflow
5. Monitor task execution in the Tasks view

### Using Docker

```bash
# Build the Docker image
docker build -t fileaction .

# Run with docker-compose
docker-compose up -d
```

## Configuration

Edit `config/config.yaml` to customize settings:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  path: "./data/fileaction.db"

logging:
  dir: "./data/logs"

execution:
  default_concurrency: 4
  max_concurrency: 16
  task_timeout: 3600s
  step_timeout: 1800s
```

Environment variables can override config values:
- `CONFIG_PATH`: Path to config file
- `DB_PATH`: Database file path
- `LOG_DIR`: Log directory path

## Workflow Definition

### Basic Structure

```yaml
name: my-workflow
description: Description of what this workflow does
on:
  paths:
    - ./input/path
    - ./another/path
convert:
  from: jpg
  to: png
steps:
  - name: step-name
    run: command "${{ input_path }}" "${{ output_path }}"
    env:
      VAR_NAME: value
options:
  concurrency: 4
  include_subdirs: true
  file_glob: "*.jpg"
  skip_on_nochange: true
```

### Available Variables

- `${{ input_path }}`: Full path to input file
- `${{ output_path }}`: Full path to output file
- `${{ file_name }}`: Filename with extension
- `${{ file_dir }}`: Directory containing the file
- `${{ file_base }}`: Filename without extension
- `${{ file_ext }}`: File extension

### Example Workflows

#### JPEG to HEIC Conversion

```yaml
name: convert-jpeg-to-heic
on:
  paths:
    - ./images
convert:
  from: jpeg
  to: heic
steps:
  - name: imagemagick-convert
    run: magick convert "${{ input_path }}" -quality 85 "${{ output_path }}"
    env:
      MAGICK_THREAD_LIMIT: "1"
  - name: verify-conversion
    run: file "${{ output_path }}" | grep -q "HEIC"
options:
  concurrency: 2
  include_subdirs: true
  file_glob: "*.jpg"
  skip_on_nochange: true
```

More examples in `docs/example-workflows/`.

## API Reference

### Workflows

- `GET /api/workflows` - List all workflows
- `POST /api/workflows` - Create a new workflow
- `GET /api/workflows/:id` - Get workflow details
- `PUT /api/workflows/:id` - Update workflow
- `DELETE /api/workflows/:id` - Delete workflow
- `POST /api/workflows/:id/scan` - Trigger scan

### Tasks

- `GET /api/tasks` - List tasks (with filters)
- `GET /api/tasks/:id` - Get task details
- `GET /api/tasks/:id/steps` - Get task steps
- `GET /api/tasks/:id/log/tail` - Stream task logs
- `POST /api/tasks/:id/retry` - Retry failed task
- `POST /api/tasks/:id/cancel` - Cancel running task
- `DELETE /api/tasks/:id` - Delete task

### Files

- `GET /api/files?workflow_id=:id` - List indexed files

## Database Schema

### workflows
- `id`: UUID primary key
- `name`: Unique workflow name
- `description`: Optional description
- `yaml_content`: YAML workflow definition
- `enabled`: Boolean flag
- `created_at`, `updated_at`: Timestamps

### files
- `id`: UUID primary key
- `workflow_id`: Foreign key to workflows
- `file_path`: Absolute file path
- `file_md5`: MD5 hash for change detection
- `file_size`: File size in bytes
- `last_scanned_at`: Last scan timestamp

### tasks
- `id`: UUID primary key
- `workflow_id`: Foreign key to workflows
- `file_id`: Foreign key to files
- `input_path`, `output_path`: File paths
- `status`: pending, running, completed, failed, cancelled
- `log_text`: Full execution log (after completion)
- `error_message`: Error details if failed
- `started_at`, `completed_at`: Timestamps

### task_steps
- `id`: UUID primary key
- `task_id`: Foreign key to tasks
- `name`: Step name from workflow
- `command`: Executed command
- `status`: Step status
- `exit_code`: Command exit code
- `stdout`, `stderr`: Command output

## Development

### Project Structure

```
fileaction/
├── backend/
│   ├── api/          # HTTP server and handlers
│   ├── config/       # Configuration loading
│   ├── database/     # Database layer and repositories
│   ├── executor/     # Task execution engine
│   ├── models/       # Data models
│   ├── scanner/      # File scanning logic
│   └── workflow/     # Workflow parser
├── frontend/
│   ├── index.html    # SPA entry point
│   ├── style.css     # Styling
│   └── app.js        # Frontend logic
├── config/
│   └── config.yaml   # Default configuration
├── docs/
│   └── example-workflows/  # Example YAML files
├── main.go           # Application entry point
├── go.mod            # Go dependencies
├── Dockerfile        # Docker build
└── docker-compose.yml # Docker compose config
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./backend/workflow/
```

### Building

```bash
# Build for current platform
go build -o fileaction .

# Build for Linux (from macOS)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o fileaction-linux .

# Build with embedded frontend
go build -tags embed -o fileaction .
```

## Deployment

### Standalone Binary

```bash
# Copy binary and config
./fileaction

# Or specify config path
CONFIG_PATH=/etc/fileaction/config.yaml ./fileaction
```

### Docker

```bash
# Using docker-compose (recommended)
docker-compose up -d

# Or using Docker directly
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/images:/app/images \
  fileaction
```

### systemd Service

Create `/etc/systemd/system/fileaction.service`:

```ini
[Unit]
Description=FileAction Workflow Engine
After=network.target

[Service]
Type=simple
User=fileaction
WorkingDirectory=/opt/fileaction
ExecStart=/opt/fileaction/fileaction
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

## Troubleshooting

### Database locked errors
- Ensure only one instance is running
- Check file permissions on database file
- WAL mode is enabled by default for better concurrency

### Tasks stuck in "running" state
- Check executor logs in `data/logs/app.log`
- Verify task timeout settings
- Use task cancel endpoint to force stop

### ImageMagick not found
```bash
# macOS
brew install imagemagick

# Ubuntu/Debian
apt-get install imagemagick

# Alpine (Docker)
apk add imagemagick imagemagick-heic
```

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

Inspired by GitHub Actions' workflow syntax and execution model.
