# Development Guide

## Project Structure

```
fileaction/
├── backend/               # Go backend code
│   ├── api/              # HTTP server and API handlers
│   ├── config/           # Configuration management
│   ├── database/         # Database layer with repositories
│   ├── executor/         # Task execution engine
│   ├── models/           # Data models
│   ├── scanner/          # File scanning and indexing
│   └── workflow/         # YAML workflow parser
├── frontend/             # Frontend SPA
│   ├── index.html        # Main HTML file
│   ├── style.css         # Styles (GitHub Actions inspired)
│   └── app.js            # Vanilla JavaScript logic
├── config/               # Configuration files
├── docs/                 # Documentation
│   ├── example-workflows/ # Example YAML workflows
│   └── API.md            # API documentation
├── main.go               # Application entry point
├── go.mod                # Go module definition
├── Makefile              # Build automation
├── Dockerfile            # Docker image definition
└── docker-compose.yml    # Docker Compose configuration
```

## Getting Started

### Prerequisites

- Go 1.21+
- Git
- Make (optional but recommended)
- Docker (optional, for containerized deployment)

### Clone and Setup

```bash
git clone <repository-url>
cd fileaction
make setup
```

### Install Dependencies

```bash
make deps
```

### Run Development Server

```bash
make run-dev
```

The server will start on http://localhost:8080

### Build Binary

```bash
make build
```

This creates a `fileaction` binary in the project root.

## Development Workflow

### 1. Make Changes

Edit files in `backend/` or `frontend/` as needed.

### 2. Format Code

```bash
make fmt
```

### 3. Run Tests

```bash
make test
```

### 4. Build and Test

```bash
make build
./fileaction
```

## Architecture Overview

### Backend Components

**Database Layer** (`backend/database/`)
- Pure Go SQLite implementation (modernc.org/sqlite)
- Repository pattern for data access
- Supports workflows, files, tasks, and task steps
- Automatic schema initialization

**Workflow Engine** (`backend/workflow/`)
- YAML parser using gopkg.in/yaml.v3
- Variable substitution (${{ variable }})
- Validation and error handling
- File glob pattern matching

**Scanner** (`backend/scanner/`)
- Recursive directory scanning
- MD5-based change detection
- File indexing in database
- Smart task creation (skip unchanged files)

**Executor** (`backend/executor/`)
- Worker pool for concurrent task execution
- Shell command execution (os/exec)
- Log file management (text files during execution, DB after completion)
- Timeout control per task and per step
- Cancellation support

**API Server** (`backend/api/`)
- Fiber framework for high performance
- REST endpoints for all operations
- Real-time log streaming (tail endpoint)
- CORS enabled for development

### Frontend Components

**SPA Architecture**
- No frameworks, pure Vanilla JavaScript
- Hash-based routing (#workflows, #tasks, #files)
- Fetch API for HTTP requests
- DOM manipulation for dynamic updates

**UI Components**
- Workflow manager (CRUD operations)
- Task list with filtering and pagination
- Real-time log viewer with polling
- File browser by workflow

**Styling**
- CSS custom properties for theming
- GitHub Actions inspired design
- Dark mode by default
- Responsive layout

## Database Schema

See `backend/database/db.go` for the complete schema initialization.

**Key Tables:**
- `workflows`: Workflow definitions
- `files`: Indexed files with MD5 hashes
- `tasks`: Conversion tasks
- `task_steps`: Individual step execution records

**Indexes:**
- Workflow name (unique)
- File path + workflow (unique composite)
- Task status
- File MD5

## Testing

### Unit Tests

```bash
# Run all tests
make test

# Run specific package
go test ./backend/workflow/

# Run with coverage
make test-coverage
```

### Integration Tests

```bash
# Run integration tests (requires database)
go test -tags=integration ./...
```

### Manual Testing

1. Start the server: `make run`
2. Open http://localhost:8080
3. Create a workflow using the UI
4. Add test files to the watched directory
5. Trigger a scan
6. Monitor task execution in real-time

## Configuration

### Environment Variables

- `CONFIG_PATH`: Path to config file (default: `./config/config.yaml`)
- `DB_PATH`: Override database path
- `LOG_DIR`: Override log directory

### Config File

Edit `config/config.yaml` to customize:
- Server host and port
- Database path
- Log directory
- Execution settings (concurrency, timeouts)
- Polling interval

## Building for Production

### Standalone Binary

```bash
# Build for current platform
make build

# Build for Linux
make build-linux

# Cross-compile for other platforms
GOOS=windows GOARCH=amd64 go build -o fileaction.exe .
```

### Docker Image

```bash
# Build image
make docker

# Run with docker-compose
make docker-up

# View logs
make docker-logs

# Stop
make docker-down
```

## Debugging

### Enable Verbose Logging

Set log level in config:
```yaml
logging:
  level: "debug"
```

### View Application Logs

```bash
tail -f data/logs/app.log
```

### View Task Logs

Task logs are in `data/logs/{task-id}.log` during execution, then moved to database.

### Database Inspection

```bash
sqlite3 data/fileaction.db
.tables
SELECT * FROM workflows;
SELECT * FROM tasks WHERE status = 'failed';
```

## Common Issues

### Database Locked

Only one instance should access the database at a time. WAL mode is enabled for better concurrency.

### Tasks Not Executing

Check:
1. Workflow is enabled
2. Executor is running (check logs)
3. Task status is 'pending'
4. No errors in application log

### ImageMagick Not Found

Install ImageMagick:
```bash
# macOS
brew install imagemagick

# Ubuntu
apt-get install imagemagick

# Docker (already included in Dockerfile)
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make test` and `make fmt`
6. Submit a pull request

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add comments for exported functions
- Keep functions focused and small
- Use meaningful variable names

## Performance Considerations

- Default concurrency is 4 workers
- Adjust based on available CPU cores
- Monitor memory usage for large file batches
- Use file glob patterns to limit scope
- Enable `skip_on_nochange` to avoid redundant work

## Security Notes

- No authentication by default (add reverse proxy for production)
- Shell command execution (sanitize inputs)
- File system access (validate paths)
- CORS enabled (restrict origins for production)

## Future Enhancements

- [ ] WebSocket support for real-time updates
- [ ] Scheduled workflow execution (cron)
- [ ] Workflow templates
- [ ] Batch operations
- [ ] REST API authentication
- [ ] Metrics and monitoring
- [ ] Plugin system for custom steps
- [ ] Multi-node support

## Resources

- [Go Documentation](https://golang.org/doc/)
- [Fiber Documentation](https://docs.gofiber.io/)
- [SQLite Documentation](https://www.sqlite.org/docs.html)
- [YAML Specification](https://yaml.org/spec/)
