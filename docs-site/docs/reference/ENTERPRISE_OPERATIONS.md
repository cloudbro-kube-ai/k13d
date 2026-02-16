# k13d Enterprise Operations Guide

This guide provides best practices for deploying, monitoring, and operating k13d in production environments.

## Table of Contents

- [Production Deployment Checklist](#production-deployment-checklist)
- [Monitoring & Observability](#monitoring--observability)
- [Troubleshooting Guide](#troubleshooting-guide)
- [Performance Tuning](#performance-tuning)
- [Security Hardening](#security-hardening)
- [High Availability](#high-availability)

---

## Production Deployment Checklist

### Pre-Deployment Verification

Before deploying k13d to production, verify the following:

#### 1. Infrastructure Requirements

| Resource | Minimum | Recommended | Notes |
|----------|---------|-------------|-------|
| **CPU** | 0.5 cores | 1-2 cores | More cores improve AI response time |
| **Memory** | 512MB | 1-2GB | Depends on cluster size and AI model |
| **Storage** | 100MB | 500MB | For audit logs and SQLite database |
| **Network** | - | - | Outbound to K8s API and LLM endpoint |

#### 2. Network Connectivity

- [ ] **Kubernetes API Access**: k13d needs access to the Kubernetes API server
  - Test: `kubectl cluster-info` from the deployment environment
  - Required ports: TCP 443 (HTTPS) or TCP 6443 (K8s API default)

- [ ] **LLM Provider Access**: Outbound HTTPS to AI provider endpoints
  - OpenAI: `api.openai.com:443`
  - Anthropic: `api.anthropic.com:443`
  - Ollama (local): `localhost:11434`
  - Test: `curl -I https://api.openai.com` (or your provider)

- [ ] **Internal Access** (Web UI mode): Ingress or LoadBalancer configuration
  - Default port: 8080 (configurable with `-port` flag)
  - HTTPS/TLS recommended for production

#### 3. Authentication & Authorization

- [ ] **Kubernetes RBAC**: Create a dedicated ServiceAccount with appropriate permissions
  ```yaml
  apiVersion: v1
  kind: ServiceAccount
  metadata:
    name: k13d
    namespace: k13d-system
  ---
  apiVersion: rbac.authorization.k8s.io/v1
  kind: ClusterRole
  metadata:
    name: k13d-viewer
  rules:
  - apiGroups: ["*"]
    resources: ["*"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["pods/log", "pods/exec"]
    verbs: ["get", "create"]
  ---
  apiVersion: rbac.authorization.k8s.io/v1
  kind: ClusterRoleBinding
  metadata:
    name: k13d-viewer
  roleRef:
    apiGroup: rbac.authorization.k8s.io
    kind: ClusterRole
    name: k13d-viewer
  subjects:
  - kind: ServiceAccount
    name: k13d
    namespace: k13d-system
  ```

- [ ] **AI API Keys**: Store securely using Kubernetes Secrets
  ```bash
  kubectl create secret generic k13d-ai-keys \
    --from-literal=OPENAI_API_KEY=sk-... \
    --namespace=k13d-system
  ```

- [ ] **Web UI Authentication** (if using web mode):
  - Configure authentication in `config.yaml`
  - Use strong passwords (minimum 12 characters)
  - Consider integrating with SSO/OIDC

#### 4. Configuration Management

- [ ] Create production `config.yaml` with:
  - Appropriate `log_level` (default: `info`, use `debug` only for troubleshooting)
  - `enable_audit: true` for compliance tracking
  - LLM provider settings (use environment variables for secrets)
  - Resource limits and safety settings

- [ ] Store configuration in a ConfigMap:
  ```bash
  kubectl create configmap k13d-config \
    --from-file=config.yaml=config.yaml \
    --namespace=k13d-system
  ```

#### 5. Security Scan

- [ ] Scan k13d binary/image for vulnerabilities
  ```bash
  # For Docker images
  trivy image ghcr.io/kube-ai-dashboard/k13d:latest

  # For binary
  syft packages k13d | grype
  ```

- [ ] Review security policy: [SECURITY.md](../SECURITY.md)

---

## Monitoring & Observability

### Log Levels

k13d supports multiple log levels configured via `log_level` in `config.yaml`:

| Level | Description | Use Case |
|-------|-------------|----------|
| `debug` | Verbose logging | Development, troubleshooting |
| `info` | Standard logging | **Production default** |
| `warn` | Warnings only | Quiet production |
| `error` | Errors only | Minimal logging |

**Production Recommendation**: Use `info` level by default, switch to `debug` for troubleshooting.

### Audit Logging

k13d maintains an audit log of all user actions in a SQLite database.

#### Audit Database Location

| Deployment Mode | Path |
|-----------------|------|
| TUI (local) | `~/.local/share/k13d/audit.db` |
| Web (container) | `/data/k13d/audit.db` (mount a persistent volume) |

#### Audit Log Schema

```sql
CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    user TEXT,
    action TEXT,
    resource TEXT,
    namespace TEXT,
    details TEXT,
    success BOOLEAN,
    error_msg TEXT
);
```

#### Querying Audit Logs

```bash
# View recent actions
sqlite3 /data/k13d/audit.db "SELECT * FROM audit_log ORDER BY timestamp DESC LIMIT 10;"

# Failed operations
sqlite3 /data/k13d/audit.db "SELECT * FROM audit_log WHERE success = 0;"

# Actions by user
sqlite3 /data/k13d/audit.db "SELECT * FROM audit_log WHERE user = 'admin@example.com';"

# Dangerous operations (delete, kill)
sqlite3 /data/k13d/audit.db "SELECT * FROM audit_log WHERE action IN ('delete', 'kill');"
```

### Health Indicators

Monitor these metrics for k13d health:

#### 1. Process Health

```bash
# Check if k13d is running
ps aux | grep k13d

# CPU and memory usage
top -p $(pgrep k13d)
```

#### 2. Kubernetes API Connectivity

```bash
# From within the container/pod
kubectl cluster-info

# Test API access
kubectl get nodes
```

#### 3. LLM Provider Connectivity

k13d logs LLM connection status:
- Look for `"AI client initialized"` (info level)
- Look for `"AI test connection failed"` (error level)

```bash
# Check logs for AI connectivity
kubectl logs -n k13d-system deployment/k13d | grep -i "AI\|LLM"
```

#### 4. Web UI Health (Web Mode)

```bash
# Health check endpoint
curl http://localhost:8080/health

# Expected response:
# {"status":"ok","version":"x.y.z"}
```

### Prometheus Metrics (Planned)

Future releases will include Prometheus metrics at `/metrics`:
- `k13d_k8s_api_requests_total` - Total Kubernetes API requests
- `k13d_ai_requests_total` - Total AI requests
- `k13d_ai_request_duration_seconds` - AI request latency
- `k13d_audit_events_total` - Total audit events

---

## Troubleshooting Guide

### Common Issues & Solutions

#### Issue 1: "Failed to load pods: connection refused"

**Symptoms:**
- Resources fail to load in dashboard
- Error: `connection refused` or `Unable to connect to the server`

**Causes:**
- Kubernetes API server is unreachable
- Incorrect kubeconfig
- Network policy blocking access
- RBAC permissions insufficient

**Solutions:**

1. **Verify kubeconfig**:
   ```bash
   kubectl cluster-info
   kubectl get nodes
   ```

2. **Check ServiceAccount permissions** (if running in-cluster):
   ```bash
   kubectl auth can-i list pods --as=system:serviceaccount:k13d-system:k13d
   ```

3. **Verify network connectivity**:
   ```bash
   # From k13d pod
   nc -zv kubernetes.default.svc.cluster.local 443
   ```

4. **Check API server logs**:
   ```bash
   kubectl logs -n kube-system -l component=kube-apiserver
   ```

---

#### Issue 2: TUI Rendering Problems

**Symptoms:**
- Garbled text or misaligned columns
- Colors not displaying correctly
- Screen flickering

**Causes:**
- Terminal emulator incompatibility
- `TERM` environment variable misconfigured
- tmux/screen session issues

**Solutions:**

1. **Set correct TERM variable**:
   ```bash
   export TERM=xterm-256color
   k13d
   ```

2. **Try different terminal emulators**:
   - **Recommended**: iTerm2 (macOS), Windows Terminal, Alacritty
   - **Avoid**: Basic Terminal.app (macOS), cmd.exe (Windows)

3. **Reset terminal**:
   ```bash
   reset
   k13d
   ```

4. **Check tmux configuration** (if using tmux):
   ```bash
   # Add to ~/.tmux.conf
   set -g default-terminal "screen-256color"
   ```

---

#### Issue 3: AI Assistant Not Responding

**Symptoms:**
- AI queries timeout or return errors
- "Failed to init model" message
- No AI suggestions

**Causes:**
- Invalid or missing API key
- LLM provider endpoint unreachable
- Rate limiting
- Model not available

**Solutions:**

1. **Verify API key**:
   ```bash
   echo $OPENAI_API_KEY
   # Should print: sk-...
   ```

2. **Test LLM connectivity manually**:
   ```bash
   # OpenAI
   curl https://api.openai.com/v1/models \
     -H "Authorization: Bearer $OPENAI_API_KEY"

   # Ollama (local)
   curl http://localhost:11434/api/tags
   ```

3. **Check rate limits**:
   - OpenAI: View usage at platform.openai.com
   - Increase timeout in `config.yaml` (default: 30s)

4. **Try a different model**:
   ```yaml
   llm:
     provider: openai
     model: gpt-3.5-turbo  # Faster, cheaper alternative
   ```

5. **Enable debug logging**:
   ```yaml
   log_level: debug
   ```
   Then check logs for detailed AI errors.

---

#### Issue 4: High Memory Usage

**Symptoms:**
- k13d consuming >2GB RAM
- OOMKilled in Kubernetes
- System slowdown

**Causes:**
- Large cluster (1000+ resources)
- Excessive watch connections
- Memory leak (report to GitHub)

**Solutions:**

1. **Filter namespaces**:
   - Use namespace filtering to reduce loaded resources
   - Press `0-9` for quick namespace switch
   - Use command mode: `:ns` to browse namespaces

2. **Increase memory limits** (Kubernetes):
   ```yaml
   resources:
     limits:
       memory: 2Gi
     requests:
       memory: 1Gi
   ```

3. **Disable unused features**:
   ```yaml
   enable_audit: false  # Reduces database overhead
   ```

4. **Restart periodically**:
   - Set up a CronJob to restart k13d daily (if memory leak suspected)

---

#### Issue 5: Slow Performance

**Symptoms:**
- Slow resource loading
- UI lag when switching views
- AI responses take >30s

**Causes:**
- Large cluster
- Slow Kubernetes API
- Network latency to LLM provider
- Insufficient resources

**Solutions:**

1. **Check Kubernetes API latency**:
   ```bash
   kubectl get pods --v=6
   # Look for request duration in verbose output
   ```

2. **Use faster LLM models**:
   - OpenAI: `gpt-3.5-turbo` (faster than `gpt-4`)
   - Local: `qwen2.5:3b` (faster than `7b` models)

3. **Increase resources**:
   - Allocate 2 CPU cores
   - Use local Ollama for AI (eliminates network latency)

4. **Optimize view refreshes**:
   - Disable auto-refresh if not needed
   - Use manual refresh (`r` key) instead

---

## Performance Tuning

### CPU Optimization

1. **Local LLM (Ollama)**:
   - Eliminates network latency
   - Requires 4-8GB RAM depending on model
   - Best for air-gapped environments

2. **Resource Limits** (Kubernetes):
   ```yaml
   resources:
     requests:
       cpu: 500m
       memory: 512Mi
     limits:
       cpu: 2
       memory: 2Gi
   ```

### Memory Optimization

1. **Namespace Filtering**:
   - Load only required namespaces
   - Use `-namespace` flag for single-namespace mode

2. **Disable Audit Logging** (if not required):
   ```yaml
   enable_audit: false
   ```

3. **Reduce Log Retention**:
   ```bash
   # Rotate logs daily
   logrotate -f /etc/logrotate.d/k13d
   ```

### Network Optimization

1. **Use In-Cluster Deployment**:
   - Lower latency to Kubernetes API
   - No network policy overhead

2. **Local AI Models**:
   ```yaml
   llm:
     provider: ollama
     endpoint: http://localhost:11434
     model: qwen2.5:3b
   ```

3. **Persistent Connections**:
   - k13d uses Kubernetes watch API for efficient resource updates
   - No periodic polling needed

---

## Security Hardening

### 1. Principle of Least Privilege

**Kubernetes RBAC:**

Create restrictive roles for different use cases:

```yaml
# Read-only viewer (safest)
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k13d-readonly
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]

---
# Logs and exec access (for debugging)
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k13d-debug
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["pods/log", "pods/exec"]
  verbs: ["get", "create"]

---
# Full operator access (use with caution)
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k13d-operator
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
```

### 2. AI Safety Settings

k13d includes a safety analyzer that prevents dangerous operations. Configure in `config.yaml`:

```yaml
ai_safety:
  require_approval: true  # Require user confirmation for destructive commands
  dangerous_commands:     # Block these patterns entirely
    - "rm -rf /"
    - "--force"
    - "DROP TABLE"
  rate_limit:
    max_commands_per_minute: 10
```

### 3. Network Isolation

**Kubernetes NetworkPolicy:**

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: k13d-netpol
  namespace: k13d-system
spec:
  podSelector:
    matchLabels:
      app: k13d
  policyTypes:
  - Egress
  egress:
  # Allow Kubernetes API access
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: TCP
      port: 443
  # Allow DNS
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: UDP
      port: 53
  # Allow LLM provider (if not using Ollama)
  - to:
    - podSelector: {}
    ports:
    - protocol: TCP
      port: 443
```

### 4. Audit Logging

Enable comprehensive audit logging:

```yaml
enable_audit: true
audit_retention_days: 90  # Keep audit logs for 90 days
```

Export audit logs to external systems:

```bash
# Daily export to S3
0 0 * * * sqlite3 /data/k13d/audit.db ".dump" | gzip | \
  aws s3 cp - s3://my-bucket/k13d-audit-$(date +\%Y-\%m-\%d).sql.gz
```

### 5. Secret Management

**Best Practices:**

1. **Never commit API keys to Git**
2. **Use Kubernetes Secrets** for sensitive data:
   ```bash
   kubectl create secret generic k13d-secrets \
     --from-literal=OPENAI_API_KEY=sk-... \
     --from-literal=WEB_UI_PASSWORD=changeme
   ```

3. **Use external secret managers** (advanced):
   - AWS Secrets Manager
   - HashiCorp Vault
   - Azure Key Vault

---

## High Availability

### Multiple Instance Deployment

k13d supports running multiple instances for high availability.

#### Session Management

- **TUI Mode**: Each user runs their own k13d process (no shared state)
- **Web UI Mode**: Sessions are stored in-memory (ephemeral)
  - For production, use sticky sessions (session affinity)
  - Consider adding Redis for shared session storage (future enhancement)

#### Load Balancing

```yaml
apiVersion: v1
kind: Service
metadata:
  name: k13d-web
  namespace: k13d-system
spec:
  type: LoadBalancer
  sessionAffinity: ClientIP  # Sticky sessions
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 3600  # 1 hour
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: k13d
```

#### Replicas Configuration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k13d-web
  namespace: k13d-system
spec:
  replicas: 3  # Run 3 instances for HA
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  selector:
    matchLabels:
      app: k13d
  template:
    metadata:
      labels:
        app: k13d
    spec:
      containers:
      - name: k13d
        image: ghcr.io/kube-ai-dashboard/k13d:latest
        args: ["-web", "-port", "8080"]
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

### Failover Patterns

1. **Active-Passive**: Single active instance, standby for failures
2. **Active-Active**: Multiple instances behind load balancer (requires session affinity)
3. **Blue-Green**: Deploy new version, switch traffic after validation

### Disaster Recovery

#### Backup Strategy

**Critical Data:**
- Audit logs (`audit.db`)
- Configuration files (`config.yaml`)

**Backup Script:**
```bash
#!/bin/bash
BACKUP_DIR="/backups/k13d-$(date +%Y-%m-%d)"
mkdir -p "$BACKUP_DIR"

# Backup audit database
cp /data/k13d/audit.db "$BACKUP_DIR/"

# Backup configuration
kubectl get configmap k13d-config -n k13d-system -o yaml > "$BACKUP_DIR/config.yaml"

# Backup secrets (encrypted)
kubectl get secret k13d-secrets -n k13d-system -o yaml > "$BACKUP_DIR/secrets.yaml"

# Upload to S3
tar czf - "$BACKUP_DIR" | aws s3 cp - s3://my-bucket/k13d-backup-$(date +%Y-%m-%d).tar.gz
```

---

## Operational Runbook

### Deployment Checklist

- [ ] Review infrastructure requirements
- [ ] Verify network connectivity (K8s API, LLM provider)
- [ ] Create ServiceAccount with appropriate RBAC
- [ ] Store API keys in Kubernetes Secrets
- [ ] Create ConfigMap with production config
- [ ] Deploy k13d (Deployment + Service)
- [ ] Configure Ingress/LoadBalancer
- [ ] Test health endpoints
- [ ] Verify audit logging
- [ ] Set up monitoring and alerting
- [ ] Document access procedures for team

### Maintenance Tasks

| Task | Frequency | Command |
|------|-----------|---------|
| Update k13d | Monthly | `kubectl set image deployment/k13d k13d=ghcr.io/kube-ai-dashboard/k13d:latest` |
| Rotate logs | Daily | `logrotate -f /etc/logrotate.d/k13d` |
| Backup audit DB | Daily | See backup script above |
| Review audit logs | Weekly | `sqlite3 audit.db "SELECT ..."` |
| Check for CVEs | Weekly | `trivy image ghcr.io/kube-ai-dashboard/k13d:latest` |
| Update API keys | As needed | `kubectl create secret --dry-run=client ...` |

---

## Support & Resources

- **GitHub Issues**: https://github.com/kube-ai-dashboard/k13d/issues
- **Security Reports**: See [SECURITY.md](../SECURITY.md)
- **Documentation**: [User Guide](./USER_GUIDE.md), [Architecture](./ARCHITECTURE.md)
- **Contributing**: [CONTRIBUTING.md](../CONTRIBUTING.md)

---

## License

k13d is open source software licensed under the MIT License.
