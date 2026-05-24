# MCP Server Examples for k13d

This directory contains example configurations and setup guides for MCP (Model Context Protocol) servers that work with k13d.

## ⚡ MCP Profiles - Fastest Way to Get Started

**New!** Use MCP Profiles to install pre-configured DevOps toolkits with one click.

**Available Profiles:**

- 🐳 **Kubernetes** (`k8s`) - K8s cluster management
- 🐳 **Docker** (`docker`) - Container management
- 🔧 **Shell & Bash** (`shell`) - System automation
- 🚀 **GitHub** (`github`) - Repository management
- ☁️ **AWS** (`aws`) - AWS infrastructure
- 🔄 **ArgoCD** (`argocd`) - GitOps deployments
- 🧠 **Sequential Thinking** (`thinking`) - Problem solving
- 🎯 **Full DevOps Stack** (`fullstack`) - Everything combined

**Get Started in 30 seconds:**

1. Open k13d: `./k13d -web -port 8080`
2. Go to: Settings → MCP Servers
3. Click **Install** on any profile
4. Tools instantly available in AI chat

👉 **[→ See Profiles Quick Start Guide](profiles-quickstart.md)**

---

## Available Examples

### 1. Kubernetes MCP Server (`kubernetes-mcp-config.yaml`)

The **mcp-server-kubernetes** provides tools for managing and inspecting Kubernetes clusters within the AI agent loop.

**Features:**

- List and describe resources (pods, deployments, services, etc.)
- Read pod logs
- Apply and delete manifests
- Port forwarding
- Event monitoring
- Resource status tracking

**Setup:**

- See [kubernetes-mcp-setup.md](./kubernetes-mcp-setup.md) for complete installation and usage guide
- Or use the configuration from [kubernetes-mcp-config.yaml](./kubernetes-mcp-config.yaml)
- **Or use Profiles**: Click "Install" on the **Kubernetes** profile (Recommended!)

## Quick Start

### Using MCP Profiles (Recommended)

The easiest way - see [profiles-quickstart.md](profiles-quickstart.md)

### Using Web UI (Manual Setup)

1. Start k13d:

   ```bash
   k13d -web -port 8080
   ```

2. Open http://localhost:8080 in your browser

3. Navigate to **Settings** → **MCP Servers**

4. Click **"Add MCP Server"** and fill in:
   - Name: `kubernetes`
   - Command: `npx`
   - Args: `-y`, `@anthropic/mcp-server-kubernetes`
   - Enabled: `true`

5. Click **Save**

6. Verify connection shows **"Status: connected"**

### Using Configuration File

1. Copy the configuration from `kubernetes-mcp-config.yaml` to `~/.config/k13d/config.yaml`

2. Update the `mcp.servers` section with your preferred servers

3. Restart k13d:
   ```bash
   k13d -web -port 8080
   ```

## API Endpoints

### List MCP Servers

```bash
GET /api/mcp/servers

# Response includes all registered servers and their connection status
```

### Get Server Details

```bash
GET /api/mcp/server-detail?name=kubernetes

# Response includes configuration, connection status, and available tools
```

### Add MCP Server

```bash
POST /api/mcp/servers
Content-Type: application/json

{
  "name": "kubernetes",
  "command": "npx",
  "args": ["-y", "@anthropic/mcp-server-kubernetes"],
  "description": "Kubernetes resource management",
  "enabled": true
}
```

### Update MCP Server

```bash
PUT /api/mcp/server-detail?name=kubernetes
Content-Type: application/json

{
  "description": "Updated description",
  "enabled": true,
  "args": ["-y", "@anthropic/mcp-server-kubernetes", "--verbose"]
}
```

### Delete MCP Server

```bash
DELETE /api/mcp/server-detail?name=kubernetes
```

### List Available Tools

```bash
GET /api/mcp/tools

# Returns both MCP tools and built-in tools (kubectl, bash)
```

## Required Setup

### System Requirements

- **Node.js 18+** - Required for running MCP servers via npx
- **kubectl** - For Kubernetes server configuration
- **kubeconfig** - Valid Kubernetes configuration file

### Installation

```bash
# Install required MCP servers
npm install -g @anthropic/mcp-server-kubernetes
npm install -g @modelcontextprotocol/server-sequential-thinking

# Verify installation
npx @anthropic/mcp-server-kubernetes --help
```

## Configuration Details

### Default Configuration (`~/.config/k13d/config.yaml`)

```yaml
mcp:
  servers:
    - name: kubernetes
      command: npx
      args:
        - -y
        - "@anthropic/mcp-server-kubernetes"
      env:
        KUBECONFIG: "~/.kube/config" # Optional, uses default if empty
      description: "Kubernetes resource management and inspection tools"
      enabled: true
```

### Environment Variables

Control MCP server behavior through environment variables:

| Variable            | Purpose                 | Example          |
| ------------------- | ----------------------- | ---------------- |
| `KUBECONFIG`        | Path to kubeconfig file | `~/.kube/config` |
| `KUBECTL_CONTEXT`   | Kubernetes context      | `docker-desktop` |
| `KUBECTL_NAMESPACE` | Default namespace       | `default`        |
| `DEBUG`             | Enable debug logging    | `true`/`false`   |

## Usage Examples

### In AI Chat

```
User: "Show me all pods in the default namespace"
→ AI uses kubernetes MCP server's get_pods tool
→ Returns pod list with status

User: "What are the recent logs from nginx?"
→ AI uses get_logs tool
→ Returns recent pod logs

User: "Deploy this application..."
→ AI uses apply_manifest tool
→ Applies Kubernetes manifest
```

### Available Tools

When mcp-server-kubernetes is connected, these tools are available:

- `get_pods` - List pods
- `get_deployments` - List deployments
- `describe_pod` - Get pod details
- `get_logs` - Get pod logs
- `apply_manifest` - Apply YAML manifest
- `delete_resource` - Delete resource
- `get_events` - Get cluster events
- `get_resource_status` - Check resource status
- And many more...

## Troubleshooting

### Server Won't Connect

1. Check npx can run the server:

   ```bash
   npx @anthropic/mcp-server-kubernetes --help
   ```

2. Verify kubeconfig exists:

   ```bash
   cat ~/.kube/config
   kubectl get pods
   ```

3. Check k13d logs:
   ```bash
   DEBUG=true k13d -web -port 8080
   ```

### Permission Denied Errors

```bash
# Check RBAC permissions
kubectl auth can-i list pods --all-namespaces

# Update kubeconfig if needed
aws eks update-kubeconfig --name <cluster-name>
```

## Managing MCP Servers

### View All Servers

**Web UI:**

- Settings → MCP Servers

**API:**

```bash
curl http://localhost:8080/api/mcp/servers \
  -H "Cookie: session=<session-id>"
```

### Enable/Disable Server

**Web UI:**

- Find server in list, toggle "Enabled" switch

**API:**

```bash
curl -X PUT http://localhost:8080/api/mcp/servers \
  -H "Content-Type: application/json" \
  -H "Cookie: session=<session-id>" \
  -d '{
    "name": "kubernetes",
    "action": "enable"  # or "disable", "reconnect"
  }'
```

### View Server Details

**Web UI:**

- Click server name to see configuration and available tools

**API:**

```bash
curl "http://localhost:8080/api/mcp/server-detail?name=kubernetes" \
  -H "Cookie: session=<session-id>"
```

## Advanced Configuration

### Multiple Kubernetes Clusters

Create separate MCP servers for different clusters:

```yaml
mcp:
  servers:
    - name: kubernetes-prod
      command: npx
      args: ["-y", "@anthropic/mcp-server-kubernetes"]
      env:
        KUBECONFIG: "~/.kube/prod-config"
        KUBECTL_CONTEXT: "prod-cluster"
      enabled: true

    - name: kubernetes-dev
      command: npx
      args: ["-y", "@anthropic/mcp-server-kubernetes"]
      env:
        KUBECONFIG: "~/.kube/dev-config"
        KUBECTL_CONTEXT: "dev-cluster"
      enabled: true
```

### Docker-Based MCP Server

If you prefer running MCP server in a container:

```yaml
mcp:
  servers:
    - name: kubernetes-docker
      command: docker
      args:
        - run
        - --rm
        - --interactive
        - -v
        - "${HOME}/.kube:/root/.kube:ro"
        - "mcp-kubernetes:latest"
      enabled: true
```

## See Also

- [MCP Guide](../../docs/MCP_GUIDE.md) - Complete MCP documentation
- [Configuration Guide](../../docs/CONFIGURATION_GUIDE.md) - k13d configuration
- [Kubernetes MCP Setup](./kubernetes-mcp-setup.md) - Detailed setup guide
- [@anthropic/mcp-server-kubernetes](https://github.com/anthropics/mcp-server-kubernetes) - GitHub repository

## Support

For issues, questions, or feature requests, refer to:

- k13d Issues: https://github.com/kube-ai-dashboard/k13d/issues
- MCP Documentation: https://modelcontextprotocol.io
