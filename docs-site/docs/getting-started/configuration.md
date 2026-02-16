# Configuration

k13d uses a YAML configuration file located at `~/.config/k13d/config.yaml`.

## Quick Setup

### Recommended: Upstage Solar

The easiest way to get started with AI features:

1. Get your API key from [Upstage Console](https://console.upstage.ai/api-keys) ($10 free credits)
2. Create the config file:

```yaml title="~/.config/k13d/config.yaml"
llm:
  provider: solar
  model: solar-pro2
  endpoint: https://api.upstage.ai/v1
  api_key: your-upstage-api-key

language: en
beginner_mode: true
enable_audit: true
```

---

## Full Configuration Reference

```yaml title="~/.config/k13d/config.yaml"
# LLM Configuration
llm:
  provider: solar           # solar, openai, ollama, azure, anthropic
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
    token_duration: 1h
    refresh_window: 15m

# Tool Approval Policy
tool_approval:
  auto_approve_read_only: true
  require_approval_for_write: true
  require_approval_for_unknown: true
  block_dangerous: false
  blocked_patterns: []
  approval_timeout_seconds: 60
```

---

## LLM Providers

### Upstage Solar (Recommended)

Best balance of quality, speed, and cost. Excellent tool calling support.

```yaml
llm:
  provider: solar
  model: solar-pro2
  endpoint: https://api.upstage.ai/v1
  api_key: your-key
```

### OpenAI

Best tool support, industry standard.

```yaml
llm:
  provider: openai
  model: gpt-4              # or gpt-3.5-turbo
  api_key: sk-your-key
```

### Azure OpenAI

For enterprise deployments with Azure infrastructure.

```yaml
llm:
  provider: azure
  model: gpt-4
  endpoint: https://your-resource.openai.azure.com
  api_key: your-azure-key
```

### Ollama (Local)

Run models locally for air-gapped environments.

```bash
# Install and run Ollama
curl -fsSL https://ollama.com/install.sh | sh
ollama pull qwen2.5:3b
```

```yaml
llm:
  provider: ollama
  model: qwen2.5:3b
  endpoint: http://localhost:11434/v1
```

**Recommended Ollama Models:**

| Model | Size | Notes |
|-------|------|-------|
| `qwen2.5:3b` | 2GB | Best for low-spec machines |
| `qwen2.5:7b` | 4.5GB | Better reasoning |
| `llama3.2:3b` | 2GB | Good general model |

### Embedded LLM

Zero external dependencies - built-in llama.cpp.

!!! warning "Limited Capability"
    Embedded models have significantly reduced capabilities.
    Use only for testing or when no other option is available.

```bash
# Download model (one-time)
./k13d --download-model

# Run with embedded LLM
./k13d --embedded-llm -web -port 8080
```

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
    provider: solar
    model: solar-pro2
    endpoint: https://api.upstage.ai/v1
    api_key: your-upstage-key
    description: "Upstage Solar Pro2 (Recommended)"

  - name: gpt-4o
    provider: openai
    model: gpt-4o
    api_key: sk-your-openai-key
    description: "OpenAI GPT-4o (Faster)"

  - name: qwen2.5-local
    provider: ollama
    model: qwen2.5:3b
    endpoint: http://localhost:11434
    description: "Local Ollama (Korean, low-spec friendly)"

active_model: solar-pro2
```

#### Profile Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique profile name (used in `:model <name>`) |
| `provider` | Yes | LLM provider: `solar`, `openai`, `ollama`, `anthropic`, `azure` |
| `model` | Yes | Model identifier (e.g., `gpt-4o`, `solar-pro2`, `qwen2.5:3b`) |
| `endpoint` | No | Custom API endpoint (required for Ollama/Azure) |
| `api_key` | No | API key (can also use environment variables) |
| `description` | No | Human-readable description shown in model selector |

#### Switching Models

**TUI:**

- Type `:model` to open the model selector modal (active model marked with `*`)
- Type `:model gpt-4o` to switch directly to a named profile
- The switch takes effect immediately and persists to `config.yaml`

**Web UI:**

- Go to Settings > LLM Settings to change the active model
- Or use the model dropdown in the AI chat panel

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
| `K13D_AUTH_MODE` | `local`, `token`, `ldap` |
| `K13D_NO_AUTH` | Disable authentication |
| `K13D_ADMIN_USER` | Default admin username |
| `K13D_ADMIN_PASSWORD` | Default admin password |

---

## Web UI Configuration

Settings can also be changed via the Web UI:

1. Click ⚙️ **Settings** in the top-right corner
2. Navigate to "LLM Settings" section
3. Configure provider, model, and API key
4. Click **Save Settings**

Changes take effect immediately without restart.
