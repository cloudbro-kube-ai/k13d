# Web Dashboard

The Web Dashboard provides a modern browser-based interface for Kubernetes management with full AI integration.

## Overview

The Web UI offers:

- **Modern Interface**: Responsive design with dark/light themes
- **Real-time Updates**: Live resource status updates
- **AI Assistant**: Integrated chat interface
- **Multi-cluster**: Switch between contexts
- **Reports**: Generate cluster analysis reports

## Getting Started

### Launch Web Mode

```bash
# Start web server on default port 8080
k13d -web

# Specify custom port
k13d -web -port 3000

# With authentication
k13d -web -password "your-secure-password"

# With embedded LLM (no API key needed)
k13d -web --embedded-llm
```

### Access the Dashboard

Open your browser to: `http://localhost:8080`

## Interface Layout

```
┌─────────────────────────────────────────────────────────────────┐
│ k13d │ Context: prod │ Namespace: default │ [Settings] [Help]  │
├───────────┬─────────────────────────────────────────────────────┤
│           │                                                     │
│ Resources │  Resource Table                                     │
│ ─────────│                                                     │
│ Pods      │  NAME         READY  STATUS   AGE                  │
│ Services  │  nginx-abc    1/1    Running  2d                   │
│ Deploys   │  api-def      2/2    Running  5d                   │
│ ConfigMaps│                                                     │
│ Secrets   │                                                     │
│ ...       │                                                     │
│           │                                                     │
├───────────┴─────────────────────────────────────────────────────┤
│ AI Assistant                                                     │
│ ───────────────────────────────────────────────────────────────│
│ Ask me anything about your cluster...                           │
│ [Send]                                                           │
└─────────────────────────────────────────────────────────────────┘
```

## Navigation

### Sidebar

| Section | Description |
|---------|-------------|
| **Workloads** | Pods, Deployments, StatefulSets, DaemonSets |
| **Config** | ConfigMaps, Secrets |
| **Network** | Services, Ingresses, Endpoints |
| **Storage** | PVs, PVCs, StorageClasses |
| **Cluster** | Nodes, Namespaces, Events |
| **Helm** | Helm releases |

### Resource Table

- **Click** row to view details
- **Search** bar for filtering
- **Namespace** dropdown for switching
- **Refresh** button for manual refresh

## Resource Actions

### Right-Click Menu

Right-click on any resource to see available actions:

| Action | Description |
|--------|-------------|
| **View YAML** | Display full YAML manifest |
| **Describe** | Show resource description |
| **Edit** | Edit resource (opens YAML editor) |
| **Delete** | Delete resource (with confirmation) |
| **AI Analyze** | Get AI analysis |

### Quick Actions

| Button | Description |
|--------|-------------|
| 📋 | Copy resource name |
| 📄 | View YAML |
| 🔍 | Describe |
| 🗑️ | Delete |
| 🤖 | AI Analyze |

## AI Assistant

### Chat Interface

1. Type your question in the input field
2. Click "Send" or press ++enter++
3. View streaming response
4. Approve/reject tool requests

### Example Queries

```
"Why is my nginx pod crashing?"
"Scale the api deployment to 5 replicas"
"Show me all pods with high CPU usage"
"Explain this HPA configuration"
```

### Tool Approval

When AI needs to execute a command:

```
┌──────────────────────────────────────┐
│ Tool Approval Required                │
│                                      │
│ kubectl get pods -n production       │
│                                      │
│ [Approve]  [Reject]                  │
└──────────────────────────────────────┘
```

## Features

### Dark/Light Theme

Toggle theme in Settings or click the theme icon in the header.

### Real-time Updates

Resources update automatically. Configure refresh interval:

Settings → General → Refresh Interval

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| ++ctrl+k++ | Focus search |
| ++ctrl+slash++ | Toggle AI panel |
| ++esc++ | Close modal |
| ++question++ | Show shortcuts |

## Settings

### LLM Configuration

Settings → AI → LLM Configuration

| Setting | Description |
|---------|-------------|
| **Provider** | OpenAI, Ollama, Gemini, Anthropic |
| **Model** | gpt-4, llama3.2, etc. |
| **Endpoint** | Custom API endpoint |
| **API Key** | Provider API key |

### MCP Servers

Settings → AI → MCP Servers

Manage external MCP servers for extended AI capabilities.

### User Management

Settings → Admin → Users

| Action | Description |
|--------|-------------|
| Add User | Create new user |
| Edit | Modify user settings |
| Delete | Remove user |

## Reports

Generate comprehensive cluster reports with selectable sections:

1. Navigate to Reports section
2. Select report type:
   - **Cluster Overview** - General health
   - **Security Audit** - Security findings
   - **Cost Analysis** - Resource costs
3. Choose which sections to include (Nodes, Namespaces, Workloads, Events, Security, FinOps, Metrics)
4. Configure options:
   - Namespace filter
   - Output format
5. Click "Generate"

## Custom Resource Detail

Click on any Custom Resource to view a rich detail modal:

- **Overview** tab with auto-detected status, metadata, key fields, spec/status summary, conditions table, labels, and annotations
- **YAML** tab with full manifest
- **Events** tab with related Kubernetes events

## Pod Actions

### View Logs

1. Click on a pod
2. Select container (if multiple)
3. View streaming logs
4. Options:
   - Previous logs
   - Follow
   - Timestamps
   - Download

### Execute Shell

1. Click on a pod
2. Click "Exec" or 🖥️ icon
3. Select container
4. Enter commands in terminal

### Port Forward

1. Click on a pod
2. Click "Port Forward"
3. Configure:
   - Local port
   - Container port
4. Click "Start"
5. Access at `localhost:<port>`

## Deployment Actions

### Scale

1. Select deployment
2. Click "Scale" or use slider
3. Enter replica count
4. Confirm

### Restart

1. Select deployment
2. Click "Restart"
3. Confirm rollout restart

### Rollback

1. Select deployment
2. Click "Rollback"
3. Select revision
4. Confirm

## Node Actions

### Cordon/Uncordon

1. Select node
2. Click "Cordon" to prevent scheduling
3. Click "Uncordon" to allow scheduling

### Drain

1. Select node
2. Click "Drain"
3. Configure options:
   - Ignore DaemonSets
   - Delete local data
   - Force
4. Confirm

## API Access

The Web UI exposes REST APIs:

```bash
# Get pods
curl http://localhost:8080/api/k8s/pods

# With authentication
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/k8s/pods

# AI chat
curl -X POST http://localhost:8080/api/chat/agentic \
     -H "Content-Type: application/json" \
     -d '{"message": "list pods"}'
```

## Mobile Support

The Web UI is responsive and works on mobile devices:

- Touch-friendly navigation
- Swipe gestures
- Collapsible panels
- Optimized for smaller screens

## Security

### Authentication

Enable password authentication:

```bash
k13d -web -password "secure-password"
```

Or in config:

```yaml
auth:
  password: "secure-password"
```

### HTTPS

For production, use a reverse proxy (nginx, traefik) with TLS.

## Troubleshooting

### WebSocket Connection Failed

- Check if port is accessible
- Verify no firewall blocking
- Try different browser

### Slow Performance

- Reduce refresh interval
- Limit namespace scope
- Use filters to reduce data

### AI Not Responding

- Check LLM configuration
- Verify API key
- Check network connectivity

## Next Steps

- [TUI Dashboard](tui.md) - Terminal interface
- [Keyboard Shortcuts](shortcuts.md) - All shortcuts
- [Configuration](../getting-started/configuration.md) - Full options
