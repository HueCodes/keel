# Complex multi-stage Dockerfile for benchmarking
# This tests the full range of Dockerfile features

# syntax=docker/dockerfile:1

# Build arguments
ARG GO_VERSION=1.21
ARG ALPINE_VERSION=3.18
ARG NODE_VERSION=18

# =============================================================================
# Stage 1: Build Go backend
# =============================================================================
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS go-builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    ca-certificates

# Set working directory
WORKDIR /build

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always)" \
    -o /app/server \
    ./cmd/server

# =============================================================================
# Stage 2: Build Node.js frontend
# =============================================================================
FROM node:${NODE_VERSION}-alpine AS node-builder

# Set environment
ENV NODE_ENV=production
ENV CI=true

WORKDIR /build

# Copy package files first for better caching
COPY frontend/package*.json ./

# Install dependencies
RUN npm ci --only=production --ignore-scripts

# Copy frontend source
COPY frontend/src/ ./src/
COPY frontend/public/ ./public/
COPY frontend/tsconfig.json ./
COPY frontend/vite.config.ts ./

# Build frontend
RUN npm run build

# =============================================================================
# Stage 3: Build documentation
# =============================================================================
FROM python:3.11-alpine AS docs-builder

WORKDIR /docs

# Install mkdocs and dependencies
RUN pip install --no-cache-dir \
    mkdocs \
    mkdocs-material \
    mkdocs-minify-plugin \
    pymdown-extensions

# Copy documentation source
COPY docs/ ./
COPY mkdocs.yml ./

# Build documentation
RUN mkdocs build

# =============================================================================
# Stage 4: Development image with hot reload
# =============================================================================
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS development

RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    bash \
    curl

# Install air for hot reload
RUN go install github.com/cosmtrek/air@latest

WORKDIR /app

# Copy go files
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Expose ports
EXPOSE 8080 8081

# Development command
CMD ["air", "-c", ".air.toml"]

# =============================================================================
# Stage 5: Test runner
# =============================================================================
FROM go-builder AS test-runner

# Install test dependencies
RUN go install github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt@latest
RUN go install gotest.tools/gotestsum@latest

# Run tests with coverage
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go test -v -race -coverprofile=coverage.out ./... 2>&1 | gotestfmt

# Generate coverage report
RUN go tool cover -html=coverage.out -o coverage.html

# =============================================================================
# Stage 6: Security scanner
# =============================================================================
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS security-scan

RUN apk add --no-cache git

# Install security tools
RUN go install github.com/securego/gosec/v2/cmd/gosec@latest
RUN go install golang.org/x/vuln/cmd/govulncheck@latest

WORKDIR /app
COPY --from=go-builder /build /app

# Run security scans
RUN gosec ./... || true
RUN govulncheck ./... || true

# =============================================================================
# Stage 7: Final production image
# =============================================================================
FROM alpine:${ALPINE_VERSION} AS production

# Labels
LABEL org.opencontainers.image.title="MyApp" \
      org.opencontainers.image.description="Production application" \
      org.opencontainers.image.version="1.0.0" \
      org.opencontainers.image.vendor="MyCompany" \
      org.opencontainers.image.source="https://github.com/mycompany/myapp" \
      org.opencontainers.image.licenses="MIT" \
      maintainer="team@mycompany.com"

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    tini

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -u 1000 -G app -s /bin/sh -D app

# Set timezone
ENV TZ=UTC

# Create directories
RUN mkdir -p /app/data /app/logs /app/config && \
    chown -R app:app /app

WORKDIR /app

# Copy artifacts from build stages
COPY --from=go-builder --chown=app:app /app/server ./
COPY --from=node-builder --chown=app:app /build/dist ./public/
COPY --from=docs-builder --chown=app:app /docs/site ./docs/

# Copy configuration
COPY --chown=app:app config/production.yaml ./config/

# Set user
USER app

# Expose ports
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

# Use tini as init
ENTRYPOINT ["/sbin/tini", "--"]

# Start the application
CMD ["./server", "--config", "./config/production.yaml"]

# =============================================================================
# Stage 8: Debug image with tools
# =============================================================================
FROM production AS debug

USER root

# Install debugging tools
RUN apk add --no-cache \
    bash \
    strace \
    ltrace \
    gdb \
    busybox-extras \
    net-tools \
    bind-tools \
    tcpdump \
    htop

# Add debug entrypoint
COPY --chown=root:root scripts/debug-entrypoint.sh /usr/local/bin/

USER app

ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/debug-entrypoint.sh"]
CMD ["./server", "--debug"]
