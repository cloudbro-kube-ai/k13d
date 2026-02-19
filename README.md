<h1 align="center">k13d</h1>

<p align="center">
  <strong>The all-in-one Kubernetes dashboard — Terminal & Web UI with AI built in.</strong>
</p>

<p align="center">
  <code><b>k</b>ubeaidashboar<b>d</b></code>  = <code><b>k</b></code>+ <code>13 letters</code> + <code><b>d</b></code> = <code><b>k13d</b></code>
</p>

<p align="center">
  Download a single binary, run one command, and get a full-featured Kubernetes dashboard<br>
  with an AI assistant that actually executes commands for you.
</p>

<p align="center">
  <a href="https://github.com/cloudbro-kube-ai/k13d/releases"><img src="https://img.shields.io/github/v/release/cloudbro-kube-ai/k13d?style=flat-square&color=blue" alt="Release"></a>
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Kubernetes-1.29+-326CE5?style=flat-square&logo=kubernetes" alt="K8s">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/AI-OpenAI%20·%20Ollama%20·%20Anthropic%20·%20Gemini-orange?style=flat-square" alt="AI">
</p>

<p align="center">
  <a href="https://cloudbro-kube-ai.github.io/k13d"><strong>Documentation</strong></a> ·
  <a href="https://github.com/cloudbro-kube-ai/k13d/releases"><strong>Download</strong></a> ·
  <a href="https://cloudbro-kube-ai.github.io/k13d/latest/features/web-ui/"><strong>Web UI Guide</strong></a> ·
  <a href="https://cloudbro-kube-ai.github.io/k13d/latest/features/tui/"><strong>TUI Guide</strong></a> ·
  <a href="https://cloudbro-kube-ai.github.io/k13d/latest/ko/"><strong>한국어</strong></a>
</p>

---

## Web UI

<p align="center">
  <img src="docs-site/docs/images/webui-full-screen.png" alt="Web UI Dashboard" width="100%">
</p>

## TUI

<p align="center">
  <img src="docs-site/docs/images/tui_auto_complete.png" alt="TUI Dashboard" width="100%">
</p>

---

## Get Started in 30 Seconds

**1. Download** from [Releases](https://github.com/cloudbro-kube-ai/k13d/releases) — single binary, no dependencies.

```bash
tar xzf k13d_*.tar.gz && chmod +x k13d
```

> macOS: if blocked, run `xattr -d com.apple.quarantine ./k13d`

**2. Run.**

#### TUI mode — terminal dashboard with Vim navigation + AI panel

```bash
./k13d
```

Opens a full-featured terminal dashboard. Use `j/k` to navigate, `Tab` to open the AI panel, `:` for commands.

#### Web UI mode — browser dashboard

```bash
./k13d -web -auth-mode local
# Open http://localhost:8080 — Default login: admin / admin
```

Starts an HTTP server with the web dashboard. Add `-port 3000` to change the port.

#### Both modes — Web UI + TUI simultaneously

```bash
./k13d -web -auth-mode local &   # Web UI in background
./k13d                            # TUI in foreground
```

Run the Web UI as a background process, then launch the TUI in the foreground. Both share the same kubeconfig.

#### Flags

| Flag | Description |
|------|-------------|
| `-web` | Start Web UI server (default port 8080) |
| `-port <N>` | Custom port for Web UI |
| `-auth-mode local` | Enable local username/password auth |
| `--no-auth` | Disable auth (dev only) |
| `--embedded-llm` | Use built-in LLM (no API key needed) |
| `--kubeconfig <path>` | Custom kubeconfig path |
| `--context <name>` | Use specific cluster context |
| `--debug` | Enable debug logging |

That's it. Your kubeconfig is auto-detected.

---

## Why k13d?

|                          |  k13d   | k9s | Lens | kubectl |
| ------------------------ | :-----: | :-: | :--: | :-----: |
| Terminal UI              | **Yes** | Yes |  -   |    -    |
| Web UI                   | **Yes** |  -  | Yes  |    -    |
| AI Assistant             | **Yes** |  -  |  -   |    -    |
| Single binary, zero deps | **Yes** | Yes |  -   |   Yes   |
| Free & open source       | **Yes** | Yes | Paid |   Yes   |

### Web UI — Everything in the browser

- **Dashboard** — Pods, Deployments, Services, all resources with real-time status
- **AI Assistant** — Ask questions, AI executes kubectl with your approval
- **Topology** — Graph & tree visualization of resource relationships
- **Reports** — Cluster health, security audit, FinOps cost analysis
- **Metrics** — Historical CPU/Memory/Pods/Nodes charts (SQLite-backed)
- **Helm** — Release management, history, rollback
- **Terminal** — Full xterm.js shell into any pod
- **Logs** — Real-time streaming with ANSI colors, search, download
- **RBAC Viewer** — Subject-to-role relationship map
- **Network Policy Map** — Ingress/egress rule visualization
- **Event Timeline** — Cluster events grouped by time windows
- **Resource Templates** — One-click deploy (Nginx, Redis, PostgreSQL, etc.)
- **Notifications** — Slack, Discord, Teams, Email alerts
- **5 Themes** — Light (default), Tokyo Night, Production, Staging, Development

### TUI — k9s on steroids

- **Vim navigation** — `j/k`, `g/G`, `/` filter, `:` commands
- **AI panel** — `Tab` to chat, AI executes commands for you
- **Sort** — `Shift+N` name, `Shift+A` age, `Shift+T` status, `:sort` picker
- **Autocomplete** — Dropdown suggestions as you type
- **Aliases** — Custom shortcuts (`pp` -> `pods`)
- **Plugins** — External tool integration via `plugins.yaml`
- **i18n** — English, Korean, Chinese, Japanese

---

## AI Setup (Optional)

Configure in **Settings > AI** in the Web UI, or via environment:

```bash
# OpenAI
export OPENAI_API_KEY=sk-...
./k13d -web -auth-mode local

# Ollama (local, free, no API key)
ollama pull qwen2.5:3b && ollama serve
./k13d -web -auth-mode local
# Set Provider: "ollama" in Settings > AI
```

The AI assistant can:

- Diagnose pod crashes and suggest fixes
- Execute kubectl commands with your approval
- Scale deployments, restart rollouts
- Analyze YAML, events, and logs in context

---

## CLI Reference

```bash
./k13d                              # TUI mode
./k13d -web                         # Web UI (port 8080)
./k13d -web -port 3000              # Custom port
./k13d -web -auth-mode local        # With authentication
./k13d -web --no-auth               # No auth (dev only)
./k13d --kubeconfig ~/.kube/prod    # Custom kubeconfig
./k13d --context prod-cluster       # Specific context
./k13d --debug                      # Debug logging
```

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
git clone https://github.com/cloudbro-kube-ai/k13d.git && cd k13d
make build
```

---

## Documentation

**Full documentation: [https://cloudbro-kube-ai.github.io/k13d](https://cloudbro-kube-ai.github.io/k13d)**

- [Installation Guide](https://cloudbro-kube-ai.github.io/k13d/latest/getting-started/installation/)
- [Web UI Features](https://cloudbro-kube-ai.github.io/k13d/latest/features/web-ui/)
- [TUI Features](https://cloudbro-kube-ai.github.io/k13d/latest/features/tui/)
- [AI Assistant](https://cloudbro-kube-ai.github.io/k13d/latest/features/ai-assistant/)
- [Configuration](https://cloudbro-kube-ai.github.io/k13d/latest/getting-started/configuration/)
- [Docker Deployment](https://cloudbro-kube-ai.github.io/k13d/latest/deployment/docker/)
- [Kubernetes Deployment](https://cloudbro-kube-ai.github.io/k13d/latest/deployment/kubernetes/)
- [한국어 가이드](https://cloudbro-kube-ai.github.io/k13d/latest/ko/)

---

## License

MIT License - see [LICENSE](LICENSE).

<p align="center">
  <a href="https://github.com/cloudbro-kube-ai/k13d">
    <img src="https://img.shields.io/github/stars/cloudbro-kube-ai/k13d?style=social" alt="GitHub Stars">
  </a>
</p>
