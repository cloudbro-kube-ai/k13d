# k13d Installation Guide

This guide covers all installation methods for k13d: binary, Docker, and Kubernetes deployments.

## Table of Contents

- [Quick Start](#quick-start)
- [Binary Installation](#binary-installation)
- [Docker Installation](#docker-installation)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Air-Gapped Installation](#air-gapped-installation)

---

## Quick Start

The fastest way to get started:

```bash
# Download and run (macOS/Linux)
curl -sSL https://github.com/cloudbro-kube-ai/k13d/releases/latest/download/k13d-$(uname -s)-$(uname -m).tar.gz | tar xz
./k13d -web -port 8080
```

Open http://localhost:8080 in your browser.

---

## Binary Installation

### Prerequisites

- Go 1.25+ (for building from source)
- kubectl configured with cluster access
- 2GB RAM minimum

### Option 1: Download Pre-built Binary

```bash
# Linux AMD64
curl -LO https://github.com/cloudbro-kube-ai/k13d/releases/latest/download/k13d-linux-amd64.tar.gz
tar -xzf k13d-linux-amd64.tar.gz

# Linux ARM64
curl -LO https://github.com/cloudbro-kube-ai/k13d/releases/latest/download/k13d-linux-arm64.tar.gz
tar -xzf k13d-linux-arm64.tar.gz

# macOS Intel
curl -LO https://github.com/cloudbro-kube-ai/k13d/releases/latest/download/k13d-darwin-amd64.tar.gz
tar -xzf k13d-darwin-amd64.tar.gz

# macOS Apple Silicon
curl -LO https://github.com/cloudbro-kube-ai/k13d/releases/latest/download/k13d-darwin-arm64.tar.gz
tar -xzf k13d-darwin-arm64.tar.gz

# Windows
# Download from GitHub Releases page
```

### Option 2: Build from Source

```bash
git clone https://github.com/cloudbro-kube-ai/k13d.git
cd k13d
go build -o k13d ./cmd/kube-ai-dashboard-cli/main.go
```

### Running

```bash
# TUI Mode (default)
./k13d

# Web Mode
./k13d -web -port 8080

# With authentication disabled (development only)
./k13d -web -port 8080 --no-auth

# With local authentication
./k13d -web -port 8080 --auth-mode local --admin-user admin --admin-password secret
```

### Configuration

Create config file at `~/.config/k13d/config.yaml`:

```yaml
llm:
  provider: solar           # solar, openai, ollama, anthropic
  model: solar-pro2
  endpoint: https://api.upstage.ai/v1
  api_key: your-api-key

language: en                # en, ko, zh, ja
beginner_mode: true
enable_audit: true
```

---

## Docker Installation

See [Docker Installation Guide](./INSTALLATION_DOCKER.md) for detailed instructions.

### Quick Start with Docker

```bash
docker run -d \
  --name k13d \
  -p 8080:8080 \
  -v ~/.kube/config:/home/k13d/.kube/config:ro \
  -e K13D_LLM_PROVIDER=solar \
  -e K13D_LLM_MODEL=solar-pro2 \
  -e K13D_LLM_API_KEY=your-api-key \
  cloudbro-kube-ai/k13d:latest
```

### Docker Compose

```bash
curl -O https://raw.githubusercontent.com/cloudbro-kube-ai/k13d/main/docker-compose.yaml
docker-compose up -d
```

---

## Kubernetes Deployment

See [Kubernetes Installation Guide](./INSTALLATION_K8S.md) for detailed instructions.

### Quick Start with kubectl

```bash
kubectl apply -f https://raw.githubusercontent.com/cloudbro-kube-ai/k13d/main/deploy/kubernetes/all-in-one.yaml
kubectl port-forward -n k13d svc/k13d 8080:80
```

---

## Air-Gapped Installation

For environments without internet access:

### 1. Prepare on Connected Machine

```bash
# Build and save Docker image
docker build -t k13d:latest .
docker save k13d:latest | gzip > k13d-image.tar.gz

# Download embedded LLM model (optional)
./k13d --download-model
tar -czf k13d-llm-models.tar.gz ~/.local/share/k13d/llm/
```

### 2. Transfer and Install

```bash
# On air-gapped machine
docker load < k13d-image.tar.gz

# Restore LLM models
tar -xzf k13d-llm-models.tar.gz -C ~/
```

### 3. Run with Embedded LLM

```bash
docker run -d \
  --name k13d \
  -p 8080:8080 \
  -v ~/.kube/config:/home/k13d/.kube/config:ro \
  -v ~/.local/share/k13d/llm:/home/k13d/.local/share/k13d/llm:ro \
  k13d:latest \
  --embedded-llm
```

---

## Next Steps

- [Configuration Guide](./CONFIGURATION_GUIDE.md) - Detailed configuration options
- [User Guide](./USER_GUIDE.md) - How to use k13d
- [Docker Guide](./INSTALLATION_DOCKER.md) - Docker and Docker Compose setup
- [Kubernetes Guide](./INSTALLATION_K8S.md) - Kubernetes deployment options
