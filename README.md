# k13d

```
 ██╗  ██╗ ██╗██████╗ ██████╗
 ██║ ██╔╝███║╚════██╗██╔══██╗
 █████╔╝ ╚██║ █████╔╝██║  ██║
 ██╔═██╗  ██║ ╚═══██╗██║  ██║
 ██║  ██╗ ██║██████╔╝██████╔╝
 ╚═╝  ╚═╝ ╚═╝╚═════╝ ╚═════╝
```

<p align="center">
  <strong>k</strong>ube<strong>a</strong>i<strong>d</strong>ashboard = <strong>k</strong> + 13 letters + <strong>d</strong> = <strong>k13d</strong>
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#installation">Installation</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#documentation">Docs</a> •
  <a href="#license">License</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Kubernetes-1.29+-326CE5?style=flat&logo=kubernetes" alt="Kubernetes">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat" alt="License">
  <img src="https://img.shields.io/badge/AI-OpenAI%20%7C%20Ollama%20%7C%20Embedded-orange?style=flat" alt="AI Support">
</p>

---

### Web UI Dashboard
<p align="center">
  <img src="docs/images/webui-dashboard.png" alt="Web UI Dashboard" width="1200">
</p>

---

## What is k13d?

> **k13d** = **k**ube**a**i**d**ashboard
> Just like **k8s** (k + 8 letters + s = kubernetes), **k13d** follows the same numeronym pattern!

**k13d** is a comprehensive Kubernetes management tool that combines:
- 🖥️ **k9s-style TUI** - Fast terminal dashboard with Vim keybindings
- 🤖 **kubectl-ai Intelligence** - Agentic AI that *actually executes* kubectl commands
- 🌐 **Modern Web UI** - Browser-based dashboard with real-time streaming

It bridges the gap between traditional cluster management and natural language AI, helping you manage, debug, and understand your Kubernetes cluster with unprecedented ease.

---

## Features

### 🖥️ TUI Dashboard

| Feature | Description |
|---------|-------------|
| **k9s Parity** | Vim-style navigation (`h/j/k/l`), quick switching (`:pods`, `:svc`) |
| **Deep Resource Support** | Pods, Deployments, Services, Nodes, Events, ConfigMaps, Secrets, RBAC... |
| **Interactive Operations** | Scale, Restart, Port-Forward, Delete with confirmation |
| **Real-time Updates** | Live resource watching with instant refresh |

### 🌐 Web UI Dashboard

| Feature | Description |
|---------|-------------|
| **Modern Interface** | Responsive design with resizable panels |
| **SSE Streaming Chat** | Real-time AI responses with live cursor |
| **Pod Terminal** | Interactive xterm.js shell in browser |
| **Log Viewer** | Real-time logs with ANSI color support |
| **Metrics Charts** | CPU/Memory graphs with Chart.js |
| **Authentication** | Session-based auth with LDAP/SSO support |
| **Audit Logging** | Track all actions in SQLite database |
| **Reports** | LLM-powered cluster analysis (PDF/CSV) |

### 🤖 Agentic AI Assistant

```
┌─────────────────────────────────────────────────────────────┐
│  You: Show me pods with high memory usage in production     │
├─────────────────────────────────────────────────────────────┤
│  AI: I'll check the pods in the production namespace.       │
│                                                             │
│  🔧 Executing: kubectl top pods -n production --sort-by=mem │
│                                                             │
│  Here are the top memory consumers:                         │
│  NAME                    CPU    MEMORY                      │
│  api-server-7d4f8b...    250m   1.2Gi   ⚠️ High            │
│  worker-processor-...    100m   890Mi                       │
│  cache-redis-0           50m    512Mi                       │
└─────────────────────────────────────────────────────────────┘
```

| Feature | Description |
|---------|-------------|
| **Tool Execution** | AI *directly runs* kubectl/bash commands (not just suggests) |
| **MCP Integration** | Model Context Protocol for extensible tools |
| **Safety First** | Dangerous commands require explicit approval |
| **Deep Context** | AI receives YAML + Events + Logs for analysis |
| **Beginner Mode** | Simple explanations for complex resources |

### 🌍 Global & Accessible

- **i18n Support**: English, 한국어, 中文, 日本語
- **Offline Ready**: Works in air-gapped environments with Ollama or Embedded SLLM
- **Embedded AI**: Built-in llama.cpp with Qwen2.5 - no API keys needed
- **Zero Dependencies**: CGO-free SQLite, self-contained binary

---

## Installation

### Quick Install

```bash
# Clone and build
git clone https://github.com/cloudbro-kube-ai/k13d.git
cd k13d
make build

# Or with Go directly
go build -o k13d ./cmd/kube-ai-dashboard-cli/main.go
```

### Cross-Platform Builds

```bash
make build-all      # All platforms
make build-linux    # Linux (amd64, arm64, arm)
make build-darwin   # macOS (Intel, Apple Silicon)
make build-windows  # Windows (amd64)
```

### Docker

```bash
# Quick start
docker run -d -p 8080:8080 \
  -v ~/.kube/config:/home/k13d/.kube/config:ro \
  -e K13D_USERNAME=admin \
  -e K13D_PASSWORD=changeme \
  youngjukim/k13d:latest

# With Docker Compose
docker-compose up -d
```

### Kubernetes

```bash
kubectl apply -f kubernetes/deployment.yaml
kubectl port-forward -n k13d svc/k13d 8080:80
```

<details>
<summary><b>📦 Air-Gapped Installation</b></summary>

```bash
# On connected machine
make bundle-offline
scp dist/k13d-offline-bundle-*.tar.gz user@airgapped:~/

# On air-gapped machine
tar -xzvf k13d-offline-bundle-*.tar.gz
cd offline-bundle && make build-offline
```

</details>

---

## Quick Start

### TUI Mode (Default)

```bash
./k13d
```

**Key Bindings:**

| Key | Action | Key | Action |
|-----|--------|-----|--------|
| `j/k` | Navigate | `d` | Describe |
| `g/G` | Top/Bottom | `y` | View YAML |
| `Enter` | Drill down | `l` | View logs |
| `Esc` | Go back | `s` | Shell |
| `/` | Filter | `S` | Scale |
| `Tab` | AI panel | `R` | Restart |
| `?` | Help | `q` | Quit |

### Web Mode

```bash
# Basic web mode
./k13d -web -port 8080

# With authentication options
./k13d -web -port 8080 --auth-mode local --admin-user admin --admin-password secret

# Disable authentication (development only)
./k13d -web -port 8080 --no-auth

# Token-based authentication (Kubernetes Dashboard style)
./k13d -web -port 8080 --auth-mode token
```

**Authentication Modes:**

| Mode | Description |
|------|-------------|
| `token` | Kubernetes ServiceAccount token (default for in-cluster) |
| `local` | Local username/password authentication |
| `ldap` | LDAP/Active Directory integration |

**Default Credentials:** `admin` / `admin123`

---

## Configuration

### Quick Setup (Recommended: Upstage Solar)

The easiest way to get started is with **Upstage Solar API** - a high-quality, affordable Korean AI model with excellent tool calling support.

**Web UI Setup:**
1. Click ⚙️ **Settings** in the top-right corner
2. In "LLM Settings" section:
   - **Provider**: `Upstage Solar` (recommended)
   - **Endpoint**: `https://api.upstage.ai/v1`
   - **Model**: `solar-pro2`
   - **API Key**: Get your key from [Upstage Console](https://console.upstage.ai/api-keys)
3. Click **Save Settings**

> 💡 **Free Credits**: Sign up at [console.upstage.ai](https://console.upstage.ai) to get **$10 free credits** - enough for extensive testing!

**Config file:** `~/.config/k13d/config.yaml`

```yaml
llm:
  provider: solar            # solar (recommended), openai, ollama, azure
  model: solar-pro2
  endpoint: https://api.upstage.ai/v1
  api_key: your-upstage-api-key

language: en                # en, ko, zh, ja
beginner_mode: true
enable_audit: true
```

### Supported LLM Providers

| Provider | Tool Calling | Best For |
|----------|:------------:|----------|
| **Upstage Solar** | ✅ | **Recommended** - Best balance of quality & cost |
| **OpenAI** | ✅ | Production use, best tool support |
| **Azure OpenAI** | ✅ | Enterprise deployments |
| **Ollama** | ✅ | Air-gapped, local models |
| **AWS Bedrock** | ✅ | AWS enterprise deployments |
| **Anthropic** | ⚠️ | Claude models via adapter |
| **Embedded (llama.cpp)** | ⚠️ | Zero-dependency, minimal resources (not recommended)

### Local LLM with Ollama

For air-gapped environments or local development, k13d works with Ollama:

```bash
# Install and run Ollama
curl -fsSL https://ollama.com/install.sh | sh
ollama pull qwen2.5:3b  # Recommended: lightweight model with tool calling

# Configure k13d to use Ollama
cat > ~/.config/k13d/config.yaml << EOF
llm:
  provider: ollama
  model: qwen2.5:3b
  endpoint: http://localhost:11434/v1
EOF

# Run k13d
./k13d -web -port 8080
```

**Recommended Ollama Models:**

| Model | Size | Tool Calling | Notes |
|-------|------|:------------:|-------|
| `qwen2.5:3b` | 2GB | ✅ | Best for low-spec machines |
| `qwen2.5:7b` | 4.5GB | ✅ | Better reasoning |
| `llama3.2:3b` | 2GB | ✅ | Good general model |
| `mistral:7b` | 4GB | ✅ | Fast inference |

### Embedded SLLM (No External Dependencies)

> ⚠️ **Warning: Not Recommended for Production Use**
>
> The embedded SLLM uses small models (0.5B-3B parameters) that have **significantly limited capabilities** compared to cloud-based AI providers:
> - **Poor reasoning ability** - Often gives incorrect or incomplete answers
> - **Limited tool calling** - May fail to properly invoke kubectl/bash commands
> - **Slow inference** - Noticeably slower than cloud APIs
> - **Context limitations** - May lose track of conversation context
>
> **Recommended alternatives:**
> - **OpenAI API** - Best quality, requires API key
> - **Ollama** - Good local option with larger models (7B+)
> - **Anthropic API** - High quality Claude models
>
> Use embedded SLLM only for **testing, demos, or air-gapped environments** where no other option is available.

k13d includes an **embedded Small Language Model (SLLM)** option using llama.cpp. This allows you to run AI assistant features with **zero external dependencies** - no Ollama, no API keys, no internet required.

**Minimum Requirements:**
- 2 CPU cores
- 4GB RAM
- ~400MB disk space (model + binary)

```bash
# Step 1: Download the model (one-time setup, ~350MB)
./k13d --download-model

# Step 2: Check status
./k13d --embedded-llm-status

# Step 3: Run with embedded LLM
./k13d --embedded-llm -web -port 8080
```

When `--embedded-llm` is enabled:
- LLM settings in Web UI are **automatically locked**
- llama.cpp server runs in the background on port 8081
- No API key or external service required

**Default Model:**

| Model | Size | License | Tool Calling |
|-------|------|---------|:------------:|
| Qwen2.5-0.5B-Instruct (Q4_K_M) | ~350MB | Apache 2.0 | ✅ |

**Using Custom Models:**

You can use any GGUF-format model compatible with llama.cpp:

```bash
# Use a different model file
./k13d --embedded-llm --embedded-llm-model /path/to/your-model.gguf -web

# Use a different port for llama.cpp server
./k13d --embedded-llm --embedded-llm-port 9000 -web
```

**Recommended Models for Embedded Use:**

| Model | Size | RAM | Best For |
|-------|------|-----|----------|
| Qwen2.5-0.5B-Instruct | ~350MB | 2GB | Minimal resource usage (default) |
| Qwen2.5-1.5B-Instruct | ~900MB | 4GB | Better reasoning, still lightweight |
| Qwen2.5-3B-Instruct | ~2GB | 6GB | Best balance of size and capability |
| Llama-3.2-1B-Instruct | ~700MB | 3GB | Good alternative |
| SmolLM2-360M-Instruct | ~200MB | 1.5GB | Ultra-lightweight |

Download GGUF models from [Hugging Face](https://huggingface.co/models?search=gguf).

**CLI Flags Reference:**

| Flag | Description |
|------|-------------|
| `--embedded-llm` | Enable embedded LLM server |
| `--embedded-llm-port` | Server port (default: 8081) |
| `--embedded-llm-model` | Path to custom GGUF model |
| `--embedded-llm-context` | Context size in tokens (0 = auto-detect) |
| `--download-model` | Download the default model |
| `--embedded-llm-status` | Show embedded LLM status |

**Context Size Configuration:**

Context size affects memory usage and input length. k13d auto-detects optimal settings based on the model:

| Model | Max Context | Recommended (4GB RAM) | Min RAM |
|-------|-------------|----------------------|---------|
| Qwen2.5-0.5B | 32K | 2048 | 2GB |
| Qwen2.5-1.5B | 32K | 2048 | 4GB |
| Qwen2.5-3B | 32K | 2048 | 6GB |
| Llama-3.2-1B | 128K | 2048 | 3GB |
| SmolLM2-360M | 8K | 2048 | 2GB |

To manually set context size:
```bash
./k13d --embedded-llm --embedded-llm-context 4096 -web
```

**Air-Gapped / Offline Installation:**

For environments without internet access, you can manually download and install the required files:

1. **Download the model file** (on a connected machine):
   ```bash
   # Default model (~350MB)
   curl -L -o qwen2.5-0.5b-instruct-q4_k_m.gguf \
     "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q4_k_m.gguf"
   ```

2. **Download the llama-server binary**:
   ```bash
   # macOS Apple Silicon
   curl -L -o llama-b4547-bin-macos-arm64.zip \
     "https://github.com/ggerganov/llama.cpp/releases/download/b4547/llama-b4547-bin-macos-arm64.zip"

   # macOS Intel
   curl -L -o llama-b4547-bin-macos-x64.zip \
     "https://github.com/ggerganov/llama.cpp/releases/download/b4547/llama-b4547-bin-macos-x64.zip"

   # Linux x64
   curl -L -o llama-b4547-bin-ubuntu-x64.zip \
     "https://github.com/ggerganov/llama.cpp/releases/download/b4547/llama-b4547-bin-ubuntu-x64.zip"

   # Linux ARM64
   curl -L -o llama-b4547-bin-ubuntu-arm64.zip \
     "https://github.com/ggerganov/llama.cpp/releases/download/b4547/llama-b4547-bin-ubuntu-arm64.zip"
   ```

3. **Transfer files to air-gapped machine** and place them:
   ```bash
   # Create directories
   mkdir -p ~/.local/share/k13d/llm/models
   mkdir -p ~/.local/share/k13d/llm/bin

   # Place model file
   cp qwen2.5-0.5b-instruct-q4_k_m.gguf ~/.local/share/k13d/llm/models/

   # Extract and place llama-server binary
   unzip llama-b4547-bin-*.zip -d /tmp/llama
   cp /tmp/llama/build/bin/llama-server ~/.local/share/k13d/llm/bin/
   chmod +x ~/.local/share/k13d/llm/bin/llama-server
   ```

4. **Verify installation**:
   ```bash
   ./k13d --embedded-llm-status
   ```

5. **Run k13d with embedded LLM**:
   ```bash
   ./k13d --embedded-llm -web -port 8080
   ```

---

## AI Model Benchmark Results

k13d includes a comprehensive benchmark suite based on [k8s-ai-bench](https://github.com/gke-labs/k8s-ai-bench) methodology to evaluate AI model performance on Kubernetes tasks.

### Benchmark Overview

| Category | Description | Tasks |
|----------|-------------|-------|
| **Creation** | Pod, Deployment, Service, ConfigMap creation | 6 |
| **Troubleshooting** | Fix CrashLoopBackOff, ImagePullBackOff, probes | 6 |
| **Operations** | Scale, rolling update, HPA, StatefulSet lifecycle | 8 |
| **Networking** | Service routing, Network Policy, multi-container | 3 |

**Total: 23 tasks** (Easy: 6, Medium: 14, Hard: 3)

### Latest Results (2026-01-23)

#### Accuracy Score Comparison

```
                        k8s-ai-bench Score (23 tasks)

  gemini-3-flash    ████████████████████████████████████████ 100.0%  🥇
  gpt-5-mini        ██████████████████████████████████████▎  95.7%  🥈
  solar-pro2        ██████████████████████████████████████▎  95.7%  🥈
  qwen3:8b          ██████████████████████████████████████▎  95.7%  (local)
  gpt-5             ████████████████████████████████████▌    91.3%
  gemini-3-pro      ████████████████████████████████████▌    91.3%
  deepseek-r1:32b   ████████████████████████████████████▌    91.3%  (local)
  o3-mini           █████████████████████████████████        82.6%
  gemma3:27b        ███████████████████████████████▎         78.3%  (local)
  gemma3:4b         ██████████████████████████               65.2%  (local)
                    └────────────────────────────────────────┘
                    0%                50%                  100%
```

#### Response Time Comparison

```
                        Average Response Time (seconds)

  gemma3:4b         ██▏                                       1.7s  (fastest local)
  gemma3:27b        █████▏                                    4.0s
  gpt-oss:latest    █████▍                                    4.1s
  gemini-3-flash    ███████▊                                  5.9s  (fastest cloud)
  o3-mini           ████████                                  6.0s
  solar-pro2        ███████████▊                              8.9s
  qwen3:8b          ████████████                              9.0s
  deepseek-r1:32b   █████████████████▎                       13.0s
  gemini-3-pro      █████████████████████████▍               19.1s
  gpt-5-mini        █████████████████████████████▌           22.2s
  gpt-5             ██████████████████████████████████████▏  35.4s  (slowest)
                    └────────────────────────────────────────┘
                    0s                 18s                  36s
```

**Cloud Provider Models (GPT-5, Gemini 3, Solar Pro2)**

| Rank | Model | Score | Easy | Medium | Hard | Avg Response |
|:----:|-------|:-----:|:----:|:------:|:----:|:------------:|
| 🥇 | **gemini-3-flash** | **100%** | 6/6 | 14/14 | 3/3 | **5.9s** |
| 🥈 | gpt-5-mini | 95.7% | 6/6 | 13/14 | 3/3 | 22.2s |
| 🥈 | solar-pro2 (high) | 95.7% | 6/6 | 13/14 | 3/3 | 8.9s |
| 4 | gpt-5 | 91.3% | 6/6 | 12/14 | 3/3 | 35.4s |
| 4 | gemini-3-pro | 91.3% | 6/6 | 12/14 | 3/3 | 19.1s |
| 6 | o3-mini | 82.6% | 6/6 | 11/14 | 2/3 | 6.0s |

**Local/Self-hosted Models (Ollama)**

| Rank | Model | Score | Easy | Medium | Hard | Avg Response |
|:----:|-------|:-----:|:----:|:------:|:----:|:------------:|
| 🥇 | qwen3:8b | 95.7% | 6/6 | 13/14 | 3/3 | 9.0s |
| 🥈 | gpt-oss:latest | 91.3% | 5/6 | 13/14 | 3/3 | 4.1s |
| 🥈 | deepseek-r1:32b | 91.3% | 6/6 | 12/14 | 3/3 | 13.0s |
| 4 | gemma3:27b | 78.3% | 3/6 | 12/14 | 3/3 | 4.0s |
| 5 | gemma3:4b | 65.2% | 3/6 | 10/14 | 2/3 | 1.7s |

> **Key Findings:**
> - **gemini-3-flash** achieves **100% accuracy** with fastest response (5.9s) - best overall!
> - **gpt-5-mini** and **solar-pro2 (high)** tie at 95.7% - excellent alternatives
> - **qwen3:8b** leads local models at 95.7% with only 8B parameters
> - Most models struggle with `fix-probes` task (liveness/readiness probe configuration)

### Running Benchmarks

The benchmark suite is based on [k8s-ai-bench](https://github.com/gke-labs/k8s-ai-bench) methodology.

```bash
# Quick benchmark script (API-based evaluation)
go run scripts/cloud_providers_bench.go    # OpenAI, Gemini, Solar
go run scripts/k8s_ai_bench.go             # Local Ollama models

# Full benchmark with actual K8s cluster execution
go build -o k13d-bench ./cmd/bench/

# List available tasks
./k13d-bench list --task-dir benchmarks/tasks

# Run with specific provider
./k13d-bench run \
  --task-dir benchmarks/tasks \
  --llm-provider openai \
  --llm-model gpt-5 \
  --llm-api-key $OPENAI_API_KEY \
  --auto-approve

# Run with Gemini
./k13d-bench run \
  --llm-provider gemini \
  --llm-model gemini-3-flash-preview \
  --llm-api-key $GEMINI_API_KEY

# Analyze results
./k13d-bench analyze --input-dir .build/bench
```

See [benchmarks/README.md](benchmarks/README.md) for detailed documentation.

---

## Architecture

```
k13d/
├── cmd/kube-ai-dashboard-cli/  # Entry point
├── pkg/
│   ├── ai/                     # AI client & tools
│   │   ├── providers/          # OpenAI, Ollama adapters
│   │   ├── tools/              # kubectl, bash tools
│   │   └── sessions/           # Conversation state
│   ├── ui/                     # TUI (tview/tcell)
│   ├── web/                    # Web server & API
│   │   ├── auth.go             # Authentication
│   │   ├── ldap.go             # LDAP/SSO
│   │   └── static/             # Frontend
│   ├── k8s/                    # Kubernetes client
│   ├── db/                     # SQLite audit logs
│   └── i18n/                   # Translations
└── docs/                       # Documentation
```

---

## API Reference

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/health` | GET | Health check |
| `/api/auth/login` | POST | Login |
| `/api/auth/logout` | POST | Logout |
| `/api/k8s/namespaces` | GET | List namespaces |
| `/api/k8s/pods` | GET | List pods |
| `/api/chat/stream` | POST | AI chat (SSE) |
| `/api/audit` | GET | Audit logs |
| `/api/reports` | GET | Generate reports |
| `/api/settings` | GET/POST | User settings |
| `/api/settings/ldap` | GET/POST | LDAP configuration |
| `/api/settings/sso` | GET/POST | SSO/OAuth configuration |

---

## Deployment

### Kubernetes Deployment

Deploy k13d in your Kubernetes cluster using Helm:

```bash
# Add Helm repository
helm repo add k13d https://cloudbro-kube-ai.github.io/k13d

# Install with default settings
helm install k13d k13d/k13d

# Install with custom values
helm install k13d k13d/k13d \
  --set auth.mode=ldap \
  --set ldap.host=ldap.example.com \
  --set llm.provider=ollama \
  --set llm.endpoint=http://ollama:11434/v1
```

Or use the raw manifests:

```bash
kubectl apply -f deploy/helm/k13d/templates/
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `K13D_AUTH_MODE` | Authentication mode | `local` |
| `K13D_NO_AUTH` | Disable authentication | `false` |
| `K13D_ADMIN_USER` | Default admin username | `admin` |
| `K13D_ADMIN_PASSWORD` | Default admin password | - |
| `K13D_LLM_PROVIDER` | LLM provider | `openai` |
| `K13D_LLM_MODEL` | LLM model name | `gpt-4` |
| `K13D_LLM_ENDPOINT` | Custom LLM endpoint | - |
| `K13D_LLM_API_KEY` | LLM API key | - |

---

## Documentation

| Document | Description |
|----------|-------------|
| [Installation Guide](docs/INSTALLATION.md) | All installation methods |
| [Docker Guide](docs/INSTALLATION_DOCKER.md) | Docker and Docker Compose setup |
| [Kubernetes Guide](docs/INSTALLATION_K8S.md) | Kubernetes deployment options |
| [User Guide](docs/USER_GUIDE.md) | Navigation, shortcuts, workflows |
| [Configuration Guide](docs/CONFIGURATION_GUIDE.md) | All config options |
| [MCP Guide](docs/MCP_GUIDE.md) | MCP integration & agentic AI |
| [TUI Architecture](docs/TUI_ARCHITECTURE.md) | Terminal UI internal structure |
| [Architecture](docs/ARCHITECTURE.md) | System architecture overview |
| [Contributing](CONTRIBUTING.md) | How to contribute |
| [Security](SECURITY.md) | Security policy |

---

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) first.

```bash
# Development workflow
make test           # Run tests
make lint           # Run linter
make build          # Build binary
```

---

## Security

We take security seriously. See [SECURITY.md](SECURITY.md) for reporting vulnerabilities.

---

## License

MIT License - see [LICENSE](LICENSE) for details.

---

<p align="center">
  <sub>Built with ❤️ for the Kubernetes Community</sub>
</p>

<p align="center">
  <a href="https://github.com/cloudbro-kube-ai/k13d">
    <img src="https://img.shields.io/github/stars/cloudbro-kube-ai/k13d?style=social" alt="GitHub Stars">
  </a>
</p>
