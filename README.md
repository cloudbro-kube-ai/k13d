# k13d

```
 ██╗  ██╗ ██╗██████╗ ██████╗
 ██║ ██╔╝███║╚════██╗██╔══██╗
 █████╔╝ ╚██║ █████╔╝██║  ██║
 ██╔═██╗  ██║ ╚═══██╗██║  ██║
 ██║  ██╗ ██║██████╔╝██████╔╝
 ╚═╝  ╚═╝ ╚═╝╚═════╝ ╚═════╝

    Kubernetes + AI Dashboard
```

<p align="center">
  <strong>k</strong>ube<strong>a</strong>i<strong>d</strong>ashboard = <strong>k</strong> + 13 letters + <strong>d</strong> = <strong>k13d</strong>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> •
  <a href="#build">Build</a> •
  <a href="#features">Features</a> •
  <a href="#documentation">Documentation</a> •
  <a href="#license">License</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Kubernetes-1.29+-326CE5?style=flat&logo=kubernetes" alt="Kubernetes">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat" alt="License">
  <img src="https://img.shields.io/badge/AI-OpenAI%20%7C%20Ollama%20%7C%20Embedded-orange?style=flat" alt="AI Support">
</p>

---

## What is k13d?

**k13d** is a comprehensive Kubernetes management tool combining:

- 🖥️ **k9s-style TUI** - Fast terminal dashboard with Vim keybindings
- 🤖 **Agentic AI** - AI that *executes* kubectl commands (not just suggests)
- 🌐 **Modern Web UI** - Browser-based dashboard with real-time streaming
- 🔐 **Enterprise Security** - RBAC, JWT, Audit logging, LDAP/SSO

---

## Quick Start

### macOS Gatekeeper

macOS may block the binary with *"Apple could not verify k13d is free of malware"*. To fix:

```bash
# Option 1: Remove quarantine attribute
xattr -d com.apple.quarantine ./k13d
xattr -d com.apple.provenance ./k13d

# Option 2: Allow in System Settings
# Go to System Settings > Privacy & Security > click "Allow Anyway"
```

### TUI Mode (Default)

```bash
./k13d
```

### Web Mode

```bash
./k13d -web -port 8080
```

Open http://localhost:8080

### With Authentication

```bash
./k13d -web -port 8080 --auth-mode local --admin-user admin --admin-password secret
```

### With Ollama (Local LLM)

```bash
# Start Ollama with a model
ollama pull qwen2.5:3b
ollama serve

# Run k13d
./k13d -web -port 8080
# Configure LLM in Settings > AI > Provider: Ollama
```

### With Embedded LLM (No API Key Needed)

```bash
# Download model (one-time)
./k13d --download-model

# Run with embedded LLM
./k13d --embedded-llm -web -port 8080
```

---

## Build

### Prerequisites

- Go 1.25+
- Access to Kubernetes cluster (kubeconfig)

### Build Binary

```bash
# Clone repository
git clone https://github.com/cloudbro-kube-ai/k13d.git
cd k13d

# Build
make build
# or
go build -o k13d ./cmd/kube-ai-dashboard-cli/main.go

# Cross-platform builds
make build-all      # All platforms
make build-linux    # Linux (amd64, arm64)
make build-darwin   # macOS (Intel, Apple Silicon)
make build-windows  # Windows
```

### Docker

```bash
# Quick start
docker run -d -p 8080:8080 \
  -v ~/.kube/config:/home/k13d/.kube/config:ro \
  -e K13D_USERNAME=admin \
  -e K13D_PASSWORD=changeme \
  cloudbro-kube-ai/k13d:latest

# Docker Compose
docker-compose up -d
```

### Kubernetes

```bash
kubectl apply -f deploy/kubernetes/deployment.yaml
kubectl port-forward -n k13d svc/k13d 8080:80
```

---

## Features

| Feature | TUI | Web | Description |
|---------|:---:|:---:|-------------|
| Dashboard | ✅ | ✅ | Real-time resource overview |
| AI Assistant | ✅ | ✅ | Agentic AI with tool execution |
| Resource Browsing | ✅ | ✅ | Pods, Deployments, Services, etc. |
| Topology View | ❌ | ✅ | Interactive resource relationship graph |
| YAML Viewer | ✅ | ✅ | View/Edit manifests |
| Log Viewer | ✅ | ✅ | Real-time streaming with ANSI colors |
| Terminal/Shell | ✅ | ✅ | Pod shell access (xterm.js) |
| Port Forward | ✅ | ✅ | Forward container ports |
| Metrics Charts | ❌ | ✅ | CPU/Memory visualization |
| Reports | ❌ | ✅ | PDF/CSV cluster reports |
| Settings UI | ❌ | ✅ | Graphical configuration |
| Multi-user Auth | ❌ | ✅ | RBAC, JWT, LDAP/SSO |
| Audit Logging | ✅ | ✅ | Track all operations |
| i18n | ✅ | ✅ | English, 한국어, 中文, 日本語 |

---

## Configuration

### Config Files

```
~/.config/k13d/
├── config.yaml       # Main configuration (LLM, language, model profiles)
├── hotkeys.yaml      # Custom hotkey bindings
├── plugins.yaml      # External plugins
├── aliases.yaml      # Resource command aliases (e.g., pp → pods)
└── views.yaml        # Per-resource sort defaults
```

`config.yaml`:

```yaml
llm:
  provider: openai      # openai, ollama, gemini, anthropic, bedrock, solar
  model: gpt-4
  api_key: ${OPENAI_API_KEY}

language: en            # en, ko, zh, ja
beginner_mode: false
enable_audit: true
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `KUBECONFIG` | Kubeconfig path | `~/.kube/config` |
| `OPENAI_API_KEY` | OpenAI API key | - |
| `K13D_AUTH_MODE` | Auth mode | `local` |
| `K13D_LLM_PROVIDER` | LLM provider | `openai` |
| `K13D_LLM_MODEL` | LLM model | `gpt-4` |
| `K13D_LLM_ENDPOINT` | Custom LLM endpoint | - |
| `K13D_PORT` | Web server port | `8080` |

### CLI Flags

| Flag | Description |
|------|-------------|
| `-web` | Enable web mode |
| `-port` | HTTP server port |
| `--auth-mode` | Authentication mode (local, token, ldap) |
| `--no-auth` | Disable authentication (dev only) |
| `--embedded-llm` | Use embedded LLM |
| `--mcp` | Run as MCP server |
| `--debug` | Enable debug logging |
| `--kubeconfig` | Kubeconfig path |
| `--context` | Kubernetes context |

---

## Key Bindings (TUI)

| Key | Action | Key | Action |
|-----|--------|-----|--------|
| `j/k` | Navigate | `d` | Describe |
| `g/G` | Top/Bottom | `y` | View YAML |
| `Enter` | Drill down | `l` | View logs |
| `Esc` | Go back | `s` | Shell |
| `/` | Filter | `S` | Scale |
| `Tab` | AI panel | `R` | Restart |
| `?` | Help | `q` | Quit |

### Management Commands

| Command | Action |
|---------|--------|
| `:alias` | View resource aliases |
| `:model` | Switch AI model profile |
| `:plugins` | View available plugins |

---

## Development

```bash
# Run tests
make test
go test -v -race ./...

# Run linter
make lint
golangci-lint run

# Format code
gofmt -s -w .
go vet ./...

# Build all platforms
make build-all

# Run benchmarks
go build -o k13d-bench ./cmd/bench/
./k13d-bench run --task-dir benchmarks/tasks --llm-provider openai
```

---

## API Reference (Quick)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/health` | GET | Health check |
| `/api/auth/login` | POST | Login |
| `/api/k8s/pods` | GET | List pods |
| `/api/k8s/deployments` | GET | List deployments |
| `/api/chat/stream` | POST | AI chat (SSE) |
| `/api/audit` | GET | Audit logs |
| `/api/reports` | GET | Generate reports |

---

## Documentation

📚 **Full documentation: [https://cloudbro-kube-ai.github.io/k13d](https://cloudbro-kube-ai.github.io/k13d)**

| Topic | Link |
|-------|------|
| Installation | [Getting Started](https://cloudbro-kube-ai.github.io/k13d/getting-started/installation/) |
| Features | [All Features](https://cloudbro-kube-ai.github.io/k13d/features/) |
| Web UI Features | [Web UI Guide](https://cloudbro-kube-ai.github.io/k13d/features/web-ui/) |
| TUI Features | [TUI Guide](https://cloudbro-kube-ai.github.io/k13d/features/tui/) |
| AI Assistant | [AI Guide](https://cloudbro-kube-ai.github.io/k13d/features/ai-assistant/) |
| Configuration | [Full Config](https://cloudbro-kube-ai.github.io/k13d/getting-started/configuration/) |
| Docker | [Docker Guide](https://cloudbro-kube-ai.github.io/k13d/deployment/docker/) |
| Kubernetes | [K8s Guide](https://cloudbro-kube-ai.github.io/k13d/deployment/kubernetes/) |
| MCP Integration | [MCP Guide](https://cloudbro-kube-ai.github.io/k13d/concepts/mcp-integration/) |
| API Reference | [REST API](https://cloudbro-kube-ai.github.io/k13d/reference/api/) |
| CLI Reference | [CLI Options](https://cloudbro-kube-ai.github.io/k13d/reference/cli/) |

### Run Docs Locally

```bash
pip install mkdocs-material mkdocs-minify-plugin
cd docs-site
mkdocs serve
# Open http://127.0.0.1:8000
```

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

```bash
git clone https://github.com/YOUR-USERNAME/k13d.git
cd k13d
make test && make lint
# Create PR
```

---

## Security

See [SECURITY.md](SECURITY.md) for reporting vulnerabilities.

---

## License

MIT License - see [LICENSE](LICENSE).

---

<p align="center">
  <sub>Built with ❤️ for the Kubernetes Community</sub>
</p>

<p align="center">
  <a href="https://github.com/cloudbro-kube-ai/k13d">
    <img src="https://img.shields.io/github/stars/cloudbro-kube-ai/k13d?style=social" alt="GitHub Stars">
  </a>
</p>
