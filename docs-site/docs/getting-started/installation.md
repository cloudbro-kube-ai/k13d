# Installation

k13d can be installed in multiple ways depending on your environment and requirements.

## Prerequisites

- **Go 1.25+** (for building from source)
- **Kubernetes 1.29+** cluster with a valid kubeconfig
- **kubectl** installed and configured

---

## Binary Installation

### Build from Source

The recommended way to install k13d is to build from source:

```bash
# Clone the repository
git clone https://github.com/cloudbro-kube-ai/k13d.git
cd k13d

# Build the binary
make build

# Or build with Go directly
go build -o k13d ./cmd/kube-ai-dashboard-cli/main.go

# Verify installation
./k13d --version
```

### Cross-Platform Builds

Build binaries for multiple platforms:

```bash
# Build for all platforms
make build-all

# Build for specific platforms
make build-linux    # Linux (amd64, arm64, arm)
make build-darwin   # macOS (Intel, Apple Silicon)
make build-windows  # Windows (amd64)
```

### Install to PATH

```bash
# Move binary to a directory in your PATH
sudo mv k13d /usr/local/bin/

# Or add to your PATH in ~/.bashrc or ~/.zshrc
export PATH="$PATH:/path/to/k13d"
```

---

## Docker

### Quick Start

```bash
docker run -d -p 8080:8080 \
  -v ~/.kube/config:/home/k13d/.kube/config:ro \
  -e K13D_USERNAME=admin \
  -e K13D_PASSWORD=changeme \
  cloudbro-kube-ai/k13d:latest
```

### Docker Compose

For more complex setups, use Docker Compose:

```yaml title="docker-compose.yaml"
version: '3.8'
services:
  k13d:
    image: cloudbro-kube-ai/k13d:latest
    ports:
      - "8080:8080"
    volumes:
      - ~/.kube/config:/home/k13d/.kube/config:ro
    environment:
      - K13D_USERNAME=admin
      - K13D_PASSWORD=changeme
      - K13D_LLM_PROVIDER=openai
      - K13D_LLM_API_KEY=${OPENAI_API_KEY}
```

```bash
docker-compose up -d
```

---

## Kubernetes

### Basic Deployment

```bash
# Deploy to Kubernetes
kubectl apply -f deploy/kubernetes/deployment.yaml

# Port forward to access locally
kubectl port-forward -n k13d svc/k13d 8080:80

# Open in browser
open http://localhost:8080
```

### Helm Chart

```bash
# Add Helm repository
helm repo add k13d https://cloudbro-kube-ai.github.io/k13d

# Install with defaults
helm install k13d k13d/k13d

# Install with custom values
helm install k13d k13d/k13d \
  --set auth.mode=ldap \
  --set llm.provider=ollama
```

---

## Air-Gapped Installation

!!! info "For Offline Environments"
    Use this method when the target environment has no internet access.

### On Connected Machine

```bash
# Create offline bundle
make bundle-offline

# Transfer to air-gapped machine
scp dist/k13d-offline-bundle-*.tar.gz user@airgapped:~/
```

### On Air-Gapped Machine

```bash
# Extract and build
tar -xzvf k13d-offline-bundle-*.tar.gz
cd offline-bundle
make build-offline

# Run with embedded LLM (no API keys needed)
./k13d --embedded-llm -web -port 8080
```

---

## Verifying Installation

```bash
# Check version
./k13d --version

# Run TUI mode (requires valid kubeconfig)
./k13d

# Run web mode
./k13d -web -port 8080

# Check embedded LLM status
./k13d --embedded-llm-status
```

---

## Next Steps

<div class="grid cards" markdown>

-   :material-rocket-launch:{ .lg .middle } __Quick Start__

    ---

    Learn the basics and run your first commands

    [:octicons-arrow-right-24: Quick Start](quick-start.md)

-   :material-cog:{ .lg .middle } __Configuration__

    ---

    Configure LLM providers and customize settings

    [:octicons-arrow-right-24: Configuration](configuration.md)

-   :material-console:{ .lg .middle } __TUI Guide__

    ---

    Master the terminal interface

    [:octicons-arrow-right-24: TUI Dashboard](../user-guide/tui.md)

</div>
