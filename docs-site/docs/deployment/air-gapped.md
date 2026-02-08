# Air-Gapped Deployment

Deploy k13d in environments without internet access.

## Overview

Air-gapped deployment requires:

1. **Pre-downloaded Images**: Container images saved locally
2. **Embedded LLM**: Local AI without API calls
3. **Offline Dependencies**: All assets bundled

## Preparation (Online Machine)

### 1. Download k13d Binary

```bash
# Download binary
wget https://github.com/cloudbro-kube-ai/k13d/releases/download/v0.6.3/k13d-linux-amd64.tar.gz

# Verify checksum
sha256sum k13d-linux-amd64.tar.gz
```

### 2. Save Docker Images

```bash
# Pull images
docker pull cloudbro/k13d:latest
docker pull ollama/ollama:latest

# Save to tar files
docker save cloudbro/k13d:latest > k13d-image.tar
docker save ollama/ollama:latest > ollama-image.tar

# Compress
gzip k13d-image.tar
gzip ollama-image.tar
```

### 3. Download Ollama Models

```bash
# Pull model
ollama pull llama3.2

# Find model location
ls ~/.ollama/models/

# Copy model files
tar -czvf ollama-models.tar.gz ~/.ollama/models/
```

### 4. Package All Files

```bash
# Create package directory
mkdir k13d-airgap
mv k13d-linux-amd64.tar.gz k13d-airgap/
mv k13d-image.tar.gz k13d-airgap/
mv ollama-image.tar.gz k13d-airgap/
mv ollama-models.tar.gz k13d-airgap/

# Create archive
tar -czvf k13d-airgap-bundle.tar.gz k13d-airgap/
```

## Transfer

Transfer `k13d-airgap-bundle.tar.gz` to air-gapped environment:

- USB drive
- Secure file transfer
- Physical media

## Installation (Air-Gapped Machine)

### 1. Extract Bundle

```bash
tar -xzvf k13d-airgap-bundle.tar.gz
cd k13d-airgap
```

### 2. Install Binary

```bash
# Extract binary
tar -xzvf k13d-linux-amd64.tar.gz

# Install
sudo mv k13d /usr/local/bin/
sudo chmod +x /usr/local/bin/k13d

# Verify
k13d --version
```

### 3. Load Docker Images

```bash
# Load images
gunzip -c k13d-image.tar.gz | docker load
gunzip -c ollama-image.tar.gz | docker load

# Verify
docker images | grep -E "(k13d|ollama)"
```

### 4. Install Ollama Models

```bash
# Extract models
mkdir -p ~/.ollama
tar -xzvf ollama-models.tar.gz -C ~/
```

## Running k13d

### With Embedded LLM (Binary)

```bash
# Run with embedded LLM
k13d --embedded-llm -web -port 8080
```

### With Docker + Ollama

```bash
# Start Ollama
docker run -d \
  --name ollama \
  -v ~/.ollama:/root/.ollama \
  ollama/ollama:latest

# Start k13d
docker run -d \
  --name k13d \
  -p 8080:8080 \
  -v ~/.kube:/root/.kube:ro \
  --link ollama \
  -e LLM_PROVIDER=ollama \
  -e LLM_ENDPOINT=http://ollama:11434 \
  cloudbro/k13d:latest \
  -web -port 8080
```

### Docker Compose (Air-Gapped)

```yaml
# docker-compose.airgapped.yaml
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
      - LLM_MODEL=llama3.2
    command: ["-web", "-port", "8080"]
    depends_on:
      - ollama

  ollama:
    image: ollama/ollama:latest
    volumes:
      - ~/.ollama:/root/.ollama

volumes:
  k13d-data:
```

```bash
docker-compose -f docker-compose.airgapped.yaml up -d
```

## Kubernetes Deployment (Air-Gapped)

### 1. Load Images to Registry

```bash
# Load to local registry
docker load < k13d-image.tar.gz
docker tag cloudbro/k13d:latest registry.local:5000/k13d:latest
docker push registry.local:5000/k13d:latest

docker load < ollama-image.tar.gz
docker tag ollama/ollama:latest registry.local:5000/ollama:latest
docker push registry.local:5000/ollama:latest
```

### 2. Deploy with Local Registry

```yaml
# k13d-airgapped.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k13d
spec:
  replicas: 1
  selector:
    matchLabels:
      app: k13d
  template:
    metadata:
      labels:
        app: k13d
    spec:
      containers:
        - name: k13d
          image: registry.local:5000/k13d:latest  # Local registry
          args: ["-web", "-port", "8080"]
          env:
            - name: LLM_PROVIDER
              value: ollama
            - name: LLM_ENDPOINT
              value: http://localhost:11434
          ports:
            - containerPort: 8080

        - name: ollama
          image: registry.local:5000/ollama:latest  # Local registry
          volumeMounts:
            - name: ollama-models
              mountPath: /root/.ollama

      volumes:
        - name: ollama-models
          persistentVolumeClaim:
            claimName: ollama-models
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ollama-models
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

### 3. Pre-load Models

```yaml
# Init container to copy models
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k13d
spec:
  template:
    spec:
      initContainers:
        - name: copy-models
          image: busybox
          command:
            - sh
            - -c
            - cp -r /models/* /ollama-models/
          volumeMounts:
            - name: model-source
              mountPath: /models
            - name: ollama-models
              mountPath: /ollama-models

      volumes:
        - name: model-source
          configMap:
            name: ollama-models  # Or hostPath with models
        - name: ollama-models
          persistentVolumeClaim:
            claimName: ollama-models
```

## Configuration

### config.yaml for Air-Gapped

```yaml
# ~/.config/k13d/config.yaml
llm:
  provider: ollama
  model: llama3.2
  endpoint: http://localhost:11434

# Or embedded LLM
# llm:
#   provider: embedded
#   model: llama-3.2-1b

# Disable telemetry
telemetry:
  enabled: false

# Local settings
enable_audit: true
report_path: /data/reports
```

## Verification

### Check Connectivity

```bash
# Verify no external connections
netstat -tuln | grep ESTABLISHED

# Test LLM
curl http://localhost:11434/api/tags

# Test k13d
curl http://localhost:8080/api/health
```

### Test AI Functionality

```bash
# Access k13d
curl -X POST http://localhost:8080/api/chat/agentic \
  -H "Content-Type: application/json" \
  -d '{"message": "List all pods"}'
```

## Model Options

### Recommended Models for Air-Gapped

| Model | Size | Memory | Quality |
|-------|------|--------|---------|
| Llama 3.2 1B | 1.2GB | 4GB | Good |
| Llama 3.2 3B | 2.5GB | 6GB | Better |
| Llama 3 8B | 4.5GB | 10GB | Best |
| Mistral 7B | 4.0GB | 8GB | Good |
| Qwen2 1.5B | 1.5GB | 4GB | Good |

### Multiple Models

Package multiple models for flexibility:

```bash
# Download multiple models
for model in llama3.2 mistral qwen2:1.5b; do
  ollama pull $model
done

# Save all models
tar -czvf ollama-models-all.tar.gz ~/.ollama/models/
```

## Updating

### Update Process

1. Download new version on online machine
2. Package as before
3. Transfer to air-gapped environment
4. Load new images
5. Restart services

```bash
# On online machine
docker pull cloudbro/k13d:v0.7.0
docker save cloudbro/k13d:v0.7.0 > k13d-v0.7.0.tar
gzip k13d-v0.7.0.tar

# Transfer and load
gunzip -c k13d-v0.7.0.tar.gz | docker load

# Update deployment
kubectl set image deployment/k13d k13d=registry.local:5000/k13d:v0.7.0
```

## Troubleshooting

### Ollama Not Responding

```bash
# Check Ollama status
docker logs ollama

# Verify model is loaded
docker exec ollama ollama list

# Manually load model
docker exec -it ollama ollama run llama3.2 --help
```

### Out of Memory

```bash
# Use smaller model
# In config:
llm:
  model: llama3.2:1b  # Smaller variant
```

### Image Load Fails

```bash
# Verify tar integrity
gzip -t k13d-image.tar.gz

# Check disk space
df -h

# Load with verbose output
docker load < k13d-image.tar
```

## Security Considerations

### Image Verification

```bash
# Verify image digest
docker inspect cloudbro/k13d:latest --format='{{.RepoDigests}}'

# Compare with published digest
```

### Network Isolation

```yaml
# Network policy for air-gapped
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: k13d-airgapped
spec:
  podSelector:
    matchLabels:
      app: k13d
  policyTypes:
    - Egress
  egress:
    # Only allow internal traffic
    - to:
        - ipBlock:
            cidr: 10.0.0.0/8
        - ipBlock:
            cidr: 172.16.0.0/12
        - ipBlock:
            cidr: 192.168.0.0/16
```

## Next Steps

- [Docker Deployment](docker.md) - Standard Docker setup
- [Kubernetes Deployment](kubernetes.md) - K8s deployment
- [Embedded LLM](../ai-llm/embedded.md) - Local AI options
