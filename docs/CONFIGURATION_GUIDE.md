# k13d Configuration Guide

`k13d` is designed to be resilient and works out-of-the-box without any manual configuration. However, you can customize your experience by editing the configuration file.

## Configuration File Structure

```
~/.config/k13d/
├── config.yaml       # Main configuration
├── hotkeys.yaml      # Custom hotkey bindings
├── plugins.yaml      # External plugins
└── skins/
    └── default.yaml  # Theme customization
```

## Configuration File Path

| Platform | Path |
|----------|------|
| macOS | `~/Library/Application Support/k13d/config.yaml` |
| Linux | `~/.config/k13d/config.yaml` |
| Windows | `%APPDATA%\k13d\config.yaml` |

## Main Configuration (config.yaml)

### Core Settings

| Key | Description | Default | Options |
|-----|-------------|---------|---------|
| `language` | UI Language | `en` | `en`, `ko`, `ja`, `zh` |
| `beginner_mode` | Simplified AI explanations | `true` | `true`, `false` |
| `enable_audit` | Audit logging | `true` | `true`, `false` |
| `report_path` | Report output path | `report.md` | Any valid path |
| `log_level` | Logging verbosity | `info` | `debug`, `info`, `warn`, `error` |

### LLM Settings

Configure your AI provider in the `llm` block:

```yaml
llm:
  provider: "openai"    # LLM provider
  model: "gpt-4"        # Model name
  endpoint: ""          # Custom API endpoint (optional)
  api_key: "sk-..."     # API key (or use env var)
```

**Supported Providers:**

| Provider | Value | Environment Variable |
|----------|-------|---------------------|
| OpenAI | `openai` | `OPENAI_API_KEY` |
| Azure OpenAI | `azure` | `AZURE_OPENAI_API_KEY` |
| Anthropic | `anthropic` | `ANTHROPIC_API_KEY` |
| Ollama (local) | `ollama` | - |
| AWS Bedrock | `bedrock` | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` |
| Google Vertex | `vertex` | `GOOGLE_APPLICATION_CREDENTIALS` |

**Recommended Ollama Models for Local Use:**

| Model | Size | Tool Calling | Notes |
|-------|------|:------------:|-------|
| `qwen2.5:3b` | 2GB | ✅ | Best for low-spec machines |
| `qwen2.5:7b` | 4.5GB | ✅ | Better reasoning |
| `llama3.2:3b` | 2GB | ✅ | Good general model |
| `mistral:7b` | 4GB | ✅ | Fast inference |

### Full Example

```yaml
# k13d Configuration
language: en
beginner_mode: true
enable_audit: true
report_path: ~/reports/k13d-report.md
log_level: info

llm:
  provider: openai
  model: gpt-4-turbo
  api_key: ${OPENAI_API_KEY}  # Uses environment variable
```

## Using Environment Variables

API keys can be set via environment variables instead of the config file:

```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# Or for Azure
export AZURE_OPENAI_API_KEY="..."
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"
```

## Web UI CLI Options

When running in web mode, you can configure authentication via CLI flags:

```bash
# Basic web mode
k13d -web -port 8080

# With local authentication
k13d -web -port 8080 --auth-mode local --admin-user admin --admin-password secret

# Token-based auth (Kubernetes Dashboard style)
k13d -web -port 8080 --auth-mode token

# Disable authentication (development only)
k13d -web -port 8080 --no-auth
```

### CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-web` | Enable web UI mode | `false` |
| `-port` | Web server port | `8080` |
| `--auth-mode` | Authentication mode (`token`, `local`, `ldap`) | `local` |
| `--no-auth` | Disable authentication | `false` |
| `--admin-user` | Default admin username | `admin` |
| `--admin-password` | Default admin password | - |
| `--kubeconfig` | Path to kubeconfig file | `~/.kube/config` |
| `--debug` | Enable debug logging | `false` |

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

## In-App Settings

### TUI Mode
Press `s` to open the Settings modal where you can modify:
- Language
- LLM Provider and Model
- Beginner Mode toggle
- Audit logging toggle

### Web UI Mode
Access settings via the Settings button in the header:
- **General Tab**: Language, log level, streaming, auto-refresh
- **LLM Tab**: Provider, model, endpoint, API key, connection test
- **Authentication Tab** (Admin only): LDAP/SSO configuration

Changes are saved automatically to `config.yaml`.

## Resilience Features

- **Zero-Config**: Works without any configuration file
- **Fail-Safe**: Corrupted config falls back to defaults
- **Auto-Create**: Config directory is created on first save
- **Environment Override**: Env vars take precedence over config file

## Troubleshooting

### Config not loading?
1. Check file permissions: `ls -la ~/.config/k13d/`
2. Validate YAML syntax: `cat ~/.config/k13d/config.yaml | python -c "import yaml, sys; yaml.safe_load(sys.stdin)"`
3. Check logs: `~/.config/k13d/k13d.log`

### AI not responding?
1. Verify API key is set
2. Check network connectivity to the provider
3. Try a different model (e.g., `gpt-3.5-turbo` instead of `gpt-4`)

### Reset to defaults
Delete the config file to reset:
```bash
rm ~/.config/k13d/config.yaml
```

---

## Custom Hotkeys (hotkeys.yaml)

Define custom keyboard shortcuts that trigger external commands.

### Example hotkeys.yaml

```yaml
hotkeys:
  stern-logs:
    shortCut: Shift-L
    description: "Stern multi-pod logs"
    scopes: [pods, deployments]
    command: stern
    args: [-n, $NAMESPACE, $NAME]
    dangerous: false

  port-forward-8080:
    shortCut: Ctrl-P
    description: "Port forward to 8080"
    scopes: [pods, services]
    command: kubectl
    args: [port-forward, -n, $NAMESPACE, $NAME, "8080:8080"]

  open-grafana:
    shortCut: Ctrl-G
    description: "Open Grafana dashboard"
    scopes: ["*"]  # All resources
    command: open
    args: ["https://grafana.example.com/d/k8s/$NAME"]
```

### Hotkey Variables

| Variable | Description |
|----------|-------------|
| `$NAMESPACE` | Current resource namespace |
| `$NAME` | Selected resource name |
| `$CONTEXT` | Current Kubernetes context |

### Hotkey Options

| Key | Description | Required |
|-----|-------------|----------|
| `shortCut` | Key combination (e.g., `Shift-L`, `Ctrl-K`) | Yes |
| `description` | Human-readable description | Yes |
| `scopes` | Resource types (`pods`, `deployments`, `*` for all) | Yes |
| `command` | Command to execute | Yes |
| `args` | Command arguments | No |
| `dangerous` | Require confirmation before execution | No |

---

## Plugins (plugins.yaml)

Extend k13d with external tools and commands.

### Example plugins.yaml

```yaml
plugins:
  dive:
    shortCut: Ctrl-I
    description: "Dive into container image layers"
    scopes: [pods]
    command: dive
    args: [$IMAGE]
    background: false
    confirm: false

  debug:
    shortCut: Shift-D
    description: "Debug pod with ephemeral container"
    scopes: [pods]
    command: kubectl
    args: [debug, -n, $NAMESPACE, $NAME, -it, --image=busybox]
    confirm: true

  lens:
    shortCut: Ctrl-O
    description: "Open in Lens"
    scopes: ["*"]
    command: lens
    args: [--context, $CONTEXT]
    background: true
```

### Plugin Variables

| Variable | Description |
|----------|-------------|
| `$NAMESPACE` | Resource namespace |
| `$NAME` | Resource name |
| `$CONTEXT` | Kubernetes context |
| `$IMAGE` | Container image (for pods) |
| `$LABELS.key` | Label value by key |
| `$ANNOTATIONS.key` | Annotation value by key |

### Plugin Options

| Key | Description | Default |
|-----|-------------|---------|
| `shortCut` | Trigger key combination | Required |
| `description` | Plugin description | Required |
| `scopes` | Applicable resource types | Required |
| `command` | Command to execute | Required |
| `args` | Command arguments | `[]` |
| `background` | Run in background | `false` |
| `confirm` | Require confirmation | `false` |

---

## Themes (skins/)

Customize the appearance of k13d with theme files.

### Example skin: skins/dracula.yaml

```yaml
k13d:
  body:
    fgColor: "#f8f8f2"
    bgColor: "#282a36"

  frame:
    borderColor: "#6272a4"
    focusBorderColor: "#bd93f9"
    titleColor: "#f8f8f2"
    focusTitleColor: "#50fa7b"

  views:
    table:
      header:
        fgColor: "#bd93f9"
        bgColor: "#282a36"
        bold: true
      rowOdd:
        fgColor: "#f8f8f2"
        bgColor: "#282a36"
      rowEven:
        fgColor: "#f8f8f2"
        bgColor: "#343746"
      rowSelected:
        fgColor: "#282a36"
        bgColor: "#8be9fd"

    log:
      fgColor: "#f8f8f2"
      bgColor: "#282a36"
      errorColor: "#ff5555"
      warningColor: "#ffb86c"
      infoColor: "#8be9fd"

  dialog:
    fgColor: "#f8f8f2"
    bgColor: "#44475a"
    buttonFgColor: "#f8f8f2"
    buttonBgColor: "#6272a4"
    buttonFocusFgColor: "#282a36"
    buttonFocusBgColor: "#50fa7b"

  statusBar:
    fgColor: "#f8f8f2"
    bgColor: "#6272a4"
    errorColor: "#ff5555"
```

### Color Formats

- **Hex colors**: `"#ff5555"`, `"#282a36"`
- **Named colors**: `"red"`, `"blue"`, `"green"`
- **Empty/Default**: `""` uses terminal default

### Creating a New Theme

1. Create a file in `~/.config/k13d/skins/mytheme.yaml`
2. Copy the structure from the default theme above
3. Modify colors to your preference
4. The theme will be applied on next startup
