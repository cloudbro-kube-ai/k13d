# CLI Reference

Complete command-line interface reference for k13d.

## Synopsis

```bash
k13d [flags]
k13d [command]
```

## Modes

k13d can run in different modes:

| Mode | Command | Description |
|------|---------|-------------|
| TUI | `k13d` | Terminal dashboard (default) |
| Web | `k13d -web` | Web dashboard |
| MCP Server | `k13d --mcp` | MCP server mode |
| CLI | `k13d <command>` | Direct command execution |

## Global Flags

### Configuration

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/k13d/config.yaml` | Config file path |
| `--kubeconfig` | string | `~/.kube/config` | Kubeconfig file path |
| `--context` | string | current | Kubernetes context |
| `--namespace`, `-n` | string | default | Default namespace |

### Logging

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--debug` | bool | false | Enable debug logging |
| `--log-level` | string | info | Log level (debug, info, warn, error) |
| `--log-file` | string | - | Log file path |

### AI Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--llm-provider` | string | openai | LLM provider |
| `--llm-model` | string | gpt-4 | LLM model name |
| `--llm-endpoint` | string | - | Custom LLM endpoint |
| `--embedded-llm` | bool | false | Use embedded LLM |

## TUI Mode

```bash
k13d [flags]
```

### TUI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--refresh` | duration | 5s | Refresh interval |
| `--no-ai` | bool | false | Disable AI panel |
| `--beginner` | bool | false | Enable beginner mode |
| `--language` | string | en | UI language (en, ko, zh, ja) |

### Examples

```bash
# Default TUI mode
k13d

# With specific context
k13d --context production

# With custom kubeconfig
k13d --kubeconfig ~/.kube/prod-config

# Debug mode
k13d --debug

# With embedded LLM
k13d --embedded-llm
```

## Web Mode

```bash
k13d -web [flags]
```

### Web Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-web` | bool | false | Enable web mode |
| `-port`, `-p` | int | 8080 | HTTP server port |
| `-password` | string | - | Web UI password |
| `-host` | string | 0.0.0.0 | Bind address |
| `-tls` | bool | false | Enable TLS |
| `-cert` | string | - | TLS certificate file |
| `-key` | string | - | TLS key file |

### Examples

```bash
# Start web server
k13d -web

# Custom port
k13d -web -port 3000

# With password protection
k13d -web -password "secret"

# Bind to localhost only
k13d -web -host 127.0.0.1

# With TLS
k13d -web -tls -cert server.crt -key server.key
```

## MCP Server Mode

```bash
k13d --mcp [flags]
```

### MCP Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--mcp` | bool | false | Enable MCP server mode |
| `--mcp-stdio` | bool | true | Use stdio transport |

### Examples

```bash
# Run as MCP server
k13d --mcp

# For Claude Desktop integration
# Add to claude_desktop_config.json:
# {
#   "mcpServers": {
#     "k13d": {
#       "command": "k13d",
#       "args": ["--mcp"]
#     }
#   }
# }
```

## Commands

### report

Generate cluster reports.

```bash
k13d report [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--type` | string | cluster-overview | Report type |
| `--format` | string | markdown | Output format |
| `--output` | string | - | Output file path |
| `--namespace` | string | - | Filter by namespace |
| `--ai-analysis` | bool | true | Include AI analysis |

Report types:
- `cluster-overview`
- `security-audit`
- `optimization`
- `ai-analysis`

```bash
# Generate cluster overview
k13d report --type cluster-overview

# PDF format
k13d report --type security-audit --format pdf --output report.pdf

# Specific namespace
k13d report --type optimization --namespace production
```

### bench

Run AI benchmarks.

```bash
k13d bench [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | bool | false | Run all benchmarks |
| `--category` | string | - | Benchmark category |
| `--models` | string | - | Comma-separated models |
| `--tasks` | string | - | Custom tasks file |
| `--format` | string | table | Output format |
| `--output` | string | - | Output file |

```bash
# Run all benchmarks
k13d bench --all

# Specific category
k13d bench --category troubleshooting

# Compare models
k13d bench --models gpt-4,gpt-3.5,ollama/llama3

# JSON output
k13d bench --format json --output results.json
```

### version

Show version information.

```bash
k13d version
```

Output:
```
k13d version v0.6.3
  Go version: go1.25.0
  Git commit: abc1234
  Built: 2024-01-15T10:30:00Z
  OS/Arch: linux/amd64
```

### help

Show help for commands.

```bash
k13d help [command]
k13d --help
k13d -h
```

## Environment Variables

CLI flags can be set via environment variables:

| Environment Variable | Equivalent Flag |
|---------------------|-----------------|
| `KUBECONFIG` | `--kubeconfig` |
| `K13D_CONFIG` | `--config` |
| `K13D_CONTEXT` | `--context` |
| `K13D_NAMESPACE` | `--namespace` |
| `K13D_DEBUG` | `--debug` |
| `K13D_PORT` | `-port` |
| `K13D_PASSWORD` | `-password` |
| `OPENAI_API_KEY` | - |
| `ANTHROPIC_API_KEY` | - |
| `K13D_LLM_PROVIDER` | `--llm-provider` |
| `K13D_LLM_MODEL` | `--llm-model` |

Example:
```bash
export K13D_DEBUG=true
export K13D_PORT=3000
k13d -web
```

## Configuration Precedence

Configuration is loaded in this order (later overrides earlier):

1. Default values
2. Config file (`~/.config/k13d/config.yaml`)
3. Environment variables
4. Command-line flags

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | Kubernetes connection error |
| 4 | LLM provider error |

## Shell Completion

### Bash

```bash
# Generate completion script
k13d completion bash > /etc/bash_completion.d/k13d

# Or add to .bashrc
source <(k13d completion bash)
```

### Zsh

```bash
# Generate completion script
k13d completion zsh > "${fpath[1]}/_k13d"

# Or add to .zshrc
source <(k13d completion zsh)
```

### Fish

```bash
k13d completion fish > ~/.config/fish/completions/k13d.fish
```

### PowerShell

```powershell
k13d completion powershell | Out-String | Invoke-Expression
```

## Examples

### Development Workflow

```bash
# Start with debug logging
k13d --debug --log-file k13d.log

# Use local Ollama
k13d --llm-provider ollama --llm-endpoint http://localhost:11434

# Test with specific cluster
k13d --context minikube --namespace test
```

### Production Deployment

```bash
# Web server with auth
k13d -web -port 8080 -password "$K13D_PASSWORD"

# With TLS
k13d -web -tls -cert /certs/server.crt -key /certs/server.key

# Custom config
k13d -web --config /etc/k13d/config.yaml
```

### Automation

```bash
# Generate daily reports
k13d report --type cluster-overview \
  --format pdf \
  --output "/reports/cluster-$(date +%Y%m%d).pdf"

# Run benchmarks
k13d bench --all --format json --output benchmark-results.json
```

## Next Steps

- [API Reference](api.md) - REST API
- [Environment Variables](env-vars.md) - Full env var list
- [Configuration](../getting-started/configuration.md) - Config file options
