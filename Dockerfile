# ═══════════════════════════════════════════════════════════
# GASK - Multi-stage Dockerfile
# Optimized for production with minimal image size
# ═══════════════════════════════════════════════════════════

# ┌─────────────────────────────────────────────────────────┐
# │ Stage 1: Builder                                         │
# └─────────────────────────────────────────────────────────┘
FROM golang:1.21-alpine AS builder

# Build arguments
ARG BUILD_DATE
ARG VERSION=2.0.0

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

# Set working directory
WORKDIR /build

# Copy go mod files first (for better caching)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildDate=${BUILD_DATE}" \
    -o gask \
    .

# ┌─────────────────────────────────────────────────────────┐
# │ Stage 2: Runtime                                         │
# └─────────────────────────────────────────────────────────┘
FROM alpine:latest

# Labels
LABEL maintainer="GASK Team" \
      description="Go-based Advanced taSK management system" \
      version="${VERSION}"

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    curl \
    wget

# Create non-root user
RUN addgroup -g 1000 gask && \
    adduser -D -u 1000 -G gask gask && \
    mkdir -p /home/gask/logs && \
    chown -R gask:gask /home/gask

# Set working directory
WORKDIR /home/gask

# Copy binary from builder
COPY --from=builder --chown=gask:gask /build/gask .

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Switch to non-root user
USER gask

# Expose port
EXPOSE 7890

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:7890/health || exit 1

# Run the application
ENTRYPOINT ["./gask"]