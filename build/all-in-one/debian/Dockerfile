# Multi-stage build for FileAction - Debian All-in-One Image
# Supports both amd64 and arm64 architectures
# Includes additional libraries and tools

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

# Final stage - debian with additional tools
FROM debian:bullseye-slim

ARG TARGETPLATFORM

LABEL maintainer="fileaction"
LABEL description="FileAction - All-in-One Debian Image (${TARGETPLATFORM})"

# Install ca-certificates first
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create necessary directories
RUN mkdir -p /app/config /app/data /app/logs /app/scripts

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/fileaction /app/fileaction

# Copy default configuration
COPY config/config.yaml /app/config/config.yaml

# Copy installation scripts
COPY build/all-in-one/install-scripts/debian/ /app/scripts/

# Set executable permissions for scripts
RUN chmod +x /app/scripts/*.sh

# Run all installation scripts
RUN for script in /app/scripts/*.sh; do \
        echo "Running $script..."; \
        bash "$script" || exit 1; \
    done

# Clean up installation scripts and apt cache
RUN rm -rf /app/scripts /var/lib/apt/lists/*

# Set executable permissions
RUN chmod +x /app/fileaction

# Expose port
EXPOSE 3000

# Run the application
CMD ["/app/fileaction"]
