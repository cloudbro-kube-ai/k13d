# MCP Server Kubernetes - Installation & Setup Guide

## Overview

The **mcp-server-kubernetes** is an MCP (Model Context Protocol) server that provides Kubernetes-related tools to k13d's AI agent. It enables the AI to inspect, manage, and troubleshoot Kubernetes clusters through a standardized interface.

## Installation

### Prerequisites

- Node.js 18+ or npm available
- kubectl installed and configured
- Valid kubeconfig file (~/.kube/config)

### Step 1: Install mcp-server-kubernetes

```bash
# Install globally (recommended)
npm install -g @anthropic/mcp-server-kubernetes

# Or install locally
npm install @anthropic/mcp-server-kubernetes
```

Verify installation:

```bash
npx @anthropic/mcp-server-kubernetes --help
```

### Step 2: Configure in k13d

#### Option A: Using Web UI (Recommended)

1. Start k13d:

   ```bash
   k13d -web -port 8080
   ```

2. Open browser: http://localhost:8080

3. Navigate to: **Settings** → **MCP Servers**

4. Click **"Add MCP Server"** button

5. Fill in the form:
   - **Server Name**: `kubernetes`
   - **Command**: `npx`
   - **Arguments** (one per line):
     - `-y`
     - `@anthropic/mcp-server-kubernetes`
   - **Environment Variables** (optional):
     - Key: `KUBECONFIG`, Value: `~/.kube/config` (or leave empty for default)
   - **Description**: `Kubernetes resource management and inspection tools`
   - **Enable**: Check the box to enable immediately

6. Click **"Save"**

7. Verify connection:
   - The server should show "Status: connected"
   - Available tools list should appear below

#### Option B: Using Configuration File

Edit `~/.config/k13d/config.yaml`:

```yaml
mcp:
  servers:
    - name: kubernetes
      command: npx
      args:
        - -y
        - "@anthropic/mcp-server-kubernetes"
      env:
        KUBECONFIG: "~/.kube/config" # Optional: specify kubeconfig path
      description: "Kubernetes resource management and inspection tools"
      enabled: true
```

Then restart k13d:

```bash
k13d -web -port 8080
```

### Step 3: Verify Installation

Check in Web UI:

1. Go to **Settings** → **MCP Servers**
2. Look for "kubernetes" server in the list
3. Verify **Status** shows **"connected"**
4. Click on server to see available tools

Via API:

```bash
curl -X GET http://localhost:8080/api/mcp/servers \
  -H "Cookie: session=<your-session-id>"
```

## Available Tools

The following tools are available once mcp-server-kubernetes is connected:

### Pod Management

- **get_pods** - List all pods in a namespace
  ```
  namespace: string (required)
  ```
- **describe_pod** - Get detailed information about a pod

  ```
  namespace: string (required)
  pod_name: string (required)
  ```

- **get_logs** - Get logs from a pod

  ```
  namespace: string (required)
  pod_name: string (required)
  container: string (optional)
  lines: integer (optional, default: 100)
  follow: boolean (optional, stream logs)
  ```

- **delete_pod** - Delete a pod
  ```
  namespace: string (required)
  pod_name: string (required)
  grace_period: integer (optional, graceful termination seconds)
  ```

### Deployment Management

- **get_deployments** - List all deployments in a namespace

  ```
  namespace: string (required)
  ```

- **describe_deployment** - Get deployment details

  ```
  namespace: string (required)
  deployment_name: string (required)
  ```

- **scale_deployment** - Scale deployment replicas

  ```
  namespace: string (required)
  name: string (required)
  replicas: integer (required)
  ```

- **update_deployment** - Update deployment (image, replicas, etc)
  ```
  namespace: string (required)
  name: string (required)
  patch: object (JSON patch)
  ```

### StatefulSet & DaemonSet Management

- **get_statefulsets** - List StatefulSets
- **get_daemonsets** - List DaemonSets
- **describe_statefulset** - Get StatefulSet details
- **describe_daemonset** - Get DaemonSet details

### Job & CronJob Management

- **get_jobs** - List Jobs
- **get_cronjobs** - List CronJobs
- **describe_job** - Get Job details

### Manifest Management

- **apply_manifest** - Apply Kubernetes manifest (YAML)

  ```
  manifest: string (YAML content, required)
  namespace: string (optional, default: default)
  dry_run: boolean (optional, preview changes)
  ```

- **delete_resource** - Delete a resource
  ```
  kind: string (Pod, Deployment, Service, etc, required)
  namespace: string (required)
  name: string (required)
  cascade: boolean (optional, delete dependent objects)
  ```

### Event & Status Monitoring

- **get_events** - Get cluster events

  ```
  namespace: string (optional, empty for all namespaces)
  limit: integer (optional, default: 50)
  ```

- **get_resource_status** - Get resource status
  ```
  kind: string (Pod, Deployment, etc, required)
  namespace: string (required)
  ```

### Port Forwarding

- **port_forward** - Forward ports from pod
  ```
  namespace: string (required)
  pod_name: string (required)
  local_port: integer (required)
  remote_port: integer (required)
  ```

## Configuration Details

### Environment Variables

Control the `mcp-server-kubernetes` server behavior through environment variables:

```yaml
mcp:
  servers:
    - name: kubernetes
      command: npx
      args: ["-y", "@anthropic/mcp-server-kubernetes"]
      env:
        # Kubernetes configuration
        KUBECONFIG: "~/.kube/config" # Path to kubeconfig file
        KUBECTL_CONTEXT: "docker-desktop" # K8s context to use
        KUBECTL_NAMESPACE: "default" # Default namespace

        # SSL/TLS configuration
        INSECURE_SKIP_VERIFY: "false" # Skip certificate verification

        # Logging
        DEBUG: "false" # Enable debug logging

        # Limits
        DEFAULT_POD_LOG_LINES: "100" # Lines of logs to retrieve
        MAX_MANIFESTS_SIZE: "1048576" # Max manifest size (bytes)
```

### Multiple Kubeconfigs

To work with multiple clusters, create separate MCP servers:

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

## Usage Examples

### In AI Chat Interface

1. **List pods:**

   ```
   User: "Show me all running pods in the default namespace"
   AI: Uses get_pods tool
       → Lists pods with status, restarts, and age
   ```

2. **Get pod logs:**

   ```
   User: "What are the recent logs from the nginx pod?"
   AI: Uses get_logs tool
       → Retrieves and displays recent pod logs
   ```

3. **Apply manifest:**

   ```
   User: "Deploy this application: [YAML content]"
   AI: Uses apply_manifest tool
       → Applies the manifest and reports status
   ```

4. **Troubleshoot issues:**

   ```
   User: "Why is the web deployment failing?"
   AI: Uses multiple tools (get_deployment, get_pods, get_events)
       → Analyzes status and provides diagnosis
   ```

5. **Manage resources:**
   ```
   User: "Scale the web deployment to 5 replicas"
   AI: Uses scale_deployment tool
       → Updates replica count and confirms
   ```

## Troubleshooting

### Server Connection Failed

**Problem:** MCP server shows "disconnected" status

**Solutions:**

1. Verify npx can run the command:

   ```bash
   npx -y @anthropic/mcp-server-kubernetes --help
   ```

2. Check kubeconfig file exists and is readable:

   ```bash
   cat ~/.kube/config
   ```

3. Verify kubectl access:

   ```bash
   kubectl get pods
   ```

4. Try manual connection in k13d logs:
   ```bash
   DEBUG=true k13d -web -port 8080
   ```

### Cannot List Resources

**Problem:** AI gets "permission denied" errors

**Solutions:**

1. Check current kubectl authentication:

   ```bash
   kubectl auth can-i list pods --all-namespaces
   ```

2. Verify current context:

   ```bash
   kubectl config current-context
   ```

3. Update kubeconfig if needed:
   ```bash
   aws eks update-kubeconfig --name <cluster-name>
   ```

### Performance Issues

**Problem:** Slow responses from MCP server

**Solutions:**

1. Limit namespace scope:

   ```yaml
   env:
     KUBECTL_NAMESPACE: "default" # Limit to specific namespace
   ```

2. Reduce log lines:

   ```yaml
   env:
     DEFAULT_POD_LOG_LINES: "50"
   ```

3. Check cluster load:
   ```bash
   kubectl top nodes
   kubectl top pods
   ```

## Managing in Web UI

### View Servers

- Settings → MCP Servers → List all configured servers

### Get Server Details

- Click on server name to see:
  - Connection status
  - Available tools
  - Configuration details (command, args, env vars)

### Update Configuration

- Click server → Edit button
- Modify command, arguments, or environment variables
- Save changes
- Server reconnects automatically

### Toggle Enable/Disable

- Toggle switch next to server name
- Enabled servers are automatically connected on startup

### Delete Server

- Click server → Delete button
- Confirm deletion
- Server is disconnected and removed from config

## Security Considerations

1. **RBAC**: Ensure the user running k13d has appropriate API keys
2. **Network**: MCP server communicates via stdin/stdout (safe)
3. **Logs**: Be careful with pod logs (may contain sensitive data)
4. **Manifests**: AI can apply resources - use with caution
5. **Kubeconfig**: Protect kubeconfig file (contains credentials)

## Advanced Topics

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

### Custom MCP Servers

You can also create custom MCP servers. See [Creating Custom MCP Servers](/docs/MCP_GUIDE.md#creating-custom-mcp-servers) in the MCP Guide.

## See Also

- [MCP Guide](/docs/MCP_GUIDE.md) - Complete MCP documentation
- [Configuration Guide](/docs/CONFIGURATION_GUIDE.md) - k13d configuration options
- [@anthropic/mcp-server-kubernetes GitHub](https://github.com/anthropics/mcp-server-kubernetes)
