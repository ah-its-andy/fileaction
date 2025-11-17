.PHONY: all build run test clean docker docker-up docker-down

# Build variables
BINARY_NAME=fileaction
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-s -w"

all: test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Build complete: $(BINARY_NAME)"

# Build for Linux (useful for cross-compilation)
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME)-linux .
	@echo "Build complete: $(BINARY_NAME)-linux"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Run without building
run-dev:
	@echo "Running in development mode..."
	$(GO) run .

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME) $(BINARY_NAME)-linux
	rm -f coverage.out coverage.html
	rm -rf data/
	@echo "Clean complete"

# Install dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Dependencies installed"

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Format complete"

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run
	@echo "Lint complete"

# Docker commands
docker:
	@echo "Building Docker image..."
	docker build -t fileaction:latest .
	@echo "Docker image built: fileaction:latest"

docker-up:
	@echo "Starting Docker containers..."
	docker-compose up -d
	@echo "Containers started"

docker-down:
	@echo "Stopping Docker containers..."
	docker-compose down
	@echo "Containers stopped"

docker-logs:
	docker-compose logs -f

# Setup development environment
setup:
	@echo "Setting up development environment..."
	mkdir -p data/logs
	mkdir -p images
	cp -n config/config.yaml config/config.local.yaml || true
	@echo "Setup complete"

# Run with custom config
run-config:
	CONFIG_PATH=./config/config.local.yaml ./$(BINARY_NAME)

help:
	@echo "FileAction Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build         - Build the binary"
	@echo "  make build-linux   - Build for Linux"
	@echo "  make run           - Build and run"
	@echo "  make run-dev       - Run without building"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make deps          - Install dependencies"
	@echo "  make fmt           - Format code"
	@echo "  make lint          - Lint code"
	@echo "  make docker        - Build Docker image"
	@echo "  make docker-up     - Start with docker-compose"
	@echo "  make docker-down   - Stop docker-compose"
	@echo "  make setup         - Setup development environment"
