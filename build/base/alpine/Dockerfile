# Multi-stage build for FileAction - Alpine Base Image
# Supports both amd64 and arm64 architectures

FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application for target architecture
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s" -o fileaction .

# Final stage - minimal alpine image
FROM alpine:latest

ARG TARGETPLATFORM

LABEL maintainer="fileaction"
LABEL description="FileAction - Base Alpine Image (${TARGETPLATFORM})"

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
