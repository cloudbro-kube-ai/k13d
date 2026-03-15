# CLI Reference

Complete command-line reference for `k13d`.

## Synopsis

```bash
k13d [flags]
kubectl k13d [flags]
```

`k13d` uses Go's standard flag parser, so both `--web` and `-web` work. This document standardizes on:

- one-letter flags: `-n`, `-A`
- multi-letter flags: `--web`, `--auth-mode`

## Modes

| Mode | Example | Description |
|------|---------|-------------|
| TUI | `k13d` | Terminal dashboard (default) |
| Web | `k13d --web` | Browser dashboard |
| MCP | `k13d --mcp` | MCP server over stdio |

## Flags

### Startup & Scope

| Flag | Default | Description |
|------|---------|-------------|
| `--web` | `false` | Start Web UI mode |
| `--tui` | `false` | Start TUI mode explicitly |
| `--mcp` | `false` | Start MCP server mode |
| `--port` | `8080` | Web server port |
| `--config` | `~/.config/k13d/config.yaml` | Config file path |
| `--namespace`, `-n` | current/default | Initial namespace |
| `--all-namespaces`, `-A` | `false` | Start with all namespaces |

### Authentication

| Flag | Default | Description |
|------|---------|-------------|
| `--auth-mode` | `token` | `token`, `local`, `ldap`, `oidc` |
| `--no-auth` | `false` | Disable authentication |
| `--admin-user` | `admin` in local mode | Default admin username |
| `--admin-password` | random in local mode | Default admin password |

### Storage

| Flag | Default | Description |
|------|---------|-------------|
| `--db-path` | `~/.config/k13d/audit.db` | SQLite database path |
| `--no-db` | `false` | Disable database-backed persistence |
| `--storage-info` | `false` | Print storage paths and exit |

### Embedded LLM

| Flag | Default | Description |
|------|---------|-------------|
| `--embedded-llm` | `false` | Start embedded llama.cpp server |
| `--embedded-llm-port` | `8081` | Embedded LLM port |
| `--embedded-llm-model` | auto | Custom GGUF model path |
| `--embedded-llm-context` | `0` | Context size (`0` = auto) |
| `--download-model` | `false` | Download default embedded model |
| `--embedded-llm-status` | `false` | Print embedded LLM status and exit |

### Utility

| Flag | Default | Description |
|------|---------|-------------|
| `--version` | `false` | Print version information |
| `--completion <shell>` | - | Generate `bash`, `zsh`, or `fish` completion |

## Examples

### TUI

```bash
k13d
k13d -n kube-system
k13d -A
```

### Web UI

```bash
k13d --web
k13d --web --port 3000
k13d --web --auth-mode local
k13d --web --auth-mode local --admin-user admin --admin-password changeme
k13d --web --no-auth
```

### Custom Config

```bash
k13d --config /etc/k13d/config.yaml
k13d --web --config ./config/dev.yaml
```

### Embedded LLM

```bash
k13d --download-model
k13d --embedded-llm --web --auth-mode local
k13d --embedded-llm-status
```

### MCP Server

```bash
k13d --mcp
kubectl k13d --mcp
```

## Environment Variable Equivalents

These environment variables are read directly by the CLI:

| Environment Variable | Equivalent Flag |
|---------------------|-----------------|
| `K13D_WEB` | `--web` |
| `K13D_PORT` | `--port` |
| `K13D_CONFIG` | `--config` |
| `K13D_NAMESPACE` | `--namespace` |
| `K13D_ALL_NAMESPACES` | `--all-namespaces` |
| `K13D_AUTH_MODE` | `--auth-mode` |
| `K13D_NO_AUTH` | `--no-auth` |
| `K13D_USERNAME` | `--admin-user` |
| `K13D_PASSWORD` | `--admin-password` |
| `K13D_DB_PATH` | `--db-path` |
| `K13D_NO_DB` | `--no-db` |
| `K13D_EMBEDDED_LLM` | `--embedded-llm` |
| `K13D_EMBEDDED_LLM_PORT` | `--embedded-llm-port` |
| `K13D_EMBEDDED_LLM_MODEL` | `--embedded-llm-model` |
| `K13D_EMBEDDED_LLM_CONTEXT` | `--embedded-llm-context` |
| `K13D_DOWNLOAD_MODEL` | `--download-model` |
| `K13D_EMBEDDED_LLM_STATUS` | `--embedded-llm-status` |

## Notes

- `KUBECONFIG` is supported through Kubernetes client-go loading rules.
- `--auth-mode ldap` and `--auth-mode oidc` select those auth paths, but the stock binary does not yet expose every provider-specific LDAP/OIDC field as first-class CLI flags.
- There is no `--kubeconfig`, `--context`, `--debug`, `--host`, `--tls`, `--password`, `report`, or `bench` CLI in the current binary.
- `config.yaml` is loaded first, then environment variables override it, then explicit CLI flags override those defaults.

## Next Steps

- [Environment Variables](env-vars.md)
- [Configuration](../getting-started/configuration.md)
- [API Reference](api.md)
