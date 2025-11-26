# Multi-stage build for FileAction - Debian Base Image
# Supports both amd64 and arm64 architectures

FROM --platform=$BUILDPLATFORM golang:1.21-bullseye AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /build

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    make \
    && rm -rf /var/lib/apt/lists/*

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application for target architecture
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s" -o fileaction .

# Final stage - minimal debian image
FROM debian:bullseye-slim

ARG TARGETPLATFORM

LABEL maintainer="fileaction"
LABEL description="FileAction - Base Debian Image (${TARGETPLATFORM})"

# Install minimal runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create necessary directories
RUN mkdir -p /app/config /app/data /app/logs

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/fileaction /app/fileaction

# Copy default configuration
COPY config/config.yaml /app/config/config.yaml

# Set executable permissions
RUN chmod +x /app/fileaction

# Expose port
EXPOSE 3000

# Run the application
CMD ["/app/fileaction"]
