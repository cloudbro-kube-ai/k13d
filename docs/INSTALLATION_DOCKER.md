# k13d Docker Installation Guide

This guide covers Docker and Docker Compose installation methods for k13d.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Docker Run Options](#docker-run-options)
- [Docker Compose](#docker-compose)
- [With Ollama (Local AI)](#with-ollama-local-ai)
- [Production Deployment](#production-deployment)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

- Docker 20.10+ or Podman 4.0+
- Access to Kubernetes cluster (kubeconfig)
- (Optional) LLM API key (OpenAI, Solar, etc.)

---

## Quick Start

```bash
# Pull the latest image
docker pull youngjukim/k13d:latest

# Run with kubeconfig mount
docker run -d \
  --name k13d \
  -p 8080:8080 \
  -v ~/.kube/config:/home/k13d/.kube/config:ro \
  youngjukim/k13d:latest

# Open http://localhost:8080
```

---

## Docker Run Options

### Basic Usage

```bash
docker run -d \
  --name k13d \
  -p 8080:8080 \
  -v ~/.kube/config:/home/k13d/.kube/config:ro \
  youngjukim/k13d:latest
```

### With LLM Configuration

```bash
docker run -d \
  --name k13d \
  -p 8080:8080 \
  -v ~/.kube/config:/home/k13d/.kube/config:ro \
  -e K13D_LLM_PROVIDER=solar \
  -e K13D_LLM_MODEL=solar-pro2 \
  -e K13D_LLM_ENDPOINT=https://api.upstage.ai/v1 \
  -e K13D_LLM_API_KEY=your-api-key \
  youngjukim/k13d:latest
```

### With Authentication

```bash
docker run -d \
  --name k13d \
  -p 8080:8080 \
  -v ~/.kube/config:/home/k13d/.kube/config:ro \
  -e K13D_AUTH_MODE=local \
  -e K13D_ADMIN_USER=admin \
  -e K13D_ADMIN_PASSWORD=your-secure-password \
  youngjukim/k13d:latest
```

### With Persistent Data

```bash
docker run -d \
  --name k13d \
  -p 8080:8080 \
  -v ~/.kube/config:/home/k13d/.kube/config:ro \
  -v k13d-data:/home/k13d/.config/k13d \
  youngjukim/k13d:latest
```

### TUI Mode (Interactive)

```bash
docker run -it --rm \
  -v ~/.kube/config:/home/k13d/.kube/config:ro \
  youngjukim/k13d:latest \
  -tui
```

---

## Docker Compose

### Basic Setup

Create `docker-compose.yaml`:

```yaml
version: '3.8'

services:
  k13d:
    image: youngjukim/k13d:latest
    container_name: k13d
    ports:
      - "8080:8080"
    volumes:
      - ~/.kube/config:/home/k13d/.kube/config:ro
      - k13d-data:/home/k13d/.config/k13d
    environment:
      - K13D_LLM_PROVIDER=solar
      - K13D_LLM_MODEL=solar-pro2
      - K13D_LLM_API_KEY=${K13D_LLM_API_KEY}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-sf", "http://localhost:8080/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  k13d-data:
```

Run:

```bash
# Set your API key
export K13D_LLM_API_KEY=your-api-key

# Start
docker-compose up -d

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

---

## With Ollama (Local AI)

Run k13d with Ollama for fully local AI without API keys.

### Docker Compose with Ollama

```yaml
version: '3.8'

services:
  k13d:
    image: youngjukim/k13d:latest
    container_name: k13d
    ports:
      - "8080:8080"
    volumes:
      - ~/.kube/config:/home/k13d/.kube/config:ro
      - k13d-data:/home/k13d/.config/k13d
    environment:
      - K13D_LLM_PROVIDER=ollama
      - K13D_LLM_MODEL=qwen2.5:7b
      - K13D_LLM_ENDPOINT=http://ollama:11434/v1
    depends_on:
      ollama:
        condition: service_healthy
    restart: unless-stopped

  ollama:
    image: ollama/ollama:latest
    container_name: ollama
    ports:
      - "11434:11434"
    volumes:
      - ollama-models:/root/.ollama
    healthcheck:
      test: ["CMD", "curl", "-sf", "http://localhost:11434/api/tags"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

volumes:
  k13d-data:
  ollama-models:
```

### Pull Ollama Model

After starting, pull a model:

```bash
docker exec ollama ollama pull qwen2.5:7b
```

Recommended models:
- `qwen2.5:7b` - Best balance of quality and speed
- `llama3.2:3b` - Lightweight, fast
- `mistral:7b` - Good general purpose

---

## Production Deployment

### With HTTPS (Traefik)

```yaml
version: '3.8'

services:
  k13d:
    image: youngjukim/k13d:latest
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.k13d.rule=Host(`k13d.example.com`)"
      - "traefik.http.routers.k13d.tls=true"
      - "traefik.http.routers.k13d.tls.certresolver=letsencrypt"
    volumes:
      - ~/.kube/config:/home/k13d/.kube/config:ro
      - k13d-data:/home/k13d/.config/k13d
    environment:
      - K13D_AUTH_MODE=local
      - K13D_ADMIN_USER=admin
      - K13D_ADMIN_PASSWORD=${ADMIN_PASSWORD}
      - K13D_LLM_PROVIDER=solar
      - K13D_LLM_API_KEY=${K13D_LLM_API_KEY}
    restart: unless-stopped

  traefik:
    image: traefik:v2.10
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - traefik-certs:/letsencrypt
    command:
      - "--providers.docker=true"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web"
      - "--certificatesresolvers.letsencrypt.acme.email=your@email.com"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"

volumes:
  k13d-data:
  traefik-certs:
```

### Environment Variables Reference

| Variable | Description | Default |
|----------|-------------|---------|
| `K13D_PORT` | Web server port | `8080` |
| `K13D_AUTH_MODE` | Auth mode: `local`, `token`, `ldap` | `token` |
| `K13D_NO_AUTH` | Disable authentication | `false` |
| `K13D_ADMIN_USER` | Admin username | `admin` |
| `K13D_ADMIN_PASSWORD` | Admin password | - |
| `K13D_LLM_PROVIDER` | LLM provider | `ollama` |
| `K13D_LLM_MODEL` | Model name | `llama3` |
| `K13D_LLM_ENDPOINT` | Custom endpoint | - |
| `K13D_LLM_API_KEY` | API key | - |

---

## Troubleshooting

### Cannot connect to Kubernetes cluster

```bash
# Check if kubeconfig is mounted correctly
docker exec k13d cat /home/k13d/.kube/config

# Check if kubectl works inside container
docker exec k13d kubectl cluster-info
```

### LLM not responding

```bash
# Check LLM configuration
docker exec k13d env | grep K13D_LLM

# Test connection manually
docker exec k13d curl -s http://ollama:11434/api/tags
```

### Container keeps restarting

```bash
# Check logs
docker logs k13d

# Check health endpoint
curl http://localhost:8080/api/health
```

### Permission denied on kubeconfig

```bash
# Ensure file is readable
chmod 644 ~/.kube/config

# Or run as root (not recommended for production)
docker run -d --user root ...
```

---

## Next Steps

- [Kubernetes Deployment](./INSTALLATION_K8S.md)
- [Configuration Guide](./CONFIGURATION_GUIDE.md)
- [User Guide](./USER_GUIDE.md)
