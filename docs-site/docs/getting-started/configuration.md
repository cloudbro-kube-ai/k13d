# Configuration

k13d stores configuration under its config directory. The main file is `config.yaml`.

## Where `config.yaml` Lives

By default, k13d reads:

| Platform | Default path |
|----------|--------------|
| Linux | `${XDG_CONFIG_HOME:-~/.config}/k13d/config.yaml` |
| macOS | `~/.config/k13d/config.yaml` |
| Windows | `%AppData%\\k13d\\config.yaml` |

You can override the path with either:

- `k13d --config /path/to/config.yaml`
- `K13D_CONFIG=/path/to/config.yaml`

On macOS, older installs may still have `~/Library/Application Support/k13d/config.yaml`. Current builds copy that legacy file to `~/.config/k13d/config.yaml` the first time they load it.

## Resolution Order

k13d resolves the active config file in this order:

1. `--config /path/to/config.yaml`
2. `K13D_CONFIG=/path/to/config.yaml`
3. `XDG_CONFIG_HOME=/custom/config-home` -> `$XDG_CONFIG_HOME/k13d/config.yaml`
4. macOS default `~/.config/k13d/config.yaml`
5. platform XDG/AppData default

Important details:

- The CLI flag `--config` is applied by exporting `K13D_CONFIG` before `LoadConfig()` runs.
- If you pass `--config`, startup logs usually show `Config Path Source: K13D_CONFIG`.
- `XDG_CONFIG_HOME` changes the base config directory, but only when `K13D_CONFIG` is not explicitly set.
- The legacy macOS copy from `~/Library/Application Support/k13d/config.yaml` only happens when you are using the default path and the new file does not already exist.

## What Happens When `config.yaml` Is Missing

If the selected `config.yaml` path does not exist:

- k13d starts with built-in defaults from `NewDefaultConfig()`
- environment overrides such as `K13D_LLM_PROVIDER` are still applied
- the missing file is **not** created just by starting the app
- the file is created later when Web UI or TUI saves settings, or any code path calls `Save()`

This is important for CI and E2E:

- `k13d --config /tmp/missing.yaml --web ...` is expected to boot successfully
- the file should remain absent until the first successful save
- a missing explicit custom path does **not** trigger the default macOS legacy copy

## Config Directory Layout

The main config file is only part of the runtime state. A typical config directory looks like this:

```text
~/.config/k13d/
├── config.yaml
├── aliases.yaml
├── hotkeys.yaml
├── plugins.yaml
├── views.yaml
├── skins/
├── audit.db
└── audit.log
```

| File | Purpose |
|------|---------|
| `config.yaml` | Main runtime configuration |
| `aliases.yaml` | TUI resource aliases |
| `hotkeys.yaml` | TUI key bindings |
| `plugins.yaml` | TUI plugins |
| `views.yaml` | TUI view/sort defaults |
| `skins/` | TUI theme overrides |
| `audit.db` | Default SQLite audit/metrics/session database |
| `audit.log` | Plain-text audit log when enabled |

AI chat sessions are stored under the data directory, usually `<XDG data home>/k13d/sessions`, not next to `config.yaml`.

### How to verify the active file

When you start Web UI mode, the terminal now prints:

- `Config File`
- `Config Path Source`
- `Env Overrides`
- `LLM Settings`

Use that startup output first if the Web UI seems to be reading a different file than expected.

You can also inspect the effective storage paths without launching the full UI:

```bash
k13d --storage-info
```

## Quick Setup

### Recommended: Upstage Solar

The easiest way to get started with AI features:

1. Get your API key from [Upstage Console](https://console.upstage.ai/api-keys) ($10 free credits)
2. Create the config file:

```yaml title="~/.config/k13d/config.yaml"
llm:
  provider: upstage
  model: solar-pro2
  endpoint: https://api.upstage.ai/v1
  api_key: ${UPSTAGE_API_KEY}

language: en
beginner_mode: true
enable_audit: true
```

`config.yaml` supports environment placeholders such as `${UPSTAGE_API_KEY}` and `${OPENAI_API_KEY}`.

When k13d reads the file, it expands `${ENV_VAR}` placeholders before runtime use. If you later save from Web UI or TUI, the rewritten file may contain resolved literal values instead of the original placeholder strings. If you need placeholders to remain untouched, keep a template copy outside the app and avoid in-app saves for those secret-bearing fields.

Recommended pattern:

- keep API keys in environment variables
- keep `config.yaml` focused on provider, model, endpoint, and profile names
- avoid committing a literal `api_key:` value into the repository

## Scope

`config.yaml` currently controls:

- LLM settings and saved model profiles
- MCP servers
- storage and audit persistence
- Prometheus settings
- RBAC, JWT, access requests, and tool approval under `authorization`

`config.yaml` does **not** currently persist LDAP or OIDC provider settings from the Web UI. Those provider settings remain startup-configured in the current build.

For the exact Web UI / TUI save behavior, field ownership, and file write rules, see [Model Settings & Storage](../ai-llm/model-settings-storage.md).

## Save Behavior

When Web UI or TUI saves settings:

1. k13d updates the in-memory config
2. it recreates the runtime AI client when LLM settings changed
3. it creates the parent directory if needed
4. it rewrites the active `config.yaml`
5. it writes the file with mode `0600`

That means:

- Web UI and TUI share the same file
- saving in one interface is immediately visible in the other
- there is no separate SQLite source of truth for active LLM settings in the current build

Typical save paths:

- Web UI `Settings -> AI -> Save Settings` updates `llm` and syncs the active profile if present
- Web UI `Add Model Profile` appends or updates `models[]`
- Web UI `Use` updates `active_model` and copies the selected profile into `llm`
- TUI `Shift+O` saves the current runtime settings
- TUI `:model <name>` switches `active_model` and rewrites `llm`

---

## Full Configuration Reference

```yaml title="~/.config/k13d/config.yaml"
# LLM Configuration
llm:
  provider: upstage         # upstage, openai, litellm, ollama, azopenai, anthropic, gemini, bedrock
  model: solar-pro2         # Model name
  endpoint: ""              # Custom endpoint (optional)
  api_key: ""               # API key
  enable_bash_tool: false   # Opt-in: expose bash to agentic AI
  enable_mcp_tools: false   # Opt-in: expose discovered MCP tools to agentic AI

# Language & UX
language: en                # en, ko, zh, ja
beginner_mode: true         # Simple explanations for complex resources

# Security & Audit
enable_audit: true          # Log all operations to SQLite

# Authorization (Teleport-inspired)
authorization:
  default_tui_role: admin   # admin, user, viewer
  access_request_ttl: 30m   # Just-in-time access duration
  require_approval_for:
    - dangerous             # Dangerous commands need approval
  impersonation:
    enabled: false          # K8s impersonation headers
  jwt:
    secret: ${K13D_JWT_SECRET}
    token_duration: 1h
    refresh_window: 15m
  tool_approval:
    auto_approve_read_only: false
    require_approval_for_write: true
    require_approval_for_unknown: true
    block_dangerous: false
    blocked_patterns: []
    approval_timeout_seconds: 60
```

## Authentication Note

Use the Web server flags to choose the login mode:

```bash
k13d --web --auth-mode local
k13d --web --auth-mode token
```

`--auth-mode local` shows the username/password login form only. The Kubernetes token input is shown when you use `--auth-mode token`.

By default, k13d shows `Decision Required` even for read-only AI tool actions such as `kubectl get pods`. Turn on `authorization.tool_approval.auto_approve_read_only` only if you intentionally want to skip those approval prompts.

Agentic AI tool exposure is kubectl-first by default and follows the kubectl-ai prompt/tool contract for the `kubectl` tool. `bash` and external MCP tools are intentionally opt-in in k13d:

```yaml
llm:
  enable_bash_tool: false
  enable_mcp_tools: false
```

`Required Decision` versus hard block is split like this:

- Read-only kubectl: allowed, but prompts by default unless `auto_approve_read_only: true`
- Write kubectl: allowed, and prompts by default unless `require_approval_for_write: false`
- Dangerous kubectl: prompts by default, or blocks completely if `block_dangerous: true`
- Unknown commands: allowed or prompted based on `require_approval_for_unknown`
- Interactive `kubectl edit`, `kubectl port-forward`, `kubectl attach`, `kubectl exec -it`: always blocked, not approvable
- Bash-wrapped Kubernetes or Helm commands: always blocked, not approvable
- `blocked_patterns`: always blocked, not approvable

`--auth-mode ldap` and `--auth-mode oidc` select those auth paths, but the stock binary does not yet expose every provider-specific LDAP/OIDC field as dedicated CLI flags. The Web UI settings page currently shows runtime auth status and does not persist provider configuration into `config.yaml`.

## GitHub Issue Automation

k13d can receive GitHub `issues` webhooks and run an automated local development workflow. This is useful when you want a new ticket to trigger a development agent, a review pass, and optionally draft PR creation from the same machine that is already serving the Web UI.

The public webhook endpoint is:

```text
POST /api/github/automation/webhook
```

If you expose the Web UI directly on HTTPS, GitHub can call it with a standard webhook URL such as:

```text
https://your-domain.example/api/github/automation/webhook
```

Enable both **Issues** and **Issue comments** in the GitHub webhook. `issues` events start development runs, while `issue_comment` events handle natural-language review and merge requests.

Recommended config:

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
  base_branch: main
  remote: origin
  repo_path: /absolute/path/to/repository
  worktree_root: ~/.cache/k13d/github-automation
  branch_prefix: codex/issue-
  development_command: ./scripts/run-agent-dev.sh
  review_command: ./scripts/run-agent-review.sh
  wait_for_ci: true
  ci_wait_timeout_seconds: 600
  ci_poll_interval_seconds: 10
  auto_deploy_preview: true
  deploy_preview_command: ./scripts/deploy-preview.sh
  preview_url_base: https://fingerscore.net
  preview_path_prefix: /previews
  auto_commit: true
  auto_push: true
  auto_create_pr: true
  allow_issue_merge: true
  merge_method: squash
  pull_request_draft: true
  cleanup_worktrees: false
  max_concurrent_jobs: 1
```

Behavior summary:

- Supported GitHub actions: `opened`, `reopened`, `labeled`
- Default trigger label: `codex:auto`
- Webhook signature: validated with `X-Hub-Signature-256`
- Repository gate: matched against `allowed_repositories`
- Author gate: `require_author_org_member: true` only runs issues opened by members of the repository owner organization
- Reviewer notification: `mention_org_members: true` mentions organization members when a trusted issue is accepted
- Assignment: trusted issues are assigned to the issue author
- Reviewers: organization members are requested as reviewers on the generated or reused PR
- Review language: `review_language: ko` makes built-in issue comments and PR review wrappers Korean
- Codex review: `review_command: ./scripts/run-agent-review.sh` runs `codex exec review` and posts the result as a PR Review
- Workspace isolation: each issue gets one stable git worktree under `worktree_root`
- PR/reporting: one issue maps to one stable branch and one open PR when `personal_access_token` is configured
- Issue review: organization members can comment `k13d 코드리뷰 해줘` on the issue to re-run the configured review command on the linked PR
- Issue merge: `allow_issue_merge: true` lets trusted organization members comment `k13d merge 해줘` on the issue to merge the linked PR and close the issue as completed
- Token safety: GitHub token env vars are not forwarded to agent commands, and captured output is redacted
- CI gate: `wait_for_ci` waits for GitHub check runs on the pushed commit before review/deploy
- Preview deploy: `deploy_preview_command` can expose branch previews through the same k13d domain and post the verification link on the generated PR

### Command Placeholders

`development_command`, `review_command`, and `deploy_preview_command` are plain shell commands. k13d expands the following placeholders before execution:

| Placeholder | Meaning |
|-------------|---------|
| `{issue_number}` | GitHub issue number |
| `{issue_title}` | Issue title |
| `{issue_body}` | Issue body text |
| `{issue_url}` | Issue URL |
| `{issue_author}` | Issue author login |
| `{repository}` | `owner/repo` |
| `{repo_path}` | Local repository path |
| `{worktree}` | Issue-specific git worktree path |
| `{branch}` | Generated branch name |
| `{base_branch}` | Base branch used for the worktree |
| `{review_language}` | Preferred review language, default `ko` |
| `{preview_slug}` | URL-safe preview slug, available to `deploy_preview_command` |
| `{preview_path}` | Preview path such as `/previews/codex-issue-123/` |
| `{preview_url}` | Full preview URL when `preview_url_base` is configured |

k13d also exports the same values as `K13D_GHA_*` environment variables for scripts that prefer environment-driven inputs.

Example with a wrapper script:

```yaml
github_automation:
  development_command: ./scripts/run-agent-dev.sh
  review_command: ./scripts/run-agent-review.sh
  deploy_preview_command: ./scripts/deploy-preview.sh
```

Example with inline commands:

```yaml
github_automation:
  development_command: >
    codex exec --cwd "{worktree}" "Read issue #{issue_number}: {issue_title}.
    Implement the requested change, run fmt, test, and build, then stop."
  review_command: >
    codex exec --cwd "{worktree}" "Review the issue #{issue_number} changes.
    Focus on bugs, regressions, security, and missing tests."
```

Example preview deploy script output:

```text
K13D_PREVIEW_TARGET=http://127.0.0.1:18123
```

When `preview_url_base: https://fingerscore.net` and `preview_path_prefix: /previews` are set, k13d publishes that target through:

```text
https://fingerscore.net/previews/{preview_slug}/
```

This path-based preview routing is useful when you only have one public URL. Each branch can run on a different local port, while k13d on `443` reverse-proxies `/previews/<branch>/...` to the correct local target.

Operational notes:

- `development_command` is required when automation is enabled.
- `repo_path` must point at the local clone that will be used as the source repo.
- `personal_access_token` is optional for local execution, but required if you want automatic issue comments, draft PR creation, or PR reviews.
- `require_author_org_member: true` needs either GitHub's `author_association` webhook value to be `OWNER`/`MEMBER` or a token with organization membership read access.
- `mention_org_members: true` also needs a GitHub token that can list organization members; `mention_max_members` limits mention volume.
- k13d assigns the issue author to the accepted issue and requests organization members as PR reviewers after the PR is created or reused.
- Branch names are stable per issue number, for example `codex/issue-123`, so re-running automation for the same issue continues on the same branch and reuses the existing open PR.
- `scripts/run-agent-review.sh` is the provided Codex review wrapper. It runs `codex exec review`, includes uncommitted changes when reviewing pre-commit automation output, and writes the last Codex message to stdout for PR Review creation.
- Review commands are handled from `issue_comment` webhooks when the comment includes `k13d` and a review phrase such as `k13d 코드리뷰 해줘`, `k13d 리뷰해줘`, or `k13d review`.
- `allow_issue_merge` is disabled by default. Enable it only when the token is scoped for the target repository and branch protection still enforces the checks/reviews you want.
- Merge commands are handled from `issue_comment` webhooks. Supported natural-language examples include `k13d merge 해줘`, `k13d main에 merge 해줘`, and `k13d 병합해줘`. After GitHub accepts the merge, k13d closes the issue with `state_reason: completed`.
- k13d removes GitHub token-like env vars from `development_command`, `review_command`, and `deploy_preview_command` environments, then redacts GitHub token patterns from captured logs before storing or commenting them.
- `review_language: ko` is passed to commands as `{review_language}` and `K13D_GHA_REVIEW_LANGUAGE`; include it in your agent prompt if the external review command should also write Korean.
- `wait_for_ci` also requires `personal_access_token` because k13d reads GitHub check runs through the GitHub API.
- `deploy_preview_command` should start or update the branch preview and print `K13D_PREVIEW_TARGET=...` if you want k13d to reverse-proxy it. When deployment succeeds, the issue completion comment and generated PR comment include the human verification link such as `https://fingerscore.net/previews/<branch>/`.
- `cleanup_worktrees: false` is the safer starting point so you can inspect failed jobs.
- Keep `max_concurrent_jobs` low unless your agent/runtime is known to be stable under parallel worktrees.

---

## LLM Providers

### Upstage Solar (Recommended)

Best balance of quality, speed, and cost. Excellent tool calling support.

```yaml
llm:
  provider: upstage
  model: solar-pro2
  endpoint: https://api.upstage.ai/v1
  api_key: ${UPSTAGE_API_KEY}
```

### OpenAI

Best tool support, industry standard.

```yaml
llm:
  provider: openai
  model: gpt-4o             # or gpt-4o-mini
  endpoint: https://api.openai.com/v1
  api_key: ${OPENAI_API_KEY}
```

### Anthropic

Strong reasoning and analysis capabilities.

```yaml
llm:
  provider: anthropic
  model: claude-sonnet-4-6
  endpoint: https://api.anthropic.com
  api_key: ${ANTHROPIC_API_KEY}
```

Anthropic model names can be long and change over time. Use the exact model `id`, not a shortened family name. If you need to verify current IDs, query Anthropic's `GET /v1/models` endpoint. Examples verified on March 17, 2026 include `claude-sonnet-4-6`, `claude-opus-4-6`, `claude-opus-4-5-20251101`, `claude-haiku-4-5-20251001`, and `claude-sonnet-4-5-20250929`.

### Google Gemini

Multimodal capable with large context windows.

```yaml
llm:
  provider: gemini
  model: gemini-2.5-flash    # or gemini-2.5-pro, gemini-2.0-flash
  api_key: ${GOOGLE_API_KEY}
```

### Azure OpenAI

For enterprise deployments with Azure infrastructure.

```yaml
llm:
  provider: azopenai
  model: gpt-4
  endpoint: ${AZURE_OPENAI_ENDPOINT}
  api_key: ${AZURE_OPENAI_API_KEY}
```

### AWS Bedrock

Access Claude, Llama, and Mistral models via AWS.

```yaml
llm:
  provider: bedrock
  model: anthropic.claude-3-sonnet
  region: us-east-1
```

### Ollama (Local)

Run models locally for air-gapped environments.

```bash
# Install and run Ollama
curl -fsSL https://ollama.com/install.sh | sh
ollama pull gpt-oss:20b
```

```yaml
llm:
  provider: ollama
  model: gpt-oss:20b
  endpoint: http://localhost:11434
```

Important: k13d requires an Ollama model with **tools/function calling** support. Text-only Ollama models may connect successfully, but the AI Assistant will not work correctly. Use `gpt-oss:20b` or another Ollama model whose model card explicitly lists tools support.

**Recommended Ollama Models:**

| Model | Size | Notes |
|-------|------|-------|
| `gpt-oss:20b` | 14GB | Recommended default for local AI |
| `qwen2.5:7b` | 4.5GB | Verify tools/function calling support before use |
| `gemma2:2b` | 2GB | Lightweight fallback only if the specific Ollama tag supports tools |

### Embedded LLM Removal

Embedded LLM support has been removed due to poor quality and maintenance cost.

- For local/private inference, use **Ollama**
- If an old config still says `provider: embedded`, change it to `provider: ollama`

---

## Configuration Files

k13d uses multiple configuration files in the platform config directory:

| Platform | Directory |
|----------|-----------|
| Linux | `${XDG_CONFIG_HOME:-~/.config}/k13d/` |
| macOS | `~/.config/k13d/` |
| Windows | `%AppData%\\k13d\\` |

| File | Purpose |
|------|---------|
| `config.yaml` | Main configuration (LLM, language, model profiles) |
| `hotkeys.yaml` | Custom hotkey bindings |
| `plugins.yaml` | External plugin definitions |
| `aliases.yaml` | Resource command aliases |
| `views.yaml` | Per-resource view settings (sort defaults) |

### Custom Hotkeys (hotkeys.yaml)

Define custom keyboard shortcuts that execute external commands, similar to plugins but designed for quick operations on resources.

#### Hotkey Structure

```yaml title="~/.config/k13d/hotkeys.yaml"
hotkeys:
  hotkey-name:
    shortCut: "Shift-L"          # Required - key combination
    description: "Description"   # Required - shown in help
    scopes: [pods]               # Required - target resource types
    command: "stern"             # Required - command to execute
    args: ["-n", "$NAMESPACE"]   # Optional - arguments with variables
    dangerous: false             # Optional - requires confirmation
```

#### Configuration Options

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `shortCut` | string | Yes | Key combination (e.g., `Shift-L`, `Ctrl-P`, `Alt-K`) |
| `description` | string | Yes | Human-readable description |
| `scopes` | list | Yes | Resource types: `[pods]`, `[pods, services]`, or `["*"]` |
| `command` | string | Yes | CLI command to execute (must be in PATH) |
| `args` | list | No | Command arguments with variable expansion |
| `dangerous` | bool | No | `true` = show confirmation before execution (default: `false`) |

#### Available Variables

Same variables as plugins:

| Variable | Description |
|----------|-------------|
| `$NAMESPACE` | Resource namespace |
| `$NAME` | Resource name |
| `$CONTEXT` | Current Kubernetes context |

#### Example Hotkeys

```yaml title="~/.config/k13d/hotkeys.yaml"
hotkeys:
  # Multi-pod log streaming with stern
  stern-logs:
    shortCut: "Shift-L"
    description: "Stern multi-pod logs"
    scopes: [pods, deployments]
    command: stern
    args: ["-n", "$NAMESPACE", "$NAME"]
    dangerous: false

  # Quick port-forward to 8080
  port-forward-8080:
    shortCut: "Ctrl-P"
    description: "Port forward to 8080"
    scopes: [pods, services]
    command: kubectl
    args: [port-forward, -n, "$NAMESPACE", "$NAME", "8080:8080"]
    dangerous: false
```

!!! info "Hotkeys vs Plugins"
    Hotkeys and plugins share a similar structure. The main difference is that hotkeys use `dangerous` for confirmation while plugins use `confirm` and `background`. Use whichever fits your workflow.

### Resource Aliases (aliases.yaml)

Define custom command shortcuts for quick resource navigation. Aliases map short names to full resource type names and integrate with the command bar autocomplete.

```yaml title="~/.config/k13d/aliases.yaml"
aliases:
  pp: pods
  dep: deployments
  svc: services
  sec: secrets
  cm: configmaps
  ds: daemonsets
  sts: statefulsets
  rs: replicasets
  cj: cronjobs
```

#### How Aliases Work

1. Type `:pp` in the command bar → resolves to `:pods` and navigates to Pods view
2. Custom aliases appear in autocomplete suggestions alongside built-in commands
3. Use `:alias` to view all active aliases (built-in + custom)

#### Built-in Aliases

k13d includes many built-in aliases (e.g., `:po` → pods, `:svc` → services). Custom aliases in `aliases.yaml` extend these. If a custom alias conflicts with a built-in, the custom alias takes precedence.

!!! tip "Alias Tips"
    - Keep aliases short (2-3 characters) for quick typing
    - Use `:alias` in TUI to see the full list of available aliases
    - Aliases are case-insensitive

### Plugins (plugins.yaml)

Extend k13d with external CLI commands triggered by keyboard shortcuts. Plugins follow the k9s plugin pattern.

#### Plugin Structure

```yaml title="~/.config/k13d/plugins.yaml"
plugins:
  plugin-name:
    shortCut: "Ctrl-I"          # Required - key combination
    description: "Description"  # Required - shown in :plugins view
    scopes: [pods]              # Required - target resource types
    command: my-command         # Required - command to execute
    args: [$NAME, $NAMESPACE]   # Optional - arguments with variables
    background: false           # Optional - run without suspending TUI
    confirm: false              # Optional - show confirmation before execution
```

#### Configuration Options

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `shortCut` | string | Yes | Key combination (e.g., `Ctrl-I`, `Shift-D`, `Alt-K`) |
| `description` | string | Yes | Displayed in `:plugins` modal |
| `scopes` | list | Yes | Resource types: `[pods]`, `[pods, deployments]`, or `["*"]` for all |
| `command` | string | Yes | CLI command to execute (must be in PATH) |
| `args` | list | No | Command arguments, supports variable expansion |
| `background` | bool | No | `true` = run without suspending TUI (default: `false`) |
| `confirm` | bool | No | `true` = show confirmation modal before execution (default: `false`) |

#### Available Variables

Variables are automatically expanded from the currently selected resource:

| Variable | Description | Example Use |
|----------|-------------|-------------|
| `$NAMESPACE` | Resource namespace | `kubectl -n $NAMESPACE` |
| `$NAME` | Resource name | `kubectl logs $NAME` |
| `$CONTEXT` | Current Kubernetes context | `lens --context $CONTEXT` |
| `$IMAGE` | Container image (pods only) | `dive $IMAGE` |
| `$LABELS.key` | Label value by key | `stern $LABELS.app` |
| `$ANNOTATIONS.key` | Annotation value by key | `$ANNOTATIONS.deployment.kubernetes.io/revision` |

#### Shortcut Format

| Format | Example | Description |
|--------|---------|-------------|
| Single key | `"p"` | Plain key press |
| Ctrl + key | `"Ctrl-D"` | Control modifier |
| Shift + key | `"Shift-L"` | Shift modifier |
| Alt + key | `"Alt-I"` | Alt/Option modifier |
| Combined | `"Ctrl-Shift-D"` | Multiple modifiers |

!!! warning "Shortcut Conflicts"
    Avoid shortcuts that conflict with built-in keybindings (e.g., `Ctrl-D` is Delete, `y` is YAML).
    Use `Ctrl-`, `Shift-`, or `Alt-` combinations to avoid conflicts.

#### Execution Modes

**Foreground** (`background: false`, default):

- Suspends the TUI while the command runs
- Command gets full terminal access (stdin/stdout/stderr)
- TUI restores automatically when the command finishes
- Use for: interactive commands like `dive`, `kubectl exec`, `vim`

**Background** (`background: true`):

- Runs silently without interrupting the TUI
- Use for: commands that open separate windows (Lens, browser) or long-running tasks (port-forward)

#### Example Plugins

```yaml title="~/.config/k13d/plugins.yaml"
plugins:
  # Analyze container image layers with dive
  dive:
    shortCut: "Ctrl-I"
    description: "Dive into container image layers"
    scopes: [pods]
    command: dive
    args: [$IMAGE]

  # Debug pod with ephemeral container
  debug:
    shortCut: "Shift-D"
    description: "Debug pod with ephemeral container"
    scopes: [pods]
    command: kubectl
    args: [debug, -n, $NAMESPACE, $NAME, -it, --image=busybox]
    confirm: true

  # Port-forward in background
  port-forward:
    shortCut: "Shift-F"
    description: "Port-forward to localhost:8080"
    scopes: [pods, services]
    command: kubectl
    args: [port-forward, -n, $NAMESPACE, $NAME, "8080:80"]
    background: true
    confirm: true

  # Multi-pod log streaming with stern
  stern:
    shortCut: "Shift-S"
    description: "Stream logs with stern (by app label)"
    scopes: [deployments]
    command: stern
    args: [-n, $NAMESPACE, $LABELS.app]

  # Open in Lens (all resources)
  lens:
    shortCut: "Ctrl-O"
    description: "Open in Lens"
    scopes: ["*"]
    command: lens
    args: [--context, $CONTEXT]
    background: true
```

!!! tip "Default Plugins"
    If `plugins.yaml` doesn't exist, k13d loads two built-in plugins: **dive** (`Ctrl-D`) and **debug** (`Shift-D`).

### Per-Resource Sort Defaults (views.yaml)

Configure default sort column and direction per resource type. Sort preferences are applied automatically when navigating to each resource.

```yaml title="~/.config/k13d/views.yaml"
views:
  pods:
    sortColumn: AGE
    sortAscending: false       # Newest pods first
  deployments:
    sortColumn: NAME
    sortAscending: true        # Alphabetical A-Z
  services:
    sortColumn: TYPE
    sortAscending: true
  nodes:
    sortColumn: STATUS
    sortAscending: true
```

#### Configuration Options

| Key | Type | Description |
|-----|------|-------------|
| `sortColumn` | string | Column header name (e.g., `NAME`, `AGE`, `STATUS`, `READY`, `RESTARTS`) |
| `sortAscending` | bool | `true` = ascending (A-Z, oldest first), `false` = descending (Z-A, newest first) |

#### Smart Sort Types

k13d automatically detects column types for proper sorting:

| Column Type | Examples | Sort Behavior |
|-------------|----------|---------------|
| Numeric | RESTARTS, COUNT, DESIRED, CURRENT, AVAILABLE | Numeric comparison |
| Ready format | READY (e.g., `1/2`, `3/3`) | Compares X/Y ratio |
| Age format | AGE (e.g., `5d`, `3h`, `10m`) | Converts to seconds for comparison |
| String | NAME, STATUS, TYPE, NAMESPACE | Case-insensitive alphabetical |

### Model Profiles

Configure multiple LLM profiles and switch between them at runtime. This is useful when you want to use different models for different tasks (e.g., a fast model for simple queries and a powerful model for complex analysis).

```yaml title="~/.config/k13d/config.yaml"
models:
  - name: solar-pro2
    provider: upstage
    model: solar-pro2
    endpoint: https://api.upstage.ai/v1
    api_key: ${UPSTAGE_API_KEY}
    description: "Upstage Solar Pro2 (Recommended)"

  - name: gpt-4o
    provider: openai
    model: gpt-4o
    api_key: ${OPENAI_API_KEY}
    description: "OpenAI GPT-4o (Faster)"

  - name: gpt-oss-local
    provider: ollama
    model: gpt-oss:20b
    endpoint: http://localhost:11434
    description: "Local Ollama (recommended default)"

active_model: solar-pro2
```

#### Profile Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique profile name (used in `:model <name>`) |
| `provider` | Yes | LLM provider: `upstage`, `openai`, `litellm`, `ollama`, `anthropic`, `azopenai`, `gemini`, `bedrock` |
| `model` | Yes | Model identifier (e.g., `gpt-4o`, `solar-pro2`, `gpt-oss:20b`) |
| `endpoint` | No | Custom API endpoint (required for Ollama/Azure) |
| `api_key` | No | API key (can also use environment variables) |
| `description` | No | Human-readable description shown in model selector |

#### Switching Models

**TUI:**

- Type `:model` to open the model selector modal (active model marked with `*`)
- Type `:model gpt-4o` to switch directly to a named profile
- The switch takes effect immediately and persists to `config.yaml`
- Saving TUI LLM settings updates the currently active profile

**Web UI:**

- Go to Settings > LLM Settings to add, delete, or switch profiles
- Saving Web UI LLM settings updates the currently active profile

For the full persistence details, including which fields stay global in `llm` and which fields are copied into `models[]`, see [Model Settings & Storage](../ai-llm/model-settings-storage.md).

!!! tip "Cost Optimization"
    Use a lightweight model (e.g., Ollama local) for routine monitoring and switch to a powerful model (e.g., GPT-4o) only when you need deep analysis.

---

## Environment Variables

All configuration can be overridden with environment variables:

| Variable | Description |
|----------|-------------|
| `K13D_LLM_PROVIDER` | LLM provider |
| `K13D_LLM_MODEL` | Model name |
| `K13D_LLM_ENDPOINT` | Custom endpoint |
| `K13D_LLM_API_KEY` | API key |
| `K13D_AUTH_MODE` | `local`, `token`, `ldap`, `oidc` |
| `K13D_NO_AUTH` | Disable authentication |
| `K13D_USERNAME` | Default admin username (local auth mode) |
| `K13D_PASSWORD` | Default admin password (local auth mode) |
| `K13D_PORT` | Web server port (default: 8080) |
| `K13D_CORS_ALLOWED_ORIGINS` | Allowed CORS origins |
| `OPENAI_API_KEY` | OpenAI API key fallback when `api_key` is omitted |
| `ANTHROPIC_API_KEY` | Anthropic API key fallback when `api_key` is omitted |

---

## Web UI Configuration

Settings can also be changed via the Web UI:

1. Click ⚙️ **Settings** in the top-right corner
2. Navigate to "LLM Settings" section
3. Configure provider, model, and API key
4. Click **Save Settings**

Changes take effect immediately without restart.

Important: saving from Web UI writes the current in-memory values back to `config.yaml`. If your file used `${ENV_VAR}` placeholders for API keys, those may be serialized as resolved literal values when you save. See [Model Settings & Storage](../ai-llm/model-settings-storage.md).
