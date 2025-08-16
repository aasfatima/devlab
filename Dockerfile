# Multi-stage build for DevLab API Server
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the API server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o devlab-api ./cmd/api

# Build the worker
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o devlab-worker ./cmd/worker

# Build sample clients
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sample-rest-client ./scripts/clients/sample
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sample-grpc-client ./scripts/clients/grpc

# Final stage
FROM alpine:latest

# Install runtime dependencies including docker-cli
RUN apk --no-cache add ca-certificates tzdata docker-cli

# Create groups and user with correct GIDs for macOS compatibility
# staff group (GID 20) - matches macOS Docker socket group (will be mapped by docker-compose)
# docker group (GID 998) - for Docker operations
# devlab user (UID 1001) - will be overridden by docker-compose
RUN addgroup -g 998 docker && \
    addgroup -g 1001 -S devlab && \
    adduser -u 1001 -S devlab -G devlab && \
    adduser devlab docker

# Set working directory
WORKDIR /app

# Copy binaries from builder stage
COPY --from=builder /app/devlab-api .
COPY --from=builder /app/devlab-worker .
COPY --from=builder /app/sample-rest-client .
COPY --from=builder /app/sample-grpc-client .

# Copy configuration files
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/scripts ./scripts

# Create necessary directories
RUN mkdir -p /app/logs /app/data

# Change ownership to non-root user
RUN chown -R devlab:devlab /app

# Note: Running as root for Docker socket access
# In production, consider using Docker-in-Docker or cloud platforms
# USER devlab

# Expose ports
EXPOSE 8000 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8000/healthz || exit 1

# Default command
CMD ["./devlab-api"]
