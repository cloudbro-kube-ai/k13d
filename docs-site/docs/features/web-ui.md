# Web UI Features

Complete feature reference for k13d Web UI with screenshots.

---

## Dashboard

The main dashboard provides a real-time overview of your Kubernetes cluster.

### Full Screen Overview

![Web UI Full Screen](../images/webui-full-screen.png)

The main interface consists of three panels:

- **Left**: Resource navigation sidebar
- **Center**: Resource table with details
- **Right**: AI Assistant panel

### Center Panel - Resource Table

![Center Panel](../images/webui-center-pannel.png)

| Feature | Description |
|---------|-------------|
| **Resource Table** | Sortable, filterable table of K8s resources |
| **Status Indicators** | Color-coded status (Running, Pending, Failed) |
| **Quick Actions** | One-click View, Edit, Delete, Scale |
| **Namespace Selector** | Switch between namespaces |
| **Search/Filter** | Real-time filtering by name, status |
| **Auto-refresh** | Configurable refresh interval (5s default) |

### Pod Detail Modal

![Pod Detail Modal](../images/webui-pods-detail-modal.png)

Click on any resource to view detailed information including:

- Full YAML specification
- Related events
- Container status
- Labels and annotations

---

## Topology View

Visualize cluster resources and their relationships.

### Cluster Topology

![Topology View](../images/webui-topology-view-all.png)

Interactive graph showing:

- Deployments, Services, Pods relationships
- Network connections
- Resource dependencies

### Topology Detail Modal

![Topology Modal](../images/webui-topology-modal-view.png)

Click any node to view resource details and navigate to related resources.

---

## AI Assistant

Integrated AI assistant with natural language understanding and tool execution.

### Assistant Panel

![AI Assistant Panel](../images/webui-assistant-pannel.png)

| Feature | Description |
|---------|-------------|
| **Natural Language** | Ask questions in plain English/Korean/Chinese/Japanese |
| **Streaming Responses** | Real-time SSE streaming with live cursor |
| **Context Awareness** | AI receives YAML, Events, Logs context |
| **Tool Calling** | Executes kubectl, bash commands |
| **History** | Conversation history within session |

### MCP Tool Calling (Debug Mode)

![MCP Tool Call Debug](../images/webui-mcp-tool-call-debugmode.png)

Enable debug mode to see:

- Tool call requests
- Raw API responses
- Execution timing

### Required Decision (Approval)

![Required Decision](../images/webui-decision-required.png)

When AI requests a write/dangerous operation:

1. **Approval Dialog** appears with command details
2. **Command Preview** shows exact command to execute
3. **Safety Warning** for dangerous commands
4. **Approve/Reject** buttons for user decision

![Required Decision Detail](../images/webui-decision-required-2.png)

---

## Reports

Generate comprehensive cluster analysis reports.

### Report Index

![Reports Index](../images/webui-cluster-assessment-report-index.png)

Available report types with generated report history.

### Generate Cluster Report

![Generate Report](../images/webui-generate-cluster-report.png)

| Report Type | Description |
|-------------|-------------|
| **Cluster Overview** | Node status, workload summary, health indicators |
| **Security Audit** | RBAC analysis, network policies, vulnerabilities |
| **Resource Optimization** | Over-provisioned resources, cost analysis |
| **AI Analysis** | AI-powered insights and recommendations |

### Security Assessment Report

![Security Assessment](../images/webui-security-assessment.png)

Comprehensive security analysis including:

- RBAC configuration review
- Network policy audit
- Vulnerability assessment

### Infrastructure Report

![Infrastructure Report](../images/webui-report-cluster-infrastructure.png)

Cluster infrastructure analysis with:

- Node health status
- Resource utilization
- Capacity planning recommendations

### FinOps Cost Analysis

![FinOps Report](../images/webui-report-finops-cost-analysis.png)

Cost optimization insights:

- Resource utilization analysis
- Over-provisioned workloads
- Cost reduction recommendations

---

## Metrics & Monitoring

Real-time metrics visualization with Chart.js.

### Metrics Dashboard

![Metrics Dashboard](../images/webui-metrics.png)

| Metric | Description |
|--------|-------------|
| **CPU Usage** | Real-time CPU consumption per pod/node |
| **Memory Usage** | Memory utilization with limits |
| **Network I/O** | Ingress/egress traffic |
| **Storage** | PV usage and capacity |

---

## Terminal & Logs

Interactive pod access and log viewing.

### Pod Terminal

![Pod Terminal](../images/webui-pod-terminal-access.png)

| Feature | Description |
|---------|-------------|
| **xterm.js** | Full terminal emulator in browser |
| **Container Selection** | Multi-container pod support |
| **Shell Selection** | /bin/bash, /bin/sh options |
| **Copy/Paste** | Clipboard support |
| **Resize** | Automatic terminal resize |

### Log Viewer

![Log Viewer](../images/webui-logs-tail-modal.png)

| Feature | Description |
|---------|-------------|
| **Real-time Streaming** | Live log tail with auto-scroll |
| **ANSI Colors** | Full color support |
| **Filter/Search** | Filter logs by pattern |
| **Download** | Export logs to file |
| **Previous Logs** | View crashed container logs |
| **Multi-container** | Select container for multi-container pods |

---

## Port Forward

Forward container ports to local machine.

### Port Forward Management

![Port Forward](../images/webui-port-forword-modal.png)

| Feature | Description |
|---------|-------------|
| **Create** | Start new port forward session |
| **Local Port** | Custom local port selection |
| **Container Port** | Select target container port |
| **Status** | Active/Stopped indicator |
| **Stop/Restart** | Manage forwarding sessions |

---

## Settings

Graphical configuration interface.

### LLM Settings

![LLM Settings](../images/webui-settings-llm.png)

| Setting | Description |
|---------|-------------|
| **Provider** | OpenAI, Ollama, Anthropic, Gemini, etc. |
| **Model** | Select model (gpt-4, llama3.2, etc.) |
| **Endpoint** | Custom API endpoint |
| **API Key** | Provider API key |
| **Temperature** | Response creativity (0-1) |

### MCP Servers

![MCP Settings](../images/webui-settings-mcp.png)

Configure Model Context Protocol servers:

- **Add Server** - Configure new MCP server
- **Enable/Disable** - Toggle server activation
- **Arguments** - Command line arguments
- **Environment** - Environment variables

### User Management

![Add New User](../images/webui-settings-new-user.png)

Create and manage user accounts:

| Feature | Description |
|---------|-------------|
| **Add User** | Create new user account |
| **Edit User** | Modify user settings |
| **Delete User** | Remove user account |
| **Role Assignment** | Assign roles (admin, user, viewer) |

### Authentication Control

![Auth Control](../images/webui-settings-admin-user-authentication-controll.png)

Admin controls for user authentication:

- Enable/disable user accounts
- Reset passwords
- Manage session timeouts

---

## Authentication

Multiple authentication options for different environments.

### Login Page

![Login Page](../images/webui-login-page.png)

| Mode | Description |
|------|-------------|
| **Local** | Username/password stored locally |
| **Token** | Kubernetes ServiceAccount token |
| **LDAP** | LDAP/Active Directory integration |
| **SSO** | OAuth2/OIDC integration |
| **No Auth** | Disabled (development only) |

---

## Search & Filtering

Find resources quickly across your cluster.

### Global Search

![Global Search](../images/webui-search-resource.png)

| Feature | Description |
|---------|-------------|
| **Quick Search** | ++ctrl+k++ to open search |
| **Type Filter** | Filter by resource type |
| **Namespace Filter** | Scope to namespace |
| **Status Filter** | Filter by status |
| **Regex Support** | Pattern matching |

---

## Keyboard Shortcuts

Efficient navigation with keyboard.

![Keyboard Shortcuts](../images/webui-keyboard-shortcut.png)

| Shortcut | Action |
|----------|--------|
| ++ctrl+k++ | Open search |
| ++j++ / ++k++ | Navigate up/down |
| ++enter++ | View details |
| ++d++ | Describe resource |
| ++l++ | View logs |
| ++t++ | Open terminal |
| ++esc++ | Close modal |

---

## Architecture Support

Deploy k13d on various platforms.

### Supported Platforms

| Platform | Support |
|----------|---------|
| **Linux amd64** | ✅ Full support |
| **Linux arm64** | ✅ Full support |
| **macOS Intel** | ✅ Full support |
| **macOS Apple Silicon** | ✅ Full support |
| **Windows amd64** | ✅ Full support |

### Kubernetes Support

| Feature | Description |
|---------|-------------|
| **ServiceAccount** | Run with K8s ServiceAccount |
| **RBAC** | Respect K8s RBAC permissions |
| **In-cluster** | Deploy as pod in cluster |
| **Out-of-cluster** | Run locally with kubeconfig |

---

## Docker Compose

Quick local deployment with Docker Compose.

### Compose Configuration

```yaml
version: '3.8'
services:
  k13d:
    image: cloudbro/k13d:latest
    ports:
      - "8080:8080"
    volumes:
      - ~/.kube:/root/.kube:ro
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
```

### With Ollama

```yaml
version: '3.8'
services:
  k13d:
    image: cloudbro/k13d:latest
    ports:
      - "8080:8080"
    environment:
      - LLM_PROVIDER=ollama
      - LLM_ENDPOINT=http://ollama:11434
  ollama:
    image: ollama/ollama:latest
```
