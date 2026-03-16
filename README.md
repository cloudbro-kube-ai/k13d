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
  <img src="https://img.shields.io/badge/AI-OpenAI%20·%20Ollama%20·%20Anthropic%20·%20Gemini%20·%20Solar%20·%20Bedrock-orange?style=flat-square" alt="AI">
</p>

<p align="center">
  <a href="https://cloudbro-kube-ai.github.io/k13d"><strong>Documentation</strong></a> ·
  <a href="https://github.com/cloudbro-kube-ai/k13d/releases"><strong>Download</strong></a> ·
  <a href="https://cloudbro-kube-ai.github.io/k13d/features/web-ui/"><strong>Web UI Guide</strong></a> ·
  <a href="https://cloudbro-kube-ai.github.io/k13d/features/tui/"><strong>TUI Guide</strong></a> ·
  <a href="https://cloudbro-kube-ai.github.io/k13d/ko/"><strong>한국어</strong></a>
</p>

---

> [!WARNING]
> **Current support status**
> k13d currently supports and recommends **local single-binary usage** for the **TUI** and **Web UI**.
> **Docker, Docker Compose, Kubernetes, Helm, and other containerized/in-cluster deployment paths are still Beta / in preparation and are not officially supported yet.**
> There is also **no official public Docker image repository** for end users at this time, so the deployment docs should be treated as roadmap/reference material, not a supported install path.

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

#### Web UI mode — browser dashboard (local / desktop)

```bash
./k13d --web --auth-mode local
# Open http://localhost:8080 — Username: admin / Password: printed in terminal
```

**`--auth-mode local`** uses simple username/password authentication stored in memory — ideal for local development and desktop use. No Kubernetes tokens or external auth providers required. Just start and log in.

> If `--admin-user` and `--admin-password` are not specified, the username defaults to `admin` and a **secure random password is generated** and printed to the terminal on startup. You can also set credentials via environment variables: `K13D_USERNAME` and `K13D_PASSWORD`.

#### Both modes — Web UI + TUI simultaneously

```bash
./k13d --web --auth-mode local &   # Web UI in background
./k13d                            # TUI in foreground
```

Run the Web UI as a background process, then launch the TUI in the foreground. Both share the same kubeconfig.

#### Authentication Modes

| Mode | Flag | Status / Use Case |
|------|------|-------------------|
| **Local** | `--auth-mode local` | Supported. Recommended for local desktop Web UI use |
| **Token** | `--auth-mode token` | Preview only. Deployment-oriented path is not officially supported yet |
| **LDAP** | `--auth-mode ldap` | Preview only. Provider wiring is still incomplete |
| **OIDC** | `--auth-mode oidc` | Preview only. Provider wiring is still incomplete |
| **No Auth** | `--no-auth` | Development/testing only (not recommended) |

`--auth-mode ldap` and `--auth-mode oidc` select those login paths, but the current stock binary does not yet expose every provider-specific LDAP/OIDC field as dedicated CLI flags. The Web UI auth settings page currently shows runtime status rather than persisting provider config.

If you're evaluating k13d today, focus on:

- Local **TUI** with your existing kubeconfig
- Local **Web UI** via `./k13d --web --auth-mode local`

#### Flags

| Flag | Description |
|------|-------------|
| `--web` | Start Web UI server (default port 8080) |
| `--tui` | Start TUI mode explicitly (default when `--web` is not specified) |
| `--mcp` | Start MCP server mode (stdio transport) |
| `--port <N>` | Custom port for Web UI |
| `--config <path>` | Use a non-default `config.yaml` path |
| `--auth-mode <mode>` | Auth mode: `local`, `token`, `ldap`, `oidc` |
| `--admin-user <name>` | Admin username for local auth (env: `K13D_USERNAME`) |
| `--admin-password <pw>` | Admin password for local auth (env: `K13D_PASSWORD`) |
| `--no-auth` | Disable auth (dev only) |
| `-n <namespace>` | Start in a specific namespace |
| `-A` | Start with all namespaces |
| `--version` | Show version information |
| `--completion <shell>` | Generate shell completion (bash, zsh, fish) |
| `--storage-info` | Show storage configuration and data locations |
| `--db-path <path>` | Custom SQLite database path |
| `--no-db` | Disable database persistence entirely |

That's it. Your kubeconfig is auto-detected.

#### Config File Locations

k13d stores `config.yaml` under the platform XDG config directory unless you override it with `--config` or `K13D_CONFIG`.

| Platform | Default config path |
|----------|---------------------|
| Linux | `${XDG_CONFIG_HOME:-~/.config}/k13d/config.yaml` |
| macOS | `~/Library/Application Support/k13d/config.yaml` |
| Windows | `%AppData%\\k13d\\config.yaml` |
| Custom | `k13d --config /path/to/config.yaml` or `K13D_CONFIG=/path/to/config.yaml` |

Many examples below use Linux-style `~/.config/k13d/...` paths for brevity. On macOS, the equivalent default directory is `~/Library/Application Support/k13d/`.

When you start Web UI mode, k13d now prints:

- `Config File`
- `Config Path Source`
- `Env Overrides`
- `LLM Settings`

That startup output is the fastest way to confirm which file is actually being used.

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
- **Metrics** — Historical CPU/Memory/Pods/Nodes charts (SQLite-backed, 7-day retention)
- **Helm** — Release management, history, rollback
- **Terminal** — Full xterm.js shell into any pod
- **Logs** — Real-time streaming with ANSI colors, search, download
- **RBAC Viewer** — Subject-to-role relationship map with permission details
- **Network Policy Map** — Ingress/egress rule visualization
- **Event Timeline** — Cluster events grouped by time windows
- **Security Scanning** — Trivy CVE scanner with air-gapped support
- **Resource Templates** — One-click deploy (Nginx, Redis, PostgreSQL, etc.)
- **Notifications** — Slack, Discord, Teams, Email (SMTP) alerts for cluster events
- **Applications View** — App-centric grouping by Helm and K8s labels
- **Validate View** — Cross-resource validation with severity levels
- **5 Themes** — Tokyo Night (default), Production, Staging, Development, Light

### TUI — k9s on steroids

- **Vim navigation** — `j/k`, `g/G`, `/` filter, `:` commands
- **AI panel** — `Tab` to chat, AI executes commands for you
- **Sort** — `Shift+N` name, `Shift+A` age, `Shift+T` status, `:sort` picker
- **Autocomplete** — Dropdown suggestions as you type
- **Aliases** — Custom shortcuts (`pp` -> `pods`) via `aliases.yaml`
- **Plugins** — External tool integration via `plugins.yaml`
- **Hotkeys** — Custom keyboard shortcuts via `hotkeys.yaml`
- **Model switching** — `:model` to switch saved AI model profiles at runtime
- **i18n** — English, Korean, Chinese, Japanese

---

## AI Setup (Optional)

Configure in **Settings > AI** in the Web UI, or via environment:

```bash
# OpenAI
export K13D_LLM_PROVIDER=openai
export K13D_LLM_MODEL=gpt-4o
export K13D_LLM_API_KEY=sk-...
./k13d --web --auth-mode local

# Ollama (local, free, no API key)
ollama pull gpt-oss:20b && ollama serve
./k13d --web --auth-mode local
# Set Provider: "ollama" in Settings > AI
```

For **Ollama**, choose a model that explicitly supports **tools/function calling**. Some Ollama models can connect and generate text, but k13d's AI Assistant will not work correctly unless the model supports tools. `gpt-oss:20b` is the recommended default.

Embedded LLM support has been removed. For local/private inference, use **Ollama** instead.

### Web UI Model Profiles

In **Settings > AI**:

- **Save Settings** updates the active `llm` connection
- **Add Model Profile** writes a saved entry under `models:` in `config.yaml`
- **Use** switches that saved profile into the active `llm` connection and updates `active_model`

The **Add Model Profile** form mirrors the same provider list as the main LLM form and opens prefilled from the current provider/model/endpoint so it is easier to save the current connection as a named profile.

Adding a profile does **not** auto-activate it. After creating it, click **Use** if you want that profile to become the active model.

### Supported AI Providers

| Provider | Models | Notes |
|----------|--------|-------|
| **OpenAI** | GPT-4o, GPT-4, o3-mini | Best tool calling support |
| **Anthropic** | Claude Opus 4, Sonnet 4, Haiku 4.5 | Native Messages API, strong reasoning |
| **Google Gemini** | Gemini 2.5, 2.0 | Multimodal capable |
| **Upstage Solar** | Solar Pro2, Solar Pro | Good balance of quality/cost |
| **Azure OpenAI** | GPT-4, GPT-3.5 | Enterprise Azure deployments |
| **AWS Bedrock** | Claude, Llama, Mistral | AWS-hosted models |
| **Ollama** | GPT-OSS, Qwen, Llama, Mistral | Local, free, no API key, choose a tools-capable model |

The AI assistant can:

- Diagnose pod crashes and suggest fixes
- Execute kubectl commands with your approval
- Scale deployments, restart rollouts
- Analyze YAML, events, and logs in context
- Use MCP tools for extended capabilities

---

## MCP (Model Context Protocol)

k13d supports MCP for extending AI capabilities with external tools. Run k13d as an MCP client (connects to external MCP servers) or as an MCP server (exposes k13d tools to other AI systems).

```bash
# Run as MCP server (stdio transport)
./k13d --mcp
```

Configure MCP servers in your active `config.yaml`:

```yaml
mcp:
  servers:
    - name: kubernetes
      command: npx
      args: ["-y", "@anthropic/mcp-server-kubernetes"]
    - name: thinking
      command: npx
      args: ["-y", "@modelcontextprotocol/server-sequential-thinking"]
```

See the [MCP Guide](https://cloudbro-kube-ai.github.io/k13d/concepts/mcp-integration/) for details.

---

## CLI Reference

```bash
./k13d                                    # TUI mode
./k13d --web --auth-mode local           # Web UI — local auth (desktop use)
./k13d --web --auth-mode token           # Web UI — token auth path (preview only)
./k13d --web --port 3000                 # Custom port
./k13d --web --no-auth                   # No auth (dev only)
./k13d --mcp                             # MCP server mode
./k13d -n kube-system                    # Start in specific namespace
./k13d -A                                # Start with all namespaces
./k13d --version                         # Show version
./k13d --completion bash                 # Generate shell completion
./k13d --storage-info                    # Show storage configuration
```

### Shell Completion

```bash
# Bash
source <(./k13d --completion bash)

# Zsh
source <(./k13d --completion zsh)

# Fish
./k13d --completion fish | source
```

---

## Deployment Status

Docker, Docker Compose, Kubernetes, and Helm packaging are still **Beta / in preparation**.

- No official public Docker image repository is available yet
- The deployment files in this repository are roadmap/reference material for upcoming work
- The supported experience today is the local **TUI** and local **Web UI**

---

## Build from Source

```bash
git clone https://github.com/cloudbro-kube-ai/k13d.git && cd k13d
make build
```

---

## Configuration

k13d uses `<XDG config home>/k13d/config.yaml` for configuration by default:

| Platform | Default config directory |
|----------|--------------------------|
| Linux | `${XDG_CONFIG_HOME:-~/.config}/k13d/` |
| macOS | `~/Library/Application Support/k13d/` |
| Windows | `%AppData%\\k13d\\` |

| File | Purpose |
|------|---------|
| `config.yaml` | Main config (LLM provider, language, model profiles, storage) |
| `hotkeys.yaml` | Custom hotkey bindings |
| `plugins.yaml` | External plugin definitions |
| `aliases.yaml` | Resource command aliases |
| `views.yaml` | Per-resource view settings (sort defaults) |

See the [Configuration Guide](https://cloudbro-kube-ai.github.io/k13d/getting-started/configuration/) for full reference.

---

## Documentation

**Full documentation: [https://cloudbro-kube-ai.github.io/k13d](https://cloudbro-kube-ai.github.io/k13d)**

- [Installation Guide](https://cloudbro-kube-ai.github.io/k13d/getting-started/installation/)
- [Web UI Features](https://cloudbro-kube-ai.github.io/k13d/features/web-ui/)
- [TUI Features](https://cloudbro-kube-ai.github.io/k13d/features/tui/)
- [AI Assistant](https://cloudbro-kube-ai.github.io/k13d/features/ai-assistant/)
- [Configuration](https://cloudbro-kube-ai.github.io/k13d/getting-started/configuration/)
- [MCP Integration](https://cloudbro-kube-ai.github.io/k13d/concepts/mcp-integration/)
- [Docker Deployment (Beta / not officially supported)](https://cloudbro-kube-ai.github.io/k13d/deployment/docker/)
- [Kubernetes Deployment (Beta / not officially supported)](https://cloudbro-kube-ai.github.io/k13d/deployment/kubernetes/)
- [한국어 가이드](https://cloudbro-kube-ai.github.io/k13d/ko/)

---

## License

MIT License - see [LICENSE](LICENSE).

<p align="center">
  <a href="https://github.com/cloudbro-kube-ai/k13d">
    <img src="https://img.shields.io/github/stars/cloudbro-kube-ai/k13d?style=social" alt="GitHub Stars">
  </a>
</p>
