<h1 align="center">k13d</h1>

<p align="center">
  <strong>The all-in-one Kubernetes dashboard for terminal and browser, with AI built in.</strong>
</p>

<p align="center">
  <code><b>k</b>ubeaidashboar<b>d</b></code>  = <code><b>k</b></code>+ <code>13 letters</code> + <code><b>d</b></code> = <code><b>k13d</b></code>
</p>

<p align="center">
  One binary, one command.<br>
  Get a full Kubernetes dashboard in TUI or Web UI, plus an AI assistant that can actually take action.
</p>

<p align="center">
  <a href="https://github.com/cloudbro-kube-ai/k13d/releases"><img src="https://img.shields.io/github/v/release/cloudbro-kube-ai/k13d?style=flat-square&color=blue" alt="Release"></a>
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Kubernetes-1.29+-326CE5?style=flat-square&logo=kubernetes" alt="K8s">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/AI-OpenAI%20·%20Ollama%20·%20Anthropic%20·%20Gemini%20·%20Solar%20·%20Bedrock-orange?style=flat-square" alt="AI">
</p>

<p align="center">
  <a href="https://cloudbro-kube-ai.github.io/k13d/"><strong>Official Docs</strong></a> ·
  <a href="https://github.com/cloudbro-kube-ai/k13d/releases"><strong>Download</strong></a> ·
  <a href="docs-site/docs/user-guide/web.md"><strong>Web UI Guide</strong></a> ·
  <a href="docs-site/docs/user-guide/tui.md"><strong>TUI Guide</strong></a> ·
  <a href="docs-site/docs/ko/index.md"><strong>한국어</strong></a>
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

Opens a full-featured terminal dashboard. Use `j/k` to navigate, `Enter` or `→` to open related resources, `Ctrl+E` to toggle the AI panel, `Alt+F` to temporarily expand the AI panel to full size, and `Enter` on a selected row while the AI panel is open to attach that row as AI context.

#### Web UI mode — browser dashboard (local / desktop)

```bash
./k13d --web --auth-mode local
# Open http://localhost:8080 — Username: admin / Password: printed in terminal
```

The Web UI is responsive on smaller screens and now renders Kubernetes lists with a stale-first dashboard flow, so recently viewed data appears immediately while fresh data is revalidated in the background.

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
- Official docs at [cloudbro-kube-ai.github.io/k13d](https://cloudbro-kube-ai.github.io/k13d/)

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

#### Configuration

k13d stores `config.yaml` in the platform config directory by default and creates it on the first successful save from Web UI, TUI, or any internal `Save()` path.

| Platform | Default config path |
|----------|---------------------|
| Linux | `${XDG_CONFIG_HOME:-~/.config}/k13d/config.yaml` |
| macOS | `~/.config/k13d/config.yaml` |
| Windows | `%AppData%\\k13d\\config.yaml` |

For configuration details, model profiles, storage paths, and example `config.yaml` files, use the official docs:

- [Official Docs (GitHub Pages)](https://cloudbro-kube-ai.github.io/k13d/)
- [Installation Guide](docs-site/docs/getting-started/installation.md)
- [Configuration Guide](docs-site/docs/getting-started/configuration.md)
- [Web UI Guide](docs-site/docs/user-guide/web.md)
- [TUI Guide](docs-site/docs/user-guide/tui.md)
- [MCP Integration](docs-site/docs/concepts/mcp-integration.md)
- [한국어 가이드](docs-site/docs/ko/index.md)

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
- **AI Assistant** — Ask questions, AI executes kubectl with explicit approval by default
  The default agentic toolset is kubectl-first. `bash` and MCP tools are opt-in, and unsupported interactive kubectl flows are hard-blocked instead of sent to approval.
- **GitHub Issue Automation** — Receive GitHub issue webhooks, run an agent-driven dev command in an isolated worktree, optionally create a PR and attach an automated review
- **Topology** — Graph & tree visualization of resource relationships
- **Reports** — Cluster health, node checks, security audit, heuristic FinOps cost analysis
- **Metrics** — Historical CPU/Memory/Pods/Nodes charts (SQLite-backed, 7-day retention)
- **Helm** — Release management, history, rollback
- **Terminal** — Full xterm.js shell into any pod
- **Logs** — Real-time streaming with ANSI colors, search, download
- **Jobs & CronJobs** — Local-time next run, last run, and recent execution history
- **Workload Security Context** — Seccomp, non-root, token mount, host access, and privileged container checks
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
- **Node insights** — `:nodes` shows control-plane/worker role plus CPU, memory, and GPU usage bars
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

# Anthropic
export K13D_LLM_PROVIDER=anthropic
export K13D_LLM_MODEL=claude-sonnet-4-6
export K13D_LLM_ENDPOINT=https://api.anthropic.com
export K13D_LLM_API_KEY=sk-ant-...
./k13d --web --auth-mode local

# Ollama (local, free, no API key)
ollama pull gpt-oss:20b && ollama serve
./k13d --web --auth-mode local
# Set Provider: "ollama" in Settings > AI
```

For **Ollama**, choose a model that explicitly supports **tools/function calling**. Some Ollama models can connect and generate text, but k13d's AI Assistant will not work correctly unless the model supports tools. `gpt-oss:20b` is the recommended default.

Embedded LLM support has been removed. For local/private inference, use **Ollama** instead.

Model profiles, provider-specific examples, and config ownership are documented in:

- [Official Docs](https://cloudbro-kube-ai.github.io/k13d/)
- [Configuration Guide](docs-site/docs/getting-started/configuration.md)
- [Model Settings & Storage](docs-site/docs/ai-llm/model-settings-storage.md)

### Supported AI Providers

| Provider | Models | Notes |
|----------|--------|-------|
| **OpenAI** | GPT-4o, GPT-4, o3-mini | Best tool calling support |
| **Anthropic** | Claude Sonnet 4.6, Opus 4.6, Haiku 4.5 | Native Messages API, strong reasoning |
| **Google Gemini** | Gemini 2.5, 2.0 | Multimodal capable |
| **Upstage Solar** | Solar Pro2, Solar Pro | Good balance of quality/cost |
| **Azure OpenAI** | GPT-4, GPT-3.5 | Enterprise Azure deployments |
| **AWS Bedrock** | Claude, Llama, Mistral | AWS-hosted models |
| **Ollama** | GPT-OSS, Qwen, Llama, Mistral | Local, free, no API key, choose a tools-capable model |

The AI assistant can:

- Diagnose pod crashes and suggest fixes
- Execute kubectl commands with your approval
- Keep read-only `kubectl get` style actions behind `Decision Required` unless you explicitly enable auto-approve
- Prefer `kubectl` over `bash`; shell access is treated as a last resort
- Expose `bash` and MCP tools only when you explicitly enable them in `config.yaml`
- Hard-block unsupported interactive flows such as `kubectl edit`, `kubectl port-forward`, and `kubectl exec -it`

## GitHub Issue Automation

k13d can also act as a lightweight GitHub issue autopilot. When GitHub sends an `issues` webhook to the Web server, k13d can:

- gate execution by label, by default `codex:auto`
- mark accepted work with a `codex:running` issue label while automation is active
- guide humans through a friendly `Codex 개발 요청` Issue Form before automation starts
- create or reuse one stable issue branch and worktree, for example `codex/issue-123`
- run your configured development command
- wait for GitHub checks on the pushed branch
- optionally run a separate review command
- auto-commit, auto-push, and create or reuse exactly one draft PR for that issue branch
- assign the issue author and request organization members as PR reviewers
- deploy a branch preview behind the same domain, for example `/previews/codex-issue-123/`
- post an issue control panel with the preview link and merge checkbox when a GitHub token is configured
- continue the same PR from follow-up issue comments such as `k13d 수정해줘: ...`
- mark follow-up development as active with a 🚀 reaction on the triggering issue comment
- run PR Preview CD on a self-hosted `fingerscore` runner, publishing PRs under `/previews/pr-<number>/`

### Issue-Driven Development Quick Start

Use this flow when you want to drive development from GitHub Issues instead of local commands:

1. Open a `Codex 개발 요청` issue and describe the goal, current context, requested behavior, acceptance criteria, validation, and safety notes.
2. Do not include secrets such as tokens, kubeconfigs, passwords, or API keys.
3. An organization member reviews the issue and adds `codex:auto` only when it is safe to start automation.
4. k13d marks the issue with `codex:running`, creates or reuses `codex/issue-<number>`, and opens or reuses one PR for that issue.
5. Review the PR comment or issue control panel for the CI result and Preview URL.
6. If the Preview needs more work, comment on the issue with `k13d 수정해줘: ...`; k13d reacts to that comment with 🚀 and continues on the same PR.
7. To request another automated review, comment `k13d 코드리뷰 해줘`.
8. After human Preview verification, check the merge box in the issue control panel or comment `k13d merge 해줘` when issue merge is enabled.
9. After a successful issue-requested merge, k13d closes the issue as completed.

This is designed for local or self-hosted operation. If you run k13d directly on a public HTTPS endpoint, GitHub can reach it without extra relay infrastructure:

```text
https://your-domain.example/api/github/automation/webhook
```

Example `config.yaml` section:

```yaml
github_automation:
  enabled: true
  webhook_secret: ${K13D_GITHUB_AUTOMATION_WEBHOOK_SECRET}
  personal_access_token: ${GITHUB_TOKEN}
  allowed_repositories:
    - cloudbro-kube-ai/k13d
  require_author_org_member: true
  mention_org_members: true
  mention_max_members: 20
  review_language: ko
  trigger_label: codex:auto
  repo_path: /absolute/path/to/k13d
  worktree_root: ~/.cache/k13d/github-automation
  development_command: ./scripts/run-agent-dev.sh
  review_command: ./scripts/run-agent-review.sh
  wait_for_ci: true
  allow_issue_merge: true
  merge_method: squash
  auto_deploy_preview: true
  deploy_preview_command: ./scripts/deploy-preview.sh
  preview_url_base: https://fingerscore.net
  preview_path_prefix: /previews
```

The development, review, and preview deployment commands are fully configurable so you can wire in Codex, Claude Code, Gemini CLI, or your own wrapper scripts. The included `scripts/run-agent-review.sh` wrapper runs `codex exec review` and emits a Korean PR review summary. By default, k13d comments and PR reviews in Korean, mentions organization members when a trusted issue is accepted, assigns the issue author, adds `codex:running` while automation is active, requests organization members as PR reviewers, and includes the branch preview link after deployment succeeds. One issue maps to one stable branch and one open PR, so re-labeling, reopening, or commenting `k13d 수정해줘: ...` continues on the same branch and reuses the same PR. Organization members can comment `k13d 코드리뷰 해줘` on the issue to re-run the configured review command. If `allow_issue_merge` is enabled, the final issue control panel includes a preview link plus a Markdown checkbox that acts like a merge button; checking it, or commenting `k13d merge 해줘`, merges the linked PR into the base branch after human preview verification. After a successful issue-requested merge, k13d closes the GitHub issue as completed. k13d never forwards GitHub token env vars to development/review/deploy commands and redacts GitHub token patterns from captured output. A local preview command can start the built branch on a localhost port and print `K13D_PREVIEW_TARGET=http://127.0.0.1:<port>`. k13d then exposes it through the main Web server as `https://fingerscore.net/previews/<branch-slug>/`, which keeps preview access on a single public URL. The PR Preview CD workflow also deploys same-repository PRs on the self-hosted `fingerscore` runner at `https://fingerscore.net/previews/pr-<number>/` and removes the preview when the PR closes. After CI/CD finishes, k13d also posts that verification path on the generated PR so reviewers do not have to jump back to the issue.

For the full config reference, placeholders, environment variables, and webhook flow, see:

- [Configuration Guide](docs-site/docs/getting-started/configuration.md)
- [Web UI Guide](docs-site/docs/user-guide/web.md)
- [GitHub Issue Automation Guide](docs-site/docs/user-guide/github-issue-automation.md)
- [Architecture](docs-site/docs/concepts/architecture.md)
- [Environment Variables](docs-site/docs/reference/env-vars.md)

- Scale deployments, restart rollouts
- Analyze YAML, events, and logs in context
- Use MCP tools for extended capabilities

If you use `config.yaml`, the safest default is:

```yaml
llm:
  enable_bash_tool: false
  enable_mcp_tools: false
```

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

See the [MCP Guide](docs-site/docs/concepts/mcp-integration.md) for details.

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

## Documentation

- Official docs site: [cloudbro-kube-ai.github.io/k13d](https://cloudbro-kube-ai.github.io/k13d/)
- Repository docs: [Installation](docs-site/docs/getting-started/installation.md), [Web UI](docs-site/docs/user-guide/web.md), [TUI](docs-site/docs/user-guide/tui.md), [Configuration](docs-site/docs/getting-started/configuration.md), [MCP](docs-site/docs/concepts/mcp-integration.md), [한국어](docs-site/docs/ko/index.md)

---

## License

MIT License - see [LICENSE](LICENSE).

<p align="center">
  <a href="https://github.com/cloudbro-kube-ai/k13d">
    <img src="https://img.shields.io/github/stars/cloudbro-kube-ai/k13d?style=social" alt="GitHub Stars">
  </a>
</p>
