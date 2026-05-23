# Architecture

k13d is designed as a single-binary application that combines a TUI dashboard, web interface, and AI-powered assistant for Kubernetes management.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         k13d Binary                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │   TUI Mode   │    │   Web Mode   │    │  CLI Mode    │       │
│  │   (tview)    │    │   (HTTP)     │    │  (direct)    │       │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘       │
│         │                   │                    │               │
│         └───────────────────┼────────────────────┘               │
│                             │                                    │
│                    ┌────────▼────────┐                          │
│                    │   Shared Core    │                          │
│                    ├──────────────────┤                          │
│                    │ • AI Agent       │                          │
│                    │ • K8s Client     │                          │
│                    │ • Tool Registry  │                          │
│                    │ • Safety Analyzer│                          │
│                    │ • Session Store  │                          │
│                    │ • Audit Logger   │                          │
│                    │ • Issue Automation│                         │
│                    └────────┬─────────┘                          │
│                             │                                    │
│         ┌───────────────────┼───────────────────┐               │
│         │                   │                   │               │
│  ┌──────▼──────┐   ┌───────▼───────┐   ┌───────▼──────┐        │
│  │ LLM Provider│   │ Kubernetes API│   │ SQLite (Audit)│        │
│  │ (OpenAI,    │   │   (client-go) │   │              │        │
│  │  Ollama, ..)│   │               │   │              │        │
│  └─────────────┘   └───────────────┘   └──────────────┘        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## System Requirements

| Component | Required | Description |
|-----------|----------|-------------|
| **k13d binary** | Yes | Single executable binary |
| **Kubernetes Cluster** | Yes | kubeconfig required (~/.kube/config) |
| **LLM Provider** | Optional | Required for AI features (OpenAI, Ollama, etc.) |
| **SQLite** | Auto-created | Built-in audit logging (CGO-free) |
| **External RDB** | No | No external database required |

## Module Structure

### Core Packages

| Package | Path | Purpose |
|---------|------|---------|
| **ui** | `pkg/ui/` | TUI components (tview-based) |
| **web** | `pkg/web/` | Web server & API handlers |
| **ai/agent** | `pkg/ai/agent/` | AI agent state machine |
| **ai/providers** | `pkg/ai/providers/` | LLM provider implementations |
| **ai/safety** | `pkg/ai/safety/` | Command safety analysis |
| **ai/tools** | `pkg/ai/tools/` | Tool registry & execution |
| **ai/sessions** | `pkg/ai/sessions/` | Conversation session management |
| **automation** | `pkg/automation/` | GitHub issue webhook queue, worktree execution, PR/reporting |
| **k8s** | `pkg/k8s/` | Kubernetes client wrapper |
| **db** | `pkg/db/` | SQLite audit logging |
| **config** | `pkg/config/` | Configuration management (config, aliases, views, hotkeys, plugins) |
| **i18n** | `pkg/i18n/` | Internationalization |

## AI Agent State Machine

The AI Agent operates as a state machine managing the conversation flow:

```
    ┌─────────┐
    │  Idle   │◄────────────────────────┐
    └────┬────┘                         │
         │ User Message                 │
         ▼                              │
    ┌─────────┐                         │
    │ Running │◄─────────────────┐      │
    └────┬────┘                  │      │
         │ LLM Response          │      │
         ▼                       │      │
    ┌──────────────┐             │      │
    │ToolAnalysis  │             │      │
    └────┬─────────┘             │      │
         │                       │      │
         ├─ Auto-approve ────────┘      │
         │  (read-only)                 │
         ▼                              │
    ┌──────────────────┐                │
    │WaitingForApproval│                │
    └────┬─────────────┘                │
         │                              │
         ├─ Approved ──► Execute ───────┤
         ├─ Rejected ───────────────────┤
         └─ Timeout ────────────────────┤
                                        │
    ┌─────────┐                         │
    │  Done   │─────────────────────────┤
    └─────────┘                         │
                                        │
    ┌─────────┐                         │
    │  Error  │─────────────────────────┘
    └─────────┘
```

## Safety Analysis

All commands pass through the safety analyzer before execution:

```
User Command
     │
     ▼
┌─────────────────┐
│  Shell Parser   │  mvdan.cc/sh/v3
│  (AST Parsing)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Safety Analyzer │
└────────┬────────┘
         │
         ├── ReadOnly? ──► Decision Required by default
         │                (auto-approve configurable)
         │
         ├── Write? ──► Require approval
         │
         ├── Dangerous? ──► Warning + Require approval
         │                  (delete ns, rm -rf, etc.)
         │
         └── Unsupported interactive / blocked pattern?
                          ──► Hard block (not approvable)
```

### Command Classification

| Type | Examples | Approval |
|------|----------|----------|
| **Read-only** | get, describe, logs | Decision Required by default, auto-approve optional |
| **Write** | apply, create, patch | Requires approval |
| **Dangerous** | delete, drain, taint | Warning + approval, or full block if configured |
| **Hard-blocked** | `kubectl edit`, `kubectl port-forward`, `kubectl exec -it`, blocked regex matches | Blocked immediately |

## Data Storage

### SQLite (Audit Log)

Location: `<XDG config home>/k13d/audit.db`

On macOS, the default path is `~/Library/Application Support/k13d/audit.db`.

```sql
CREATE TABLE audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    user TEXT,
    action TEXT,           -- query, approve, reject, execute
    resource TEXT,         -- pod/nginx, deployment/app
    details TEXT,          -- JSON details
    llm_request TEXT,      -- LLM request (optional)
    llm_response TEXT      -- LLM response (optional)
);
```

### Session Storage

- **Memory Store**: Default, deleted on process exit
- **Filesystem Store**: platform data directory, for example `~/.local/share/k13d/sessions/` on Linux or `~/Library/Application Support/k13d/sessions/` on macOS

## Web UI Architecture

### HTTP API Endpoints

```
/                           # Static page (embedded)
/api/health                 # Health check

# Authentication
/api/auth/login             # Login
/api/auth/logout            # Logout

# AI Chat (SSE)
/api/chat/agentic           # SSE streaming chat
/api/tool/approve           # Tool approve/reject

# GitHub Issue Automation
/api/github/automation/webhook          # Public GitHub issues webhook
/api/admin/github-automation/status     # Admin status + recent jobs
/api/admin/github-automation/jobs       # Admin jobs summary
/api/admin/github-automation/jobs/{id}  # Admin single-job details
/previews/{branch-slug}/...             # Branch preview reverse proxy

# Kubernetes Resources
/api/k8s/pods               # Pod list
/api/k8s/deployments        # Deployment list
/api/k8s/services           # Service list
/api/k8s/{resource}         # Other resources

# Operations
/api/deployment/scale       # Scale
/api/deployment/restart     # Restart
/api/node/cordon            # Node Cordon
/api/portforward            # Port Forwarding
```

### SSE Event Flow

```
Browser                          Server
   │                               │
   │ POST /api/chat/agentic        │
   │──────────────────────────────►│
   │                               │
   │◄── SSE: event: chunk ─────────│
   │◄── SSE: event: chunk ─────────│
   │                               │
   │◄── SSE: event: tool_request ──│ (approval needed)
   │                               │
   │ POST /api/tool/approve        │
   │──────────────────────────────►│
   │                               │
   │◄── SSE: event: tool_execution │
   │◄── SSE: event: chunk ─────────│
   │◄── SSE: event: stream_end ────│
   │                               │
```

## GitHub Issue Automation Flow

The issue automation path is intentionally local-first. k13d receives a GitHub issue webhook, validates it, and then runs the configured agent commands inside an isolated git worktree.

```
GitHub Issues Webhook
        │
        ▼
┌──────────────────────────┐
│ /api/github/automation/  │
│ webhook                  │
└────────────┬─────────────┘
             │ verify signature
             ▼
┌──────────────────────────┐
│ automation.Manager       │
│ - label gate             │
│ - repo allow-list        │
│ - active job dedupe      │
│ - worker queue           │
└────────────┬─────────────┘
             │
             ▼
┌──────────────────────────┐
│ automation.Executor      │
│ - create worktree        │
│ - checkout issue branch  │
│ - run development cmd    │
│ - commit / push          │
└────────────┬─────────────┘
             │
             ▼
┌──────────────────────────┐
│ GitHub REST integration  │
│ - draft PR               │
│ - wait for check runs    │
│ - PR review              │
│ - issue comment          │
└────────────┬─────────────┘
             │
             ▼
┌──────────────────────────┐
│ Preview deploy           │
│ - deploy command output  │
│ - preview target registry│
│ - /previews/<branch>/    │
└──────────────────────────┘
```

Important characteristics:

- commands are fully configurable in `config.yaml`
- each issue runs in its own worktree under `worktree_root`
- automation is off by default
- without a GitHub token, local execution still works, but PR/comments/reviews are skipped
- the webhook route is public by design, so the shared secret and allowed repository list are both important
- branch previews use path-based reverse proxying so one public domain can expose many local branch instances

## Next Steps

- [AI Assistant](ai-assistant.md) - Learn about AI capabilities
- [MCP Integration](mcp-integration.md) - Extend with custom tools
- [Security & RBAC](security.md) - Security features
