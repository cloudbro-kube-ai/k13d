# Docker Deployment

Deploy k13d using Docker for quick setup and consistent environments.

## Quick Start

```bash
# Run with kubeconfig mounted
docker run -d \
  --name k13d \
  -p 8080:8080 \
  -v ~/.kube:/root/.kube:ro \
  -e OPENAI_API_KEY=$OPENAI_API_KEY \
  cloudbro/k13d:latest \
  -web -port 8080
```

Access at: http://localhost:8080

## Docker Images

### Available Tags

| Tag | Description |
|-----|-------------|
| `latest` | Latest stable release |
| `v0.6.x` | Specific version |
| `edge` | Development builds |
| `slim` | Minimal image (no embedded LLM) |

### Image Size

| Tag | Size | Includes |
|-----|------|----------|
| `latest` | ~100MB | Full features |
| `slim` | ~30MB | No embedded LLM |

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENAI_API_KEY` | OpenAI API key | - |
| `ANTHROPIC_API_KEY` | Anthropic API key | - |
| `KUBECONFIG` | Kubeconfig path | /root/.kube/config |
| `K13D_PASSWORD` | Web UI password | - |
| `K13D_PORT` | Web server port | 8080 |

### Volume Mounts

| Host Path | Container Path | Purpose |
|-----------|----------------|---------|
| `~/.kube` | `/root/.kube` | Kubeconfig |
| `~/.config/k13d` | `/root/.config/k13d` | Config & data |

## Docker Compose

### Basic Setup

```yaml
# docker-compose.yaml
version: '3.8'

services:
  k13d:
    image: cloudbro/k13d:latest
    ports:
      - "8080:8080"
    volumes:
      - ~/.kube:/root/.kube:ro
      - k13d-data:/root/.config/k13d
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - K13D_PASSWORD=${K13D_PASSWORD}
    command: ["-web", "-port", "8080"]
    restart: unless-stopped

volumes:
  k13d-data:
```

Run:
```bash
docker-compose up -d
```

### With Ollama (Local LLM)

```yaml
version: '3.8'

services:
  k13d:
    image: cloudbro/k13d:latest
    ports:
      - "8080:8080"
    volumes:
      - ~/.kube:/root/.kube:ro
      - k13d-data:/root/.config/k13d
    environment:
      - LLM_PROVIDER=ollama
      - LLM_ENDPOINT=http://ollama:11434
    command: ["-web", "-port", "8080"]
    depends_on:
      - ollama

  ollama:
    image: ollama/ollama:latest
    volumes:
      - ollama-models:/root/.ollama
    # Pull model on start
    entrypoint: ["/bin/sh", "-c", "ollama serve & sleep 5 && ollama pull llama3.2 && wait"]

volumes:
  k13d-data:
  ollama-models:
```

### With Traefik (HTTPS)

```yaml
version: '3.8'

services:
  k13d:
    image: cloudbro/k13d:latest
    volumes:
      - ~/.kube:/root/.kube:ro
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    command: ["-web", "-port", "8080"]
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.k13d.rule=Host(`k13d.example.com`)"
      - "traefik.http.routers.k13d.entrypoints=websecure"
      - "traefik.http.routers.k13d.tls.certresolver=letsencrypt"

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
      - "--certificatesresolvers.letsencrypt.acme.email=admin@example.com"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web"

volumes:
  traefik-certs:
```

## Advanced Configuration

### Custom Config File

```yaml
# docker-compose.yaml
services:
  k13d:
    image: cloudbro/k13d:latest
    volumes:
      - ~/.kube:/root/.kube:ro
      - ./k13d-config.yaml:/root/.config/k13d/config.yaml:ro
```

```yaml
# k13d-config.yaml
llm:
  provider: openai
  model: gpt-4

auth:
  password: ${K13D_PASSWORD}

enable_audit: true
beginner_mode: false

mcp:
  servers:
    - name: thinking
      enabled: true
      command: npx
      args: ["-y", "@modelcontextprotocol/server-sequential-thinking"]
```

### Multi-Cluster Setup

```yaml
version: '3.8'

services:
  k13d-prod:
    image: cloudbro/k13d:latest
    ports:
      - "8081:8080"
    volumes:
      - ~/.kube/prod-config:/root/.kube/config:ro
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    command: ["-web", "-port", "8080"]

  k13d-staging:
    image: cloudbro/k13d:latest
    ports:
      - "8082:8080"
    volumes:
      - ~/.kube/staging-config:/root/.kube/config:ro
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    command: ["-web", "-port", "8080"]
```

## Building Custom Images

### Dockerfile

```dockerfile
FROM cloudbro/k13d:latest

# Add custom config
COPY config.yaml /root/.config/k13d/config.yaml

# Add custom scripts
COPY scripts/ /opt/k13d/scripts/

# Set environment
ENV K13D_PORT=8080
ENV LLM_PROVIDER=openai

EXPOSE 8080

CMD ["-web", "-port", "8080"]
```

### Build & Push

```bash
docker build -t myorg/k13d:custom .
docker push myorg/k13d:custom
```

## Health Checks

### Docker Health Check

```yaml
services:
  k13d:
    image: cloudbro/k13d:latest
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

### Monitoring

```bash
# Check container status
docker ps

# View logs
docker logs k13d

# View resource usage
docker stats k13d
```

## Networking

### Bridge Network (Default)

```yaml
services:
  k13d:
    networks:
      - k13d-network

networks:
  k13d-network:
    driver: bridge
```

### Host Network

For accessing local Kubernetes:

```yaml
services:
  k13d:
    network_mode: host
```

### Access from Host

```bash
# Port mapping
docker run -p 8080:8080 ...

# Access
curl http://localhost:8080/api/health
```

## Security

### Read-Only Kubeconfig

```yaml
volumes:
  - ~/.kube:/root/.kube:ro  # Read-only mount
```

### Non-Root User

```dockerfile
FROM cloudbro/k13d:latest

# Create non-root user
RUN adduser -D -u 1000 k13d
USER k13d

# Copy kubeconfig
COPY --chown=k13d:k13d kubeconfig /home/k13d/.kube/config
```

### Secrets Management

```yaml
services:
  k13d:
    secrets:
      - openai_key
    environment:
      - OPENAI_API_KEY_FILE=/run/secrets/openai_key

secrets:
  openai_key:
    file: ./secrets/openai_key.txt
```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs k13d

# Common issues:
# - Missing kubeconfig
# - Invalid API key
# - Port already in use
```

### Can't Access Kubernetes

```bash
# Verify kubeconfig mount
docker exec k13d cat /root/.kube/config

# Test kubectl
docker exec k13d kubectl get nodes
```

### Permission Denied

```bash
# Check file permissions
ls -la ~/.kube/config

# Fix permissions
chmod 644 ~/.kube/config
```

### High Memory Usage

```yaml
services:
  k13d:
    deploy:
      resources:
        limits:
          memory: 512M
```

## Best Practices

### 1. Use Specific Tags

```yaml
# Good
image: cloudbro/k13d:v0.6.3

# Avoid in production
image: cloudbro/k13d:latest
```

### 2. Persist Data

```yaml
volumes:
  - k13d-data:/root/.config/k13d
```

### 3. Use Docker Secrets

Never commit API keys to docker-compose files.

### 4. Enable Health Checks

Always configure health checks for production.

### 5. Resource Limits

Set memory limits to prevent runaway usage.

## Next Steps

- [Kubernetes Deployment](kubernetes.md) - Deploy to K8s
- [Helm](helm.md) - Helm chart
- [Air-Gapped](air-gapped.md) - Offline deployment
