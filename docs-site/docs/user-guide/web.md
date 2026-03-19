# Web Dashboard

The Web Dashboard provides a modern browser-based interface for Kubernetes management with full AI integration.

## Overview

The Web UI offers:

- **Modern Interface**: Responsive design with dark/light themes
- **Real-time Updates**: Stale-first dashboard refresh with background revalidation
- **AI Assistant**: Integrated chat interface
- **Multi-cluster**: Switch between contexts
- **Reports**: Generate cluster analysis reports

## Getting Started

### Launch Web Mode

```bash
# Start web server on default port 8080
k13d --web

# Specify custom port
k13d --web --port 3000

# With local authentication
k13d --web --auth-mode local
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
- **Freshness badge** when cached data is shown first and live data is still revalidating

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

By default, this modal appears for both read-only and write AI tool actions. You only skip it for read-only commands if you explicitly enable auto-approve in Settings.

## Features

### Dark/Light Theme

Toggle theme in Settings or click the theme icon in the header.

### Real-time Updates

Resources update automatically. The dashboard now follows a stale-while-revalidate pattern:

- recent data is reused immediately when you switch resources or reload the page
- live data is fetched in the background and the table updates in place
- the header shows a freshness badge when cached data is being refreshed

Configure refresh interval:

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

The Web UI saves active LLM settings back to `config.yaml` immediately and can also manage named profiles through **Add Model Profile**, **Use**, and **Delete**.

For the full storage model, including how `llm`, `models[]`, and `active_model` change, see [Model Settings & Storage](../ai-llm/model-settings-storage.md).

### AI Input History

The AI input box supports shell-like history recall:

- ++arrow-up++ loads the previous submitted prompt
- ++arrow-down++ moves forward again
- on a single-line draft, history works even when the cursor is at the end of the line
- in a multi-line draft, history only takes over from the first line on ++arrow-up++ and the last line on ++arrow-down++, so normal caret movement still works inside the textarea

The recent prompt history is stored in browser `localStorage` under `k13d_query_history`, so it survives reloads in the same browser profile.

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
- Collapsible sidebar and AI panel
- Condensed panel header and filter bar on narrow screens
- Horizontal table scrolling with sticky pagination for resource-heavy views
- Optimized for smaller screens without dropping core dashboard actions

## Security

### Authentication

Run the Web UI with local authentication:

```bash
k13d --web --auth-mode local
```

For production, prefer token auth:

```bash
k13d --web --auth-mode token
```

Provider-specific LDAP/OIDC settings are startup-configured in the current build and are not persisted from the Web UI settings page.

### HTTPS

For production, use a reverse proxy (nginx, traefik) with TLS.

## Troubleshooting

### WebSocket Connection Failed

- Check if port is accessible
- Verify no firewall blocking
- Try different browser

### Slow Performance

- k13d now shows recent cached resource data first and refreshes in the background, so the first paint should feel faster after the initial load
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
