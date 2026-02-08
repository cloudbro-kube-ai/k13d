# Environment Variables

Complete reference for k13d environment variables.

## Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `K13D_CONFIG` | string | `~/.config/k13d/config.yaml` | Config file path |
| `XDG_CONFIG_HOME` | string | `~/.config` | XDG config base directory |

## Kubernetes

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `KUBECONFIG` | string | `~/.kube/config` | Kubeconfig file path |
| `K13D_CONTEXT` | string | current | Kubernetes context name |
| `K13D_NAMESPACE` | string | `default` | Default namespace |
| `K13D_ALL_NAMESPACES` | bool | `false` | View all namespaces |

## Web Server

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `K13D_WEB` | bool | `false` | Enable web mode |
| `K13D_PORT` | int | `8080` | HTTP server port |
| `K13D_HOST` | string | `0.0.0.0` | Bind address |
| `K13D_PASSWORD` | string | - | Web UI password |

### TLS

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `K13D_TLS` | bool | `false` | Enable TLS |
| `K13D_TLS_CERT` | string | - | TLS certificate path |
| `K13D_TLS_KEY` | string | - | TLS private key path |

## LLM Providers

### OpenAI

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `OPENAI_API_KEY` | string | - | OpenAI API key |
| `OPENAI_ORG_ID` | string | - | OpenAI organization ID |
| `OPENAI_BASE_URL` | string | - | Custom API base URL |

### Anthropic

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `ANTHROPIC_API_KEY` | string | - | Anthropic API key |

### Google Gemini

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `GOOGLE_API_KEY` | string | - | Google AI API key |

### Azure OpenAI

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `AZURE_OPENAI_API_KEY` | string | - | Azure OpenAI key |
| `AZURE_OPENAI_ENDPOINT` | string | - | Azure endpoint URL |
| `AZURE_OPENAI_API_VERSION` | string | `2024-02-15-preview` | API version |

### AWS Bedrock

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `AWS_ACCESS_KEY_ID` | string | - | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | string | - | AWS secret key |
| `AWS_SESSION_TOKEN` | string | - | AWS session token |
| `AWS_REGION` | string | - | AWS region |
| `AWS_PROFILE` | string | - | AWS profile name |

### Ollama

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `OLLAMA_HOST` | string | `localhost:11434` | Ollama server address |

### General LLM

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `K13D_LLM_PROVIDER` | string | `openai` | LLM provider name |
| `K13D_LLM_MODEL` | string | `gpt-4` | Model name |
| `K13D_LLM_ENDPOINT` | string | - | Custom endpoint URL |
| `K13D_LLM_API_KEY` | string | - | Generic API key |
| `K13D_EMBEDDED_LLM` | bool | `false` | Use embedded LLM |

## Features

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `K13D_BEGINNER_MODE` | bool | `false` | Enable beginner mode |
| `K13D_LANGUAGE` | string | `en` | UI language |
| `K13D_ENABLE_AUDIT` | bool | `true` | Enable audit logging |
| `K13D_AUTO_APPROVE_READONLY` | bool | `true` | Auto-approve read commands |
| `K13D_REFRESH_INTERVAL` | duration | `5s` | Dashboard refresh interval |

## Logging

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `K13D_DEBUG` | bool | `false` | Enable debug mode |
| `K13D_LOG_LEVEL` | string | `info` | Log level |
| `K13D_LOG_FILE` | string | - | Log file path |
| `K13D_LOG_FORMAT` | string | `text` | Log format (text, json) |

## Database

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `K13D_DB_PATH` | string | `~/.config/k13d/audit.db` | SQLite database path |

## Reports

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `K13D_REPORT_PATH` | string | `~/k13d-reports` | Default report output path |
| `K13D_REPORT_FORMAT` | string | `markdown` | Default report format |

## MCP

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `K13D_MCP_SERVERS` | string | - | MCP servers JSON config |
| `K13D_MCP_TIMEOUT` | duration | `30s` | MCP call timeout |

## Session

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `K13D_SESSION_STORAGE` | string | `memory` | Session storage type |
| `K13D_SESSION_MAX_AGE` | duration | `24h` | Session expiry time |
| `K13D_SESSION_PATH` | string | `~/.config/k13d/sessions` | Session file path |

## Development

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `K13D_DEV_MODE` | bool | `false` | Development mode |
| `K13D_DISABLE_TELEMETRY` | bool | `false` | Disable telemetry |
| `K13D_PROFILE` | bool | `false` | Enable profiling |
| `K13D_PROFILE_PORT` | int | `6060` | pprof port |

## Examples

### Basic Setup

```bash
# Set API key
export OPENAI_API_KEY=sk-your-key-here

# Run k13d
k13d
```

### Production Web Server

```bash
export K13D_WEB=true
export K13D_PORT=8080
export K13D_PASSWORD=secure-password
export K13D_TLS=true
export K13D_TLS_CERT=/etc/ssl/certs/k13d.crt
export K13D_TLS_KEY=/etc/ssl/private/k13d.key
export OPENAI_API_KEY=sk-your-key

k13d
```

### Local Ollama

```bash
export K13D_LLM_PROVIDER=ollama
export K13D_LLM_MODEL=llama3.2
export OLLAMA_HOST=localhost:11434

k13d
```

### AWS Bedrock

```bash
export K13D_LLM_PROVIDER=bedrock
export K13D_LLM_MODEL=anthropic.claude-3-sonnet-20240229-v1:0
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret

k13d
```

### Multi-Cluster

```bash
# Switch between clusters
export KUBECONFIG=~/.kube/prod-config
k13d -web -port 8081 &

export KUBECONFIG=~/.kube/staging-config
k13d -web -port 8082 &
```

### Docker

```bash
docker run -d \
  -p 8080:8080 \
  -v ~/.kube:/root/.kube:ro \
  -e OPENAI_API_KEY=$OPENAI_API_KEY \
  -e K13D_PASSWORD=secret \
  cloudbro/k13d:latest \
  -web
```

### Kubernetes

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: k13d-secrets
type: Opaque
stringData:
  OPENAI_API_KEY: sk-your-key
  K13D_PASSWORD: secure-password
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k13d
spec:
  template:
    spec:
      containers:
        - name: k13d
          envFrom:
            - secretRef:
                name: k13d-secrets
          env:
            - name: K13D_WEB
              value: "true"
            - name: K13D_PORT
              value: "8080"
            - name: K13D_DEBUG
              value: "false"
```

## Precedence

Environment variables override config file settings but are overridden by CLI flags:

1. Default values
2. Config file
3. **Environment variables**
4. CLI flags

## Security

### Best Practices

1. **Never commit secrets**: Use environment variables for API keys
2. **Use secret managers**: Vault, AWS Secrets Manager, etc.
3. **Rotate keys regularly**
4. **Limit access**: Set restrictive file permissions

### Secure Shell History

```bash
# Prevent secrets in shell history
 export OPENAI_API_KEY=sk-...  # Note: leading space

# Or use read
read -s OPENAI_API_KEY
export OPENAI_API_KEY
```

### .env File

```bash
# .env (add to .gitignore!)
OPENAI_API_KEY=sk-your-key
K13D_PASSWORD=secure-password
```

```bash
# Load .env
set -a; source .env; set +a
k13d
```

## Next Steps

- [CLI Reference](cli.md) - Command line options
- [API Reference](api.md) - REST API
- [Configuration](../getting-started/configuration.md) - Config file options
