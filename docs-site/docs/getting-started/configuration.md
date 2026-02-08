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
