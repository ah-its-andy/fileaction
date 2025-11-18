# FileAction

A lightweight workflow automation engine inspired by GitHub Actions, designed for file format conversions and batch processing. Features YAML-based workflows, MD5-based smart file scanning, concurrent task execution, and a clean web UI for real-time monitoring.

## ✨ Features

- **� YAML Workflows** - Define file processing pipelines with simple, readable syntax
- **� Smart Scanning** - Automatic directory scanning with MD5 change detection
- **⚡ Concurrent Execution** - Configurable worker pools for parallel processing
- **📊 Real-time Monitoring** - Live task logs and status updates via Web UI
- **🎯 Skip Unchanged Files** - Only process new or modified files to save resources
- **🐳 Docker Ready** - One-command deployment with Docker Compose
- **🚀 Single Binary** - Pure Go implementation, no CGO dependencies, cross-platform
- **💾 Flexible Storage** - Supports both SQLite and MySQL databases
- **🎨 Quick Start** - Pre-configured JPEG to HEIC workflow included
- **🔄 Exit Control** - Special exit codes (100, 101) for workflow flow control

## 🚀 Quick Start

## 🚀 Quick Start

### Option 1: Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/ah-its-andy/fileaction.git
cd fileaction

# Start with Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f fileaction
```

### Option 2: Build from Source

```bash
# Prerequisites: Go 1.21+, ImageMagick (for image conversion)
git clone https://github.com/ah-its-andy/fileaction.git
cd fileaction

# Build
go mod download
go build -o fileaction .

# Run
./fileaction
```

Access the web interface at **http://localhost:8080**

### Try the Default Workflow

FileAction includes a pre-configured JPEG to HEIC conversion workflow:

```bash
# 1. Create the images directory
mkdir -p ./images

# 2. Add some JPEG files
cp your-photos/*.jpg ./images/

# 3. Open http://localhost:8080 in your browser
# 4. Click "🔍 Scan" on the "convert-jpeg-to-heic" workflow
# 5. Go to "Tasks" to watch the conversion progress
```

## 📖 Workflow Definition

### Basic Structure

```yaml
name: my-workflow
description: What this workflow does
on:
  paths:
    - ./input/directory
convert:
  from: jpg
  to: png
steps:
  - name: convert-image
    run: convert "${{ input_path }}" "${{ output_path }}"
    env:
      QUALITY: "85"
options:
  concurrency: 4
  include_subdirs: true
  file_glob: "*.jpg"
  skip_on_nochange: true
```

### Available Variables

| Variable | Description |
|----------|-------------|
| `${{ input_path }}` | Full path to input file |
| `${{ output_path }}` | Full path to output file |
| `${{ file_name }}` | Filename with extension |
| `${{ file_dir }}` | Directory containing the file |
| `${{ file_base }}` | Filename without extension |
| `${{ file_ext }}` | File extension |

### Exit Code Control

Use special exit codes to control workflow execution:

| Exit Code | Meaning | Task Status | Continue? |
|-----------|---------|-------------|-----------|
| `0` | Success | Running | ✅ Next step |
| `100` | Success & Stop | **Completed** | ❌ Stop workflow |
| `101` | Failure & Stop | **Failed** | ❌ Stop workflow |
| `1-99, 102+` | Step failed | Failed | ❌ Stop workflow |

**Example: Skip already processed files**

```yaml
steps:
  - name: check-if-processed
    run: |
      if [ -f "${{ output_path }}" ]; then
        echo "File already processed, skipping"
        exit 100  # Success & stop workflow
      fi
  
  - name: process-file
    run: convert "${{ input_path }}" "${{ output_path }}"
```

## 📚 Example Workflows

### JPEG to HEIC

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
options:
  concurrency: 2
  file_glob: "*.jpg"
  skip_on_nochange: true
```

### PNG to WebP

```yaml
name: png-to-webp
on:
  paths:
    - ./images/png
convert:
  from: png
  to: webp
steps:
  - name: convert-to-webp
    run: cwebp -q 85 "${{ input_path }}" -o "${{ output_path }}"
options:
  concurrency: 4
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
options:
  concurrency: 2
  file_glob: "*.md"
  skip_on_nochange: true
```

More examples in [docs/example-workflows/](docs/example-workflows/)

## 🏗️ Architecture

### Backend (Go)
- **Fiber Framework** - High-performance HTTP server
- **Database** - SQLite (modernc.org/sqlite) or MySQL support
- **Workflow Parser** - YAML parsing with variable substitution
- **File Watcher** - MD5-based change detection and indexing
- **Task Scheduler** - Worker pool with concurrent execution
- **Log Management** - Real-time streaming and persistent storage

### Frontend (Vanilla JS)
- **No Framework** - Pure JavaScript, fast and lightweight
- **Hash Routing** - SPA navigation without page reloads
- **Real-time Updates** - Polling-based log streaming
- **GitHub Actions Style** - Clean, familiar interface

### Database Schema

**Main Tables:**
- `workflows` - Workflow definitions and settings
- `files` - Indexed files with MD5 hashes
- `tasks` - Conversion tasks with status tracking
- `task_steps` - Individual step execution records

## ⚙️ Configuration

Edit `config/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  # SQLite
  path: "./data/fileaction.db"
  # MySQL (uncomment to use)
  # path: "user:password@tcp(localhost:3306)/fileaction?charset=utf8mb4&parseTime=True"

logging:
  dir: "./data/logs"

execution:
  default_concurrency: 4
  task_timeout: 3600s
  step_timeout: 1800s
```

### Environment Variables

Override config with environment variables:

```bash
CONFIG_PATH=/etc/fileaction/config.yaml ./fileaction
DB_PATH=./custom/db.sqlite ./fileaction
LOG_DIR=./custom/logs ./fileaction
```

## 🔌 API Reference

### Workflows

- `GET /api/workflows` - List all workflows
- `POST /api/workflows` - Create workflow
- `GET /api/workflows/:id` - Get workflow details
- `PUT /api/workflows/:id` - Update workflow
- `DELETE /api/workflows/:id` - Delete workflow
- `POST /api/workflows/:id/scan` - Trigger scan
- `POST /api/workflows/:id/enable` - Enable workflow
- `POST /api/workflows/:id/disable` - Disable workflow

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

Full API documentation: [docs/API.md](docs/API.md)

## 🐳 Docker Deployment

### Using Docker Compose (with MySQL)

```yaml
version: '3.8'
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root_password
      MYSQL_DATABASE: fileaction
      MYSQL_USER: fileaction
      MYSQL_PASSWORD: fileaction_pass
    volumes:
      - mysql_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5

  fileaction:
    image: fileaction:latest
    ports:
      - "8080:8080"
    environment:
      DB_PATH: "fileaction:fileaction_pass@tcp(mysql:3306)/fileaction?charset=utf8mb4&parseTime=True"
    volumes:
      - ./data/logs:/app/data/logs
      - ./images:/app/images
    depends_on:
      mysql:
        condition: service_healthy
    restart: unless-stopped

volumes:
  mysql_data:
```

### Standalone Docker

```bash
docker build -t fileaction .
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/images:/app/images \
  fileaction
```

## 🛠️ Development

### Project Structure

```
fileaction/
├── backend/
│   ├── api/              # HTTP server and handlers
│   ├── config/           # Configuration management
│   ├── database/         # Database layer & repositories
│   ├── models/           # Data models
│   ├── scheduler/        # Task scheduler & executor pool
│   ├── watcher/          # File watcher & scanner
│   └── workflow/         # YAML parser
├── frontend/
│   ├── index.html        # SPA entry point
│   ├── style.css         # Styling
│   └── app.js            # Frontend logic
├── config/
│   └── config.yaml       # Default configuration
├── docs/                 # Documentation
├── main.go               # Application entry
├── Makefile              # Build automation
├── Dockerfile            # Docker image
└── docker-compose.yml    # Docker Compose config
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./backend/workflow/
```

### Building

```bash
# Build for current platform
make build

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o fileaction-linux .

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o fileaction.exe .
```

## 🔧 Troubleshooting

### Tasks Not Executing

Check logs: `tail -f data/logs/app.log`
- Verify workflow is enabled
- Check executor concurrency settings
- Ensure no resource constraints

### Database Locked (SQLite)

- Only one instance should run with SQLite
- Consider using MySQL for multi-instance deployment
- Check for file permission issues

### ImageMagick Not Found

```bash
# macOS
brew install imagemagick

# Ubuntu/Debian
sudo apt-get install imagemagick

# Alpine (Docker)
apk add imagemagick imagemagick-heic
```

### Files Not Detected

- Verify `file_glob` pattern matches your files
- Check `on.paths` points to correct directory
- Enable `include_subdirs` for nested directories
- Review `ignore` patterns if set

## 📊 Performance Tips

- **Concurrency**: Adjust based on CPU cores (default: 4)
- **File Glob**: Use specific patterns to limit scope
- **Skip Unchanged**: Enable `skip_on_nochange` to avoid redundant work
- **Database**: Use MySQL for better concurrency in production
- **Timeouts**: Tune `task_timeout` and `step_timeout` for your workload

## 🔒 Security Considerations

- **Shell Execution**: Commands run with application privileges
- **Authentication**: No built-in auth, use reverse proxy for production
- **File Access**: Workflows can access any file the user can read
- **Input Validation**: YAML and file paths are validated
- **CORS**: Enabled by default, restrict origins in production

## 🗺️ Roadmap

- [ ] WebSocket support for real-time updates
- [ ] Scheduled workflow execution (cron)
- [ ] Workflow templates marketplace
- [ ] Batch operations support
- [ ] Built-in authentication & authorization
- [ ] Metrics and monitoring dashboard
- [ ] Plugin system for custom steps
- [ ] Multi-node cluster support

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📖 Documentation

- [Quick Start Guide](docs/QUICKSTART.md)
- [API Documentation](docs/API.md)
- [Development Guide](docs/DEVELOPMENT.md)
- [Workflow Exit Control](docs/WORKFLOW_EXIT_CONTROL.md)
- [MySQL Migration Guide](MYSQL_MIGRATION.md)
- [Example Workflows](docs/example-workflows/)

## 🙏 Acknowledgments

Inspired by GitHub Actions' workflow syntax and execution model.

---

**Built with ❤️ using Go and Vanilla JavaScript**
