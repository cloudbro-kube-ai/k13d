# API Reference

k13d exposes a REST API for programmatic access to Kubernetes management features.

## Authentication

### Login

```http
POST /api/auth/login
Content-Type: application/json

{
  "password": "your-password"
}
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-01-16T10:30:00Z"
}
```

### Using Token

Include the token in subsequent requests:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

### Logout

```http
POST /api/auth/logout
Authorization: Bearer <token>
```

## Health & Status

### Health Check

```http
GET /api/health
```

Response:
```json
{
  "status": "healthy",
  "kubernetes": "connected",
  "llm": "available"
}
```

## Kubernetes Resources

### List Resources

```http
GET /api/k8s/{resource}
GET /api/k8s/{resource}?namespace={ns}
```

Resources: `pods`, `deployments`, `services`, `configmaps`, `secrets`, `nodes`, etc.

Example:
```bash
curl http://localhost:8080/api/k8s/pods?namespace=default
```

Response:
```json
{
  "items": [
    {
      "name": "nginx-abc123",
      "namespace": "default",
      "status": "Running",
      "ready": "1/1",
      "restarts": 0,
      "age": "2d"
    }
  ],
  "total": 1
}
```

### Get Resource

```http
GET /api/k8s/{resource}/{name}?namespace={ns}
```

Example:
```bash
curl http://localhost:8080/api/k8s/pods/nginx-abc123?namespace=default
```

### Get YAML

```http
GET /api/k8s/{resource}/{name}/yaml?namespace={ns}
```

### Describe Resource

```http
GET /api/k8s/{resource}/{name}/describe?namespace={ns}
```

### Delete Resource

```http
DELETE /api/k8s/{resource}/{name}?namespace={ns}
```

## Pod Operations

### Get Logs

```http
GET /api/pods/{name}/logs?namespace={ns}&container={container}&tail={lines}
```

Parameters:
- `namespace`: Pod namespace
- `container`: Container name (optional if single container)
- `tail`: Number of lines (default: 100)
- `follow`: Stream logs (default: false)
- `previous`: Previous container logs (default: false)

Example:
```bash
curl "http://localhost:8080/api/pods/nginx-abc123/logs?namespace=default&tail=50"
```

### Execute Command

```http
POST /api/pods/{name}/exec
Content-Type: application/json

{
  "namespace": "default",
  "container": "nginx",
  "command": ["sh", "-c", "ls -la"]
}
```

### Port Forward

```http
POST /api/portforward
Content-Type: application/json

{
  "namespace": "default",
  "pod": "nginx-abc123",
  "localPort": 8888,
  "podPort": 80
}
```

Response:
```json
{
  "id": "pf-123",
  "localPort": 8888,
  "status": "active"
}
```

Stop port forward:
```http
DELETE /api/portforward/{id}
```

## Deployment Operations

### Scale

```http
POST /api/deployment/scale
Content-Type: application/json

{
  "namespace": "default",
  "name": "nginx",
  "replicas": 5
}
```

### Restart

```http
POST /api/deployment/restart
Content-Type: application/json

{
  "namespace": "default",
  "name": "nginx"
}
```

### Rollback

```http
POST /api/deployment/rollback
Content-Type: application/json

{
  "namespace": "default",
  "name": "nginx",
  "revision": 2
}
```

## Node Operations

### Cordon

```http
POST /api/node/cordon
Content-Type: application/json

{
  "name": "node-1"
}
```

### Uncordon

```http
POST /api/node/uncordon
Content-Type: application/json

{
  "name": "node-1"
}
```

### Drain

```http
POST /api/node/drain
Content-Type: application/json

{
  "name": "node-1",
  "ignoreDaemonSets": true,
  "deleteLocalData": false,
  "force": false
}
```

## Metrics

### Pod Metrics

```http
GET /api/metrics/pod?namespace={ns}
```

Response:
```json
{
  "items": [
    {
      "name": "nginx-abc123",
      "namespace": "default",
      "cpu": "50m",
      "memory": "128Mi"
    }
  ]
}
```

### Node Metrics

```http
GET /api/metrics/node
```

## Helm

### List Releases

```http
GET /api/helm/releases?namespace={ns}
```

### Install Chart

```http
POST /api/helm/install
Content-Type: application/json

{
  "name": "my-release",
  "namespace": "default",
  "chart": "nginx",
  "repo": "https://charts.bitnami.com/bitnami",
  "values": {
    "replicaCount": 3
  }
}
```

### Upgrade Release

```http
POST /api/helm/upgrade
Content-Type: application/json

{
  "name": "my-release",
  "namespace": "default",
  "chart": "nginx",
  "values": {
    "replicaCount": 5
  }
}
```

### Uninstall Release

```http
DELETE /api/helm/releases/{name}?namespace={ns}
```

## AI Chat

### Send Message (SSE)

```http
POST /api/chat/agentic
Content-Type: application/json

{
  "message": "Why is my nginx pod failing?",
  "context": {
    "namespace": "default",
    "resource": "pod/nginx-abc123"
  }
}
```

Response (Server-Sent Events):
```
event: chunk
data: {"content": "Let me check"}

event: chunk
data: {"content": " your nginx pod..."}

event: tool_request
data: {"tool": "kubectl", "command": "get pod nginx-abc123 -o yaml", "id": "tc-123"}

event: tool_execution
data: {"id": "tc-123", "result": "..."}

event: stream_end
data: {"complete": true}
```

### Approve Tool

```http
POST /api/tool/approve
Content-Type: application/json

{
  "id": "tc-123",
  "approved": true
}
```

### Get Chat History

```http
GET /api/chat/history?session_id={id}
```

### Clear Chat

```http
DELETE /api/chat/history?session_id={id}
```

## Audit

### Get Audit Logs

```http
GET /api/audit?limit={n}&offset={offset}
```

Response:
```json
{
  "items": [
    {
      "id": 1,
      "timestamp": "2024-01-15T10:30:00Z",
      "user": "admin",
      "action": "execute",
      "resource": "pod/nginx",
      "details": {
        "command": "kubectl get pods"
      }
    }
  ],
  "total": 100
}
```

### Filter Audit Logs

```http
GET /api/audit?action={action}&user={user}&from={timestamp}&to={timestamp}
```

## Settings

### Get Settings

```http
GET /api/settings
```

### Update Settings

```http
PUT /api/settings
Content-Type: application/json

{
  "language": "en",
  "beginner_mode": false,
  "auto_approve_readonly": true
}
```

### Get LLM Config

```http
GET /api/settings/llm
```

### Update LLM Config

```http
PUT /api/settings/llm
Content-Type: application/json

{
  "provider": "openai",
  "model": "gpt-4",
  "api_key": "sk-..."
}
```

## MCP Servers

### List MCP Servers

```http
GET /api/mcp/servers
```

### Add MCP Server

```http
POST /api/mcp/servers
Content-Type: application/json

{
  "name": "thinking",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-sequential-thinking"],
  "enabled": true
}
```

### Enable/Disable Server

```http
PATCH /api/mcp/servers/{name}
Content-Type: application/json

{
  "enabled": true
}
```

### Remove MCP Server

```http
DELETE /api/mcp/servers/{name}
```

## Reports

### Generate Report

```http
POST /api/reports
Content-Type: application/json

{
  "type": "cluster-overview",
  "format": "pdf",
  "namespace": "default"
}
```

Response:
```json
{
  "id": "report-123",
  "status": "generating"
}
```

### Get Report Status

```http
GET /api/reports/{id}
```

### Download Report

```http
GET /api/reports/{id}/download
```

## Error Responses

### Standard Error Format

```json
{
  "error": "Error message",
  "code": "ERROR_CODE",
  "details": {}
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `UNAUTHORIZED` | 401 | Invalid or missing token |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `VALIDATION_ERROR` | 400 | Invalid request |
| `K8S_ERROR` | 500 | Kubernetes API error |
| `LLM_ERROR` | 500 | LLM provider error |

## Rate Limiting

API requests are rate limited:

- Default: 100 requests/minute
- AI Chat: 10 requests/minute

Rate limit headers:
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1705318800
```

## SDK Examples

### cURL

```bash
# Login
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"password": "secret"}' | jq -r '.token')

# List pods
curl http://localhost:8080/api/k8s/pods \
  -H "Authorization: Bearer $TOKEN"

# AI chat
curl -X POST http://localhost:8080/api/chat/agentic \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "list pods"}'
```

### Python

```python
import requests

base_url = "http://localhost:8080"

# Login
resp = requests.post(f"{base_url}/api/auth/login",
                     json={"password": "secret"})
token = resp.json()["token"]

headers = {"Authorization": f"Bearer {token}"}

# List pods
pods = requests.get(f"{base_url}/api/k8s/pods", headers=headers)
print(pods.json())
```

### JavaScript

```javascript
const baseUrl = 'http://localhost:8080';

// Login
const loginResp = await fetch(`${baseUrl}/api/auth/login`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ password: 'secret' })
});
const { token } = await loginResp.json();

// List pods
const podsResp = await fetch(`${baseUrl}/api/k8s/pods`, {
  headers: { 'Authorization': `Bearer ${token}` }
});
const pods = await podsResp.json();
```

## Next Steps

- [CLI Reference](cli.md) - Command line options
- [Environment Variables](env-vars.md) - Configuration
- [Configuration](../getting-started/configuration.md) - Full config guide
