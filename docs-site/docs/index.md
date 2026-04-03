---
hide:
  - navigation
---

# k13d

<div align="center" markdown>
<pre>
 ██╗  ██╗ ██╗██████╗ ██████╗
 ██║ ██╔╝███║╚════██╗██╔══██╗
 █████╔╝ ╚██║ █████╔╝██║  ██║
 ██╔═██╗  ██║ ╚═══██╗██║  ██║
 ██║  ██╗ ██║██████╔╝██████╔╝
 ╚═╝  ╚═╝ ╚═╝╚═════╝ ╚═════╝
</pre>

**k**ube**a**i**d**ashboard = **k** + 13 letters + **d** = **k13d**

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.29+-326CE5?style=flat&logo=kubernetes)](https://kubernetes.io)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat)](https://github.com/cloudbro-kube-ai/k13d/blob/main/LICENSE)
[![AI Support](https://img.shields.io/badge/AI-OpenAI%20%7C%20Anthropic%20%7C%20Gemini%20%7C%20Ollama-orange?style=flat)](getting-started/configuration.md#llm-providers)

</div>

---

## What is k13d?

**k13d** is a comprehensive Kubernetes management tool that combines:

- :desktop_computer: **k9s-style TUI** - Fast terminal dashboard with Vim keybindings
- :robot: **kubectl-ai Intelligence** - Agentic AI that *actually executes* kubectl commands
- :globe_with_meridians: **Modern Web UI** - Browser-based dashboard with real-time streaming

It bridges the gap between traditional cluster management and natural language AI, helping you manage, debug, and understand your Kubernetes cluster with unprecedented ease.

<div class="grid cards" markdown>

-   :material-clock-fast:{ .lg .middle } __Quick Start__

    ---

    Get up and running in minutes

    [:octicons-arrow-right-24: Quick Start](getting-started/quick-start.md)

-   :material-kubernetes:{ .lg .middle } __TUI Dashboard__

    ---

    k9s-style terminal interface with Vim bindings

    [:octicons-arrow-right-24: TUI Guide](user-guide/tui.md)

-   :material-robot:{ .lg .middle } __AI Assistant__

    ---

    Natural language Kubernetes management

    [:octicons-arrow-right-24: AI Features](concepts/ai-assistant.md)

-   :material-shield-check:{ .lg .middle } __Enterprise Security__

    ---

    RBAC, audit logging, access requests

    [:octicons-arrow-right-24: Security](concepts/security.md)

</div>

---

## Key Features

### :desktop_computer: TUI Dashboard

| Feature | Description |
|---------|-------------|
| **k9s Parity** | Vim-style navigation (`h/j/k/l`), quick switching (`:pods`, `:svc`) |
| **Deep Resource Support** | Pods, Deployments, Services, Nodes, Events, ConfigMaps, Secrets, RBAC... |
| **Interactive Operations** | Scale, Restart, Port-Forward, Delete with confirmation |
| **Real-time Updates** | Live resource watching with instant refresh |
| **Smart Autocomplete** | Dropdown suggestions with custom alias support |
| **Plugin System** | Extend TUI with external tools via `plugins.yaml` |
| **Model Switching** | Switch AI profiles on the fly with `:model` command |
| **Configurable Aliases** | Custom resource shortcuts via `aliases.yaml` |

### :globe_with_meridians: Web Dashboard

| Feature | Description |
|---------|-------------|
| **Modern Interface** | Responsive design with resizable panels |
| **SSE Streaming Chat** | Real-time AI responses with live cursor |
| **Pod Terminal** | Interactive xterm.js shell in browser |
| **Log Viewer** | Real-time logs with ANSI color support |
| **Topology Tree** | Hierarchical resource ownership visualization |
| **Applications** | App-centric view by `app.kubernetes.io/name` labels |
| **Validate** | Cross-resource validation with severity levels |
| **Healing** | Auto-remediation rules with event history |
| **Helm Manager** | Release management, rollback, uninstall |
| **Metrics Dashboard** | Cluster health cards with CPU/Memory bars |
| **5 Color Themes** | Tokyo Night, Production, Staging, Dev, Light |
| **Trivy Scanner** | CVE vulnerability scanning with auto-download |
| **Multi-Cluster** | Context switcher for multiple kubeconfig clusters |
| **RBAC Viewer** | Visual subject→role relationship viewer |
| **Net Policy Map** | Network policy ingress/egress visualization |
| **Event Timeline** | Events grouped by time with warning stats |
| **GitOps** | ArgoCD/Flux application sync status |
| **Templates** | One-click deploy for common K8s patterns |
| **Backups (Velero)** | Backup and schedule management |
| **Resource Diff** | Side-by-side YAML diff |
| **Notifications** | Slack/Discord/Teams webhook alerts |
| **AI Troubleshoot** | One-click AI cluster diagnosis |
| **kubectl Plugin** | Install as `kubectl k13d` |

### :robot: Agentic AI Assistant

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
└─────────────────────────────────────────────────────────────┘
```

| Feature | Description |
|---------|-------------|
| **Tool Execution** | AI *directly runs* kubectl/bash commands (not just suggests) |
| **MCP Integration** | Extensible tools via Model Context Protocol |
| **Safety First** | Dangerous commands require explicit approval |
| **Deep Context** | AI receives YAML + Events + Logs for analysis |

---

## Quick Install

!!! warning "Current support status"
    k13d currently recommends **local single-binary usage** for both the **TUI** and **Web UI**.
    Docker, Docker Compose, Kubernetes, Helm, and other in-cluster deployment paths are still **Beta / in preparation** and are not officially supported yet.

=== "1. Download"

    Download the matching asset from [Release v1.0.1](https://github.com/cloudbro-kube-ai/k13d/releases/tag/v1.0.1).
    You can also browse every available asset on the full [GitHub Releases page](https://github.com/cloudbro-kube-ai/k13d/releases).

    - macOS Apple Silicon: `k13d_v1.0.1_darwin_arm64.tar.gz`
    - macOS Intel: `k13d_v1.0.1_darwin_amd64.tar.gz`
    - Linux amd64: `k13d_v1.0.1_linux_amd64.tar.gz`
    - Linux arm64: `k13d_v1.0.1_linux_arm64.tar.gz`
    - Windows amd64: `k13d_v1.0.1_windows_amd64.zip`

=== "2. Run Web UI"

    ```bash
    ./k13d --web --port 9090 --auth-mode local
    ```

    TUI default:

    ```bash
    ./k13d
    ```

=== "Need your exact OS / CPU?"

    ```text
    See Quick Start for macOS Intel, macOS Apple Silicon, Linux amd64, Linux arm64,
    Windows, plus the required macOS xattr commands.
    ```

---

## Supported LLM Providers

| Provider | Tool Calling | Best For |
|----------|:------------:|----------|
| **Upstage Solar** | :white_check_mark: | **Recommended** - Best balance of quality & cost |
| **OpenAI** | :white_check_mark: | Production use, best tool support |
| **Anthropic** | :white_check_mark: | Claude models, strong reasoning |
| **Google Gemini** | :white_check_mark: | Multimodal, fast responses |
| **Azure OpenAI** | :white_check_mark: | Enterprise deployments |
| **AWS Bedrock** | :white_check_mark: | AWS-hosted models |
| **Ollama** | :white_check_mark: | Recommended local/private inference with a tools-capable model such as `gpt-oss:20b` |

[:octicons-arrow-right-24: Full LLM Configuration](ai-llm/providers.md)

---

## License

k13d is released under the [MIT License](https://github.com/cloudbro-kube-ai/k13d/blob/main/LICENSE).

<p align="center" markdown>
  Built with :heart: for the Kubernetes Community
</p>
