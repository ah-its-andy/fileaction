# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy source code (including vendor if it exists)
COPY . .

# Build the application with vendor support
# If vendor directory exists, use -mod=vendor flag
RUN if [ -d "vendor" ]; then \
        echo "Building with vendor mode..."; \
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -ldflags="-s -w" -o fileaction .; \
    else \
        echo "Building with downloaded modules..."; \
         go mod download && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o fileaction .; \
    fi

# Base runtime stage with all dependencies (cacheable)
FROM alpine:latest AS runtime-base

# Install all runtime dependencies
RUN apk add --no-cache \
    imagemagick \
    imagemagick-heic \
    libwebp-tools \
    file \
    ca-certificates \
    tzdata \
    wget

# Set environment variables for ImageMagick
ENV PATH="/usr/bin:${PATH}" \
    MAGICK_HOME="/usr" \
    LD_LIBRARY_PATH="/usr/lib:${LD_LIBRARY_PATH}"

# Verify required commands are available
RUN echo "Verifying installed commands..." && \
    command -v magick || echo "Warning: magick not found" && \
    command -v convert || echo "Warning: convert not found" && \
    command -v cwebp || echo "Warning: cwebp not found" && \
    command -v pandoc || echo "Warning: pandoc not found" && \
    echo "Command verification complete"

# Final runtime stage
FROM runtime-base

# Create app user
RUN addgroup -g 1000 fileaction && \
    adduser -D -u 1000 -G fileaction fileaction

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/fileaction .

# Copy static files
COPY --chown=fileaction:fileaction frontend ./frontend
COPY --chown=fileaction:fileaction config ./config
COPY --chown=fileaction:fileaction docs ./docs

# Create necessary directories
RUN mkdir -p /app/data/logs && \
    mkdir -p /app/images && \
    chown -R fileaction:fileaction /app

# Switch to non-root user
USER fileaction

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Run the application
CMD ["./fileaction"]
