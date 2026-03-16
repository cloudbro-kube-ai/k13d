# Environment Variables

Reference for environment variables that `k13d` currently reads.

Default file paths are based on the platform XDG config directory:

| Platform | Default config path |
|----------|---------------------|
| Linux | `${XDG_CONFIG_HOME:-~/.config}/k13d/config.yaml` |
| macOS | `~/Library/Application Support/k13d/config.yaml` |
| Windows | `%AppData%\\k13d\\config.yaml` |

## Config & Startup

| Variable | Description | Default |
|----------|-------------|---------|
| `K13D_CONFIG` | Override `config.yaml` path | `<XDG config home>/k13d/config.yaml` |
| `K13D_WEB` | Start in Web UI mode | `false` |
| `K13D_PORT` | Web server port | `8080` |
| `K13D_NAMESPACE` | Initial namespace | cluster default |
| `K13D_ALL_NAMESPACES` | Start with all namespaces | `false` |
| `KUBECONFIG` | Kubeconfig path used by client-go | client-go default |
| `XDG_CONFIG_HOME` | XDG config base directory override | platform default |

## Authentication

| Variable | Description | Default |
|----------|-------------|---------|
| `K13D_AUTH_MODE` | `token`, `local`, `ldap`, `oidc` | `token` |
| `K13D_NO_AUTH` | Disable authentication | `false` |
| `K13D_USERNAME` | Default admin username for local auth | `admin` |
| `K13D_PASSWORD` | Default admin password for local auth | random if omitted |
| `K13D_JWT_SECRET` | JWT signing secret | auto-generated if omitted |
| `K13D_DEFAULT_ROLE` | Default TUI RBAC role | `admin` |
| `K13D_CORS_ALLOWED_ORIGINS` | Extra allowed CORS origins | none |

## Storage

| Variable | Description | Default |
|----------|-------------|---------|
| `K13D_DB_PATH` | SQLite database path | `<XDG config home>/k13d/audit.db` |
| `K13D_NO_DB` | Disable DB-backed persistence | `false` |

## Generic LLM Overrides

These variables override `config.yaml` values directly:

| Variable | Description |
|----------|-------------|
| `K13D_LLM_PROVIDER` | Active LLM provider |
| `K13D_LLM_MODEL` | Active model |
| `K13D_LLM_ENDPOINT` | Custom API endpoint |
| `K13D_LLM_API_KEY` | API key |

## Embedded LLM Removal

Embedded LLM support has been removed from the runtime.

- Use `K13D_LLM_PROVIDER=ollama` with `OLLAMA_HOST` for local inference.
- Old `K13D_EMBEDDED_*` and `K13D_DOWNLOAD_MODEL` variables are no longer recognized.

## Provider-Specific Fallbacks

If `api_key` or `endpoint` is not set in `config.yaml`, the provider factory also checks:

| Variable | Used For |
|----------|----------|
| `OPENAI_API_KEY` | `openai`, fallback for `upstage`/`solar` |
| `UPSTAGE_API_KEY` | `upstage` / `solar` |
| `ANTHROPIC_API_KEY` | `anthropic` |
| `GOOGLE_API_KEY` | `gemini` |
| `AZURE_OPENAI_API_KEY` | `azopenai` / `azure` |
| `AZURE_OPENAI_ENDPOINT` | `azopenai` / `azure` endpoint |
| `OLLAMA_HOST` | `ollama` endpoint fallback |
| `AWS_ACCESS_KEY_ID` | `bedrock` |
| `AWS_SECRET_ACCESS_KEY` | `bedrock` |
| `AWS_SESSION_TOKEN` | `bedrock` |
| `AWS_REGION` | `bedrock` |

## Examples

### Web UI With Local Auth

```bash
export K13D_WEB=true
export K13D_AUTH_MODE=local
export K13D_USERNAME=admin
export K13D_PASSWORD=changeme
k13d
```

### OpenAI Via Environment Only

```bash
export K13D_LLM_PROVIDER=openai
export K13D_LLM_MODEL=gpt-4o
export OPENAI_API_KEY=sk-your-key
k13d --web --auth-mode local
```

### Ollama

```bash
export K13D_LLM_PROVIDER=ollama
export K13D_LLM_MODEL=gpt-oss:20b
export OLLAMA_HOST=localhost:11434
k13d
```

### Custom Config Path

```bash
export K13D_CONFIG=/etc/k13d/config.yaml
k13d --web
```

## Notes

- `config.yaml` values support shell-style placeholders such as `${OPENAI_API_KEY}`.
- Environment variables override values loaded from `config.yaml`.
- Web UI startup logs print `Config File`, `Config Path Source`, and `Env Overrides`, which is useful when debugging unexpected config values.
- `K13D_AUTH_MODE=ldap` and `K13D_AUTH_MODE=oidc` select those auth paths, but the stock binary does not yet expose every provider-specific LDAP/OIDC field as environment variables.
- Variables not listed here are not currently wired into the runtime.
