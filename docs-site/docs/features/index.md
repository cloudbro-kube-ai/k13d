# Features Overview

k13d provides comprehensive Kubernetes management with both TUI and Web interfaces, featuring an integrated AI assistant.

## Feature Comparison

| Feature | TUI | Web UI | Description |
|---------|:---:|:------:|-------------|
| **Dashboard** | ✅ | ✅ | Real-time resource overview |
| **AI Assistant** | ✅ | ✅ | Agentic AI with tool calling |
| **Resource Browsing** | ✅ | ✅ | View all K8s resources |
| **YAML Viewer** | ✅ | ✅ | View resource manifests |
| **Log Viewer** | ✅ | ✅ | Real-time log streaming |
| **Terminal/Shell** | ✅ | ✅ | Pod shell access |
| **Port Forward** | ✅ | ✅ | Forward ports locally |
| **Topology View** | ❌ | ✅ | Resource relationship graph |
| **Reports** | ❌ | ✅ | PDF/CSV cluster reports |
| **Metrics Charts** | ❌ | ✅ | Visual CPU/Memory graphs |
| **Settings UI** | ❌ | ✅ | Graphical configuration |
| **Multi-user** | ❌ | ✅ | Authentication & RBAC |
| **Audit Logging** | ✅ | ✅ | Track all operations |
| **i18n** | ✅ | ✅ | Multi-language support |
| **Vim Navigation** | ✅ | ❌ | h/j/k/l keybindings |

## Quick Links

<div class="grid cards" markdown>

-   :material-view-dashboard:{ .lg .middle } **[Web UI Features](web-ui.md)**

    ---

    Dashboard, AI Assistant, Reports, Metrics, and more

-   :material-console:{ .lg .middle } **[TUI Features](tui.md)**

    ---

    Terminal dashboard with Vim-style navigation

-   :material-robot:{ .lg .middle } **[AI Assistant](ai-assistant.md)**

    ---

    Agentic AI with kubectl/bash tool calling

-   :material-shield-lock:{ .lg .middle } **[Security Features](security.md)**

    ---

    Authentication, RBAC, Audit logging

</div>

---

## Interface Screenshots

### Web UI Dashboard

![Web UI Full Screen](../images/webui-full-screen.png)

*Main dashboard showing resources, AI assistant panel, and navigation*

### Web UI - AI Assistant

![Web UI AI Assistant](../images/webui-assistant-pannel.png)

*AI assistant panel with natural language queries and tool execution*

### Web UI - Topology View

![Topology View](../images/webui-topology-view-all.png)

*Interactive resource relationship visualization*

---

### TUI Dashboard

![TUI Full Screen](../images/tui-full-screen.png)

*Terminal dashboard with k9s-style interface*

### TUI - AI Assistant

![TUI AI Panel](../images/tui-assistant-pannel.png)

*AI assistant integrated in TUI*

### TUI - AI Conversation

![TUI AI Conversation](../images/tui-ask-answer-test.png)

*Example AI interaction with tool execution*
