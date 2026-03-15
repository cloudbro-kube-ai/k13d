# Configuration

k13d uses a YAML configuration file located at `~/.config/k13d/config.yaml`.

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

## Scope

`config.yaml` currently controls:

- LLM settings and saved model profiles
- MCP servers
- storage and audit persistence
- Prometheus settings
- RBAC, JWT, access requests, and tool approval under `authorization`

`config.yaml` does **not** currently persist LDAP or OIDC provider settings from the Web UI. Those provider settings remain startup-configured in the current build.

For the exact Web UI / TUI save behavior, field ownership, and file write rules, see [Model Settings & Storage](../ai-llm/model-settings-storage.md).

---

## Full Configuration Reference

```yaml title="~/.config/k13d/config.yaml"
# LLM Configuration
llm:
  provider: upstage         # upstage, openai, ollama, azopenai, anthropic, gemini, bedrock
  model: solar-pro2         # Model name
  endpoint: ""              # Custom endpoint (optional)
  api_key: ""               # API key

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
    auto_approve_read_only: true
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

`--auth-mode ldap` and `--auth-mode oidc` select those auth paths, but the stock binary does not yet expose every provider-specific LDAP/OIDC field as dedicated CLI flags. The Web UI settings page currently shows runtime auth status and does not persist provider configuration into `config.yaml`.

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
  model: gpt-4              # or gpt-4o, gpt-3.5-turbo
  api_key: ${OPENAI_API_KEY}
```

### Anthropic

Strong reasoning and analysis capabilities.

```yaml
llm:
  provider: anthropic
  model: claude-3-sonnet     # or claude-3-opus, claude-3-haiku
  api_key: ${ANTHROPIC_API_KEY}
```

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

k13d uses multiple configuration files in `~/.config/k13d/`:

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
| `provider` | Yes | LLM provider: `upstage`, `openai`, `ollama`, `anthropic`, `azopenai`, `gemini`, `bedrock` |
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
