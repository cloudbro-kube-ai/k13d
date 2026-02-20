# Multi-architecture build stage
# For standard build, use: docker build -t k13d .
# For pre-built binary, use: docker build -f Dockerfile.prebuilt -t k13d .
#
# Air-gapped/Offline environment:
# 1. Build image on a machine with internet access
# 2. Save: docker save k13d:latest | gzip > k13d.tar.gz
# 3. Transfer to air-gapped environment
# 4. Load: docker load < k13d.tar.gz
#
# Usage in Kubernetes (air-gapped):
# - Ensure image is available in local registry
# - Set imagePullPolicy: IfNotPresent or Never
# - Configure LLM endpoint to internal Ollama/vLLM server

FROM golang:1.25.5-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
ENV GOTOOLCHAIN=auto
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -o k13d ./cmd/kube-ai-dashboard-cli/main.go

# Final stage
FROM alpine:3.21

# Install runtime dependencies
# - ncurses: Required for TUI mode (terminal UI)
# - ca-certificates: For HTTPS connections
# - tzdata: For timezone support
# - curl: For health checks
# - bash: Required for AI agent shell commands
# - kubectl: Required for AI agent Kubernetes operations
RUN apk add --no-cache ca-certificates tzdata curl ncurses bash
RUN apk add --no-cache kubectl

# Create non-root user
RUN adduser -D -g '' k13d

# Copy binary from builder
COPY --from=builder /app/k13d /usr/local/bin/k13d

# Create config, kubeconfig, and data directories
# data directory is writable for SQLite audit DB and user locks
RUN mkdir -p /home/k13d/.config/k13d /home/k13d/.kube /home/k13d/data && \
    chown -R k13d:k13d /home/k13d

# Switch to non-root user
USER k13d
WORKDIR /home/k13d

# Environment variables for configuration
# K13D_AUTH_MODE: token (k8s RBAC - recommended) or local (username/password)
# K13D_USERNAME/K13D_PASSWORD: credentials for local auth mode (optional)
# KUBECONFIG: path to kubeconfig file (mount as volume)
# K13D_LLM_PROVIDER: openai, ollama, gemini, anthropic, bedrock
# K13D_LLM_ENDPOINT: Custom LLM endpoint (e.g., http://ollama:11434 for air-gapped)
# K13D_LLM_MODEL: Model name (e.g., gpt-4o-mini, llama3)
# K13D_LLM_API_KEY: API key for LLM provider
# K13D_JWT_SECRET: Secret key for JWT token signing (auto-generated if not set)
# K13D_DEFAULT_ROLE: Default TUI role (admin, user, viewer)
ENV K13D_PORT=8080 \
    K13D_AUTH_MODE=token \
    K13D_LLM_PROVIDER="" \
    K13D_LLM_MODEL="" \
    K13D_LLM_ENDPOINT="" \
    K13D_JWT_SECRET="" \
    K13D_DEFAULT_ROLE=admin

# Expose web UI port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -sf http://localhost:${K13D_PORT}/api/health || exit 1

# Default command: run in web mode
# TUI mode: docker run -it --rm -v ~/.kube/config:/home/k13d/.kube/config:ro k13d -tui
# Web mode: docker run -d -p 8080:8080 -v ~/.kube/config:/home/k13d/.kube/config:ro k13d
ENTRYPOINT ["/usr/local/bin/k13d"]
CMD ["-web", "-port", "8080"]
