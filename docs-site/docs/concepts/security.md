# Security & RBAC

k13d provides comprehensive security features including authentication, authorization, command safety analysis, and audit logging.

## Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    Security Layers                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │Authentication│─►│Authorization │─►│Safety Checks │          │
│  │              │  │   (RBAC)     │  │              │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│         │                 │                   │                 │
│         ▼                 ▼                   ▼                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    Audit Logging                          │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Authentication

### Web UI Authentication

The Web UI supports multiple authentication methods:

#### Password Authentication

```yaml
# config.yaml
auth:
  password: "your-secure-password"  # SHA256 hashed
```

#### Token Authentication

```yaml
auth:
  token: "your-api-token"
```

### Kubernetes RBAC

k13d respects Kubernetes RBAC. Users can only access resources their kubeconfig allows.

## Command Safety Analysis

All AI-generated commands pass through the safety analyzer before execution.

### Safety Levels

| Level | Description | Action | Examples |
|-------|-------------|--------|----------|
| **Read-Only** | Non-modifying commands | Auto-approve (configurable) | `get`, `describe`, `logs` |
| **Write** | Resource modifications | Requires approval | `apply`, `create`, `patch` |
| **Dangerous** | Destructive operations | Warning + approval | `delete`, `drain`, `taint` |
| **Interactive** | Requires TTY | Requires approval | `exec`, `attach`, `edit` |

### AST-Based Analysis

k13d uses AST parsing for accurate command analysis:

```
"kubectl get pods | xargs rm -rf /"
         │
         ▼
┌─────────────────┐
│  Shell Parser   │  mvdan.cc/sh/v3
│  (AST Parsing)  │
└────────┬────────┘
         │
         ▼
    Detects: pipe, xargs, rm
    Result: DANGEROUS
```

### Configuration

```yaml
safety:
  auto_approve_readonly: true    # Auto-approve read-only commands
  require_approval_for_write: true
  block_dangerous: false         # Block dangerous commands entirely
  blocked_patterns:              # Regex patterns to block
    - "rm -rf /"
    - "kubectl delete ns kube-system"
```

## Audit Logging

All actions are logged to SQLite for compliance and troubleshooting.

### Logged Events

| Event | Description |
|-------|-------------|
| `login` | User authentication |
| `query` | AI queries |
| `approve` | Tool approvals |
| `reject` | Tool rejections |
| `execute` | Command executions |
| `error` | Errors |

### Audit Log Schema

```sql
CREATE TABLE audit_logs (
    id INTEGER PRIMARY KEY,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    user TEXT,
    action TEXT,
    resource TEXT,
    details TEXT,      -- JSON
    llm_request TEXT,  -- LLM request
    llm_response TEXT  -- LLM response
);
```

### Viewing Audit Logs

**TUI Mode:**

```
:audit
```

**Web Mode:**

Navigate to Settings → Audit Logs

**API:**

```bash
curl http://localhost:8080/api/audit
```

## Session Management

### Session Storage

| Type | Location | Persistence |
|------|----------|-------------|
| Memory | RAM | Process lifetime |
| Filesystem | `~/.config/k13d/sessions/` | Persistent |

### Session Configuration

```yaml
sessions:
  storage: filesystem    # memory or filesystem
  max_age: 24h          # Session expiry
  max_sessions: 100     # Max concurrent sessions
```

## Network Security

### TLS Configuration

For production deployments:

```yaml
web:
  tls:
    enabled: true
    cert_file: /path/to/cert.pem
    key_file: /path/to/key.pem
```

### CORS Settings

```yaml
web:
  cors:
    allowed_origins:
      - "https://your-domain.com"
    allowed_methods:
      - GET
      - POST
```

## MCP Server Security

### Isolation

MCP servers run as separate processes with isolated permissions.

### Environment Variables

Be careful with environment variables containing secrets:

```yaml
mcp:
  servers:
    - name: github
      command: npx
      args: ["-y", "@modelcontextprotocol/server-github"]
      env:
        GITHUB_TOKEN: "${GITHUB_TOKEN}"  # From environment
```

### Sandboxing

For enhanced security, run MCP servers in containers:

```yaml
mcp:
  servers:
    - name: secure-tool
      command: docker
      args:
        - run
        - --rm
        - --read-only
        - --network=none
        - mcp-server:latest
```

## Best Practices

### 1. Use Strong Passwords

```bash
# Generate a strong password
openssl rand -base64 32
```

### 2. Enable Audit Logging

```yaml
enable_audit: true
```

### 3. Review Approvals

Always review AI-generated commands before approving.

### 4. Use Minimal RBAC

Grant only the Kubernetes permissions needed:

```yaml
# Example: Read-only access
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k13d-readonly
rules:
  - apiGroups: [""]
    resources: ["pods", "services", "configmaps"]
    verbs: ["get", "list", "watch"]
```

### 5. Secure Secrets

Never commit secrets to config files:

```yaml
# Bad
auth:
  password: "mysecret"

# Good - use environment variables
auth:
  password: "${K13D_PASSWORD}"
```

## Compliance

### GDPR Considerations

- Audit logs may contain PII
- Configure retention policies
- Enable encryption at rest

### SOC 2

- Enable comprehensive audit logging
- Use TLS for all connections
- Implement access controls

## Next Steps

- [Configuration](../getting-started/configuration.md) - Security configuration
- [Deployment](../deployment/kubernetes.md) - Secure deployment
- [Architecture](architecture.md) - Security architecture details
