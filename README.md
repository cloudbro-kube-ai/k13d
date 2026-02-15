# k13d

<p align="center">
  <strong>Kubernetes + AI Dashboard</strong><br>
  <sub><strong>k</strong>ube<strong>a</strong>i<strong>d</strong>ashboard = <strong>k</strong> + 13 letters + <strong>d</strong> = <strong>k13d</strong></sub>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Kubernetes-1.29+-326CE5?style=flat&logo=kubernetes" alt="Kubernetes">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat" alt="License">
  <img src="https://img.shields.io/badge/AI-OpenAI%20%7C%20Ollama%20%7C%20Anthropic-orange?style=flat" alt="AI Support">
</p>

---

## Web UI

<p align="center">
  <img src="docs-site/docs/images/webui-full-screen.png" alt="Web UI Dashboard" width="100%">
</p>

<p align="center">
  <img src="docs-site/docs/images/webui-topology-view-all.png" alt="Topology View" width="49%">
  <img src="docs-site/docs/images/web_ui_applications.png" alt="Applications View" width="49%">
</p>

<p align="center">
  <img src="docs-site/docs/images/web_ui_cluster_report_preview.png" alt="Cluster Report" width="49%">
  <img src="docs-site/docs/images/webui-metrics.png" alt="Metrics Dashboard" width="49%">
</p>

<p align="center">
  <img src="docs-site/docs/images/web_ui_event_timeline.png" alt="Event Timeline" width="49%">
  <img src="docs-site/docs/images/web_ui_network_policy_map.png" alt="Network Policy Map" width="49%">
</p>

<p align="center">
  <img src="docs-site/docs/images/webui-assistant-pannel.png" alt="AI Assistant" width="49%">
  <img src="docs-site/docs/images/webui-pod-terminal-access.png" alt="Pod Terminal" width="49%">
</p>

## TUI

<p align="center">
  <img src="docs-site/docs/images/tui-full-screen.png" alt="TUI Dashboard" width="100%">
</p>

<p align="center">
  <img src="docs-site/docs/images/tui_help.png" alt="TUI Help" width="49%">
  <img src="docs-site/docs/images/tui_auto_complete.png" alt="TUI Autocomplete" width="49%">
</p>

---

## Download

Download the latest binary for your platform from **[Releases](https://github.com/cloudbro-kube-ai/k13d/releases)**.

| Platform | File |
|----------|------|
| macOS (Apple Silicon) | `k13d_darwin_arm64.tar.gz` |
| macOS (Intel) | `k13d_darwin_amd64.tar.gz` |
| Linux (amd64) | `k13d_linux_amd64.tar.gz` |
| Linux (arm64) | `k13d_linux_arm64.tar.gz` |
| Windows | `k13d_windows_amd64.zip` |

```bash
# Extract
tar xzf k13d_*.tar.gz
chmod +x k13d
```

> **macOS Gatekeeper**: If macOS blocks the binary, run:
> ```bash
> xattr -d com.apple.quarantine ./k13d
> ```

---

## Quick Start

### Web UI (Recommended)

```bash
./k13d -web -auth-mode local
```

Open http://localhost:8080 — default account: `admin` / `admin`

### TUI

```bash
./k13d
```

### With AI (Optional)

Configure your LLM provider in **Settings > AI** after launching, or via environment:

```bash
# OpenAI
export OPENAI_API_KEY=sk-...
./k13d -web -auth-mode local

# Ollama (local, free)
ollama pull qwen2.5:3b && ollama serve
./k13d -web -auth-mode local
# Then set Provider to "Ollama" in Settings > AI
```

---

## What You Get

### Web UI
- Real-time resource dashboard with namespace/context switching
- AI Assistant that executes kubectl commands with approval workflow
- Topology graph & tree view of resource relationships
- Cluster reports with FinOps cost analysis
- Historical metrics charts (CPU, Memory, Pods, Nodes)
- Helm release management with rollback
- Pod terminal (xterm.js), log viewer, port forwarding
- RBAC viewer, network policy map, event timeline
- Resource templates, validation, notifications (Slack/Discord/Teams)
- 5 themes: Tokyo Night, Production, Staging, Development, Light

### TUI
- k9s-style Vim navigation (`j/k`, `g/G`, `/` filter, `:` commands)
- AI assistant panel (`Tab` to focus)
- Sort resources by any column (`Shift+N` name, `Shift+A` age, `:sort` picker)
- YAML viewer, log streaming, shell access, port forwarding
- Autocomplete, custom aliases, plugin system
- i18n: English, Korean, Chinese, Japanese

---

## CLI Options

| Flag | Description | Example |
|------|-------------|---------|
| `-web` | Launch Web UI | `./k13d -web` |
| `-port` | Web server port (default: 8080) | `./k13d -web -port 3000` |
| `--auth-mode` | Auth mode: `local`, `token` | `./k13d -web -auth-mode local` |
| `--no-auth` | Disable auth (dev only) | `./k13d -web --no-auth` |
| `--admin-user` | Admin username (default: admin) | `--admin-user myuser` |
| `--admin-password` | Admin password (default: admin) | `--admin-password secret` |
| `--kubeconfig` | Kubeconfig path | `--kubeconfig ~/.kube/prod` |
| `--context` | Kubernetes context | `--context prod-cluster` |
| `--debug` | Enable debug logging | `./k13d --debug` |

---

## Docker

```bash
docker run -d -p 8080:8080 \
  -v ~/.kube/config:/home/k13d/.kube/config:ro \
  cloudbro/k13d:latest \
  -web -auth-mode local
```

---

## Build from Source

```bash
git clone https://github.com/cloudbro-kube-ai/k13d.git
cd k13d
make build    # produces ./k13d binary
```

---

## Documentation

Full docs: **[https://cloudbro-kube-ai.github.io/k13d](https://cloudbro-kube-ai.github.io/k13d)**

---

## License

MIT License - see [LICENSE](LICENSE).

<p align="center">
  <a href="https://github.com/cloudbro-kube-ai/k13d">
    <img src="https://img.shields.io/github/stars/cloudbro-kube-ai/k13d?style=social" alt="GitHub Stars">
  </a>
</p>
