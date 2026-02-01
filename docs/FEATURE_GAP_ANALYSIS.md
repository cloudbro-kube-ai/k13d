# k13d Feature Gap Analysis

## Overview

This document compares k13d with three major Kubernetes dashboard/AI projects to identify missing features.

**Analyzed Projects:**
- **kubernetes-dashboard** - Official Kubernetes Dashboard
- **headlamp** - Cloud-native Kubernetes UI with plugin system
- **kubectl-ai** - AI-powered kubectl assistant

**Last Updated:** 2026-01-22

---

## Current k13d Features (Baseline)

### CLI/TUI Features
- [x] Resource listing: Pods, Deployments, Services, Nodes, Namespaces, Events, ConfigMaps, Secrets, DaemonSets, StatefulSets, Jobs, CronJobs, Ingresses
- [x] Namespace switching
- [x] Context switching
- [x] Real-time table filtering with highlight
- [x] Command autocomplete with hints
- [x] AI chat integration (streaming)
- [x] YAML view for resources
- [x] Pod logs viewer
- [x] Delete with confirmation
- [x] Port forwarding
- [x] Shell exec (spawns external terminal)
- [x] Health check command
- [x] i18n support (en, ko, ja, zh)

### Web UI Features (index.html)
- [x] Login page with username/password
- [x] Dark theme (Tokyo Night style)
- [x] Sidebar navigation with resource groups
- [x] Resource table with status colors
- [x] Namespace selector dropdown
- [x] AI chat panel with streaming responses
- [x] Resizable panels (drag handle)
- [x] Settings modal (language, LLM config)
- [x] Audit log viewer
- [x] Reports viewer
- [x] Refresh button per resource
- [x] User badge and logout
- [x] Tab system (Table/YAML/Logs)
- [x] Message history in AI panel
- [x] Tool execution visualization with Show More button

### Web Server/API Features
- [x] REST API for resources (pods, deployments, services, namespaces, nodes, events)
- [x] AI chat endpoint (SSE streaming with agentic mode)
- [x] JWT authentication with session management
- [x] LDAP authentication support
- [x] Audit logging (SQLite)
- [x] Report generation API
- [x] Settings API (GET/PUT)
- [x] LLM settings API
- [x] Health check endpoint
- [x] CORS middleware
- [x] Static file serving (embedded)
- [x] Tool approval flow for write/dangerous commands
- [x] Command classification (read-only, write, dangerous)

### Backend Features
- [x] Multi-provider LLM support (OpenAI-compatible, Ollama, Anthropic)
- [x] Kubernetes client with metrics API
- [x] Dynamic client for CRDs
- [x] Context switching support
- [x] MCP (Model Context Protocol) integration
- [x] Tool registry with MCP tool support

---

## Recently Implemented Features (2026-01)

### AI Safety & Validation ✅
**Status:** Implemented
- [x] Command classification (read-only, write, dangerous)
- [x] Tool approval flow for destructive operations
- [x] Auto-approve read-only commands
- [x] User confirmation required for write/dangerous commands
- [x] Timeout handling for approval requests

### In-Browser Terminal ✅
**Status:** Implemented
- [x] WebSocket-based terminal (`pkg/web/terminal.go`)
- [x] Bidirectional stdin/stdout streaming
- [x] Terminal resize support
- [x] Container selection support

### Deployment Operations ✅
**Status:** Implemented (`pkg/web/operations.go`)
- [x] Scale deployments
- [x] Rollout restart
- [x] Pause/resume deployment
- [x] Rollback to previous revision
- [x] View revision history

### StatefulSet/DaemonSet Operations ✅
**Status:** Implemented
- [x] Scale StatefulSets
- [x] Restart StatefulSets
- [x] Restart DaemonSets

### CronJob Operations ✅
**Status:** Implemented
- [x] Trigger CronJob manually (create Job)
- [x] Suspend/resume CronJob

### Node Operations ✅
**Status:** Implemented
- [x] Node cordon/uncordon
- [x] Node drain with eviction
- [x] List pods on node

### Helm Integration ✅
**Status:** Implemented (`pkg/helm/client.go`)
- [x] List Helm releases
- [x] Get release details
- [x] View release history
- [x] Get release values
- [x] Get release manifest
- [x] Install releases
- [x] Upgrade releases
- [x] Uninstall releases
- [x] Rollback releases
- [x] Repository management (add, remove, update)
- [x] Search charts

### MCP Support ✅
**Status:** Implemented
- [x] MCP client mode (use external tools)
- [x] Tool registration system
- [x] MCP server connection management
- [x] Dynamic tool discovery

---

## Priority 1: Critical Missing Features

### 1.1 Multi-Provider LLM Support (from kubectl-ai)
**Status:** ✅ Implemented (2026-01-22)
- [x] OpenAI-compatible provider (with tool calling)
- [x] Ollama (local) provider
- [x] Model profile management
- [x] Google Gemini provider
- [x] AWS Bedrock provider (Claude) - with tool calling
- [x] Azure OpenAI provider - with tool calling
- [ ] Llama.cpp provider (low priority)
- [ ] Grok provider (low priority)

### 1.2 Plugin System (from headlamp)
**Status:** Not Implemented
**Required:**
- [ ] Plugin registry system
- [ ] Dynamic plugin loading from URL
- [ ] Plugin configuration management
- [ ] Extension points:
  - Custom sidebar items
  - Custom routes
  - Custom resource detail views
  - Custom table columns
  - Custom themes

**Files to create:**
```
pkg/plugins/
├── registry.go      # Plugin registry
├── loader.go        # Dynamic plugin loading
├── config.go        # Plugin configuration
└── types.go         # Plugin interfaces
```

---

## Priority 2: Important Missing Features

### 2.1 Session Persistence (from kubectl-ai)
**Status:** ✅ Implemented (2026-01-22)
**Location:** `pkg/ai/session/session.go`
- [x] Session storage interface (`Store`)
- [x] Filesystem backend (JSON files in XDG data directory)
- [x] Session metadata (ID, timestamps, model, provider)
- [x] Resume/load previous sessions (`Get`, `GetContextMessages`)
- [x] Clear conversation command (`Clear`, `Delete`)
- [x] List sessions with pagination (`List`, `GetRecentSessions`)
- [x] Export/Import sessions
- [x] Tool call recording in messages

### 2.2 Resource Graph Visualization (from headlamp)
**Status:** Not Implemented
**Required:**
- [ ] Pod-to-Service-to-Deployment graph
- [ ] Resource relationship mapping
- [ ] Interactive graph navigation
- [ ] Namespace grouping

**Implementation:** Use @xyflow/react or similar for web UI

### 2.3 Metrics Visualization (from kubernetes-dashboard)
**Status:** API Available, UI Not Implemented
**Required:**
- [ ] CPU/Memory sparklines in table
- [ ] Resource usage charts
- [ ] Historical metrics storage
- [ ] Metrics aggregation

---

## Priority 3: Enhancement Features

### 3.1 OIDC/OAuth2 Authentication (from headlamp)
**Status:** Not Implemented
**Required:**
- [ ] OIDC provider integration
- [ ] PKCE flow support
- [ ] Token refresh automation
- [ ] JMESPath claim extraction
- [ ] Multiple auth method support

### 3.2 Multi-Cluster Support (from headlamp)
**Status:** Not Implemented
**Required:**
- [ ] Cluster switcher UI
- [ ] Per-cluster authentication
- [ ] Kubeconfig file watcher
- [ ] Dynamic cluster addition

### 3.3 Advanced Search (from headlamp)
**Status:** Not Implemented
**Required:**
- [ ] Global search across clusters
- [ ] Label selector filtering
- [ ] Field selector filtering
- [ ] Recent searches tracking
- [ ] Full-text search with fuse.js

### 3.4 Custom Resource Definition Support (from kubernetes-dashboard, headlamp)
**Status:** Partial (API exists)
**Required:**
- [ ] List all CRDs in UI
- [ ] Browse CRD instances
- [ ] Create/edit custom resources
- [ ] CRD schema validation
- [ ] OpenAPI documentation display

### 3.5 Retry Logic with Backoff (from kubectl-ai)
**Status:** ✅ Implemented
**Location:** `pkg/ai/providers/factory.go`
- [x] Exponential backoff
- [x] Jitter support
- [x] Retryable error detection (429, 5xx, timeout, connection errors)
- [x] Max attempts configuration
- [x] Configurable via `RetryConfig`

### 3.6 YAML Editor with Validation (from headlamp)
**Status:** Not Implemented
**Required:**
- [ ] Monaco editor integration
- [ ] Syntax highlighting
- [ ] Schema validation
- [ ] Auto-completion
- [ ] Inline documentation

---

## Priority 4: UI/UX Improvements

### 4.1 From kubernetes-dashboard
- [ ] Deployment creation wizard (form-based)
- [ ] Image reference validation
- [ ] Protocol validation for services
- [ ] Log download as file
- [ ] Multi-container log selection
- [ ] Sparkline metrics in list views
- [ ] CSRF protection for API

### 4.2 From headlamp
- [ ] Dark/Light theme toggle
- [ ] Custom theme support
- [ ] Responsive mobile layout
- [ ] Breadcrumb navigation
- [ ] Action confirmation dialogs
- [ ] Empty state handling
- [ ] Loading animations
- [ ] Error boundary UI
- [ ] Internationalization (12 languages)

### 4.3 From kubectl-ai
- [x] Streaming log UI for AI responses
- [ ] Meta-commands (clear, model, tools, sessions)
- [x] Tool call visualization
- [x] Audit/journal logging with structured output

---

## API Feature Gaps (Backend)

### From kubernetes-dashboard API
| Endpoint | Description | k13d Status |
|----------|-------------|-------------|
| `POST /appdeployment` | Deploy application | Missing |
| `POST /appdeployment/validate/*` | Validation endpoints | Missing |
| `PUT /deployment/rollback` | Rollback deployment | ✅ Implemented |
| `PUT /deployment/pause` | Pause deployment | ✅ Implemented |
| `PUT /deployment/resume` | Resume deployment | ✅ Implemented |
| `PUT /deployment/restart` | Restart deployment | ✅ Implemented |
| `POST /cronjob/trigger` | Trigger CronJob | ✅ Implemented |
| `POST /node/drain` | Drain node | ✅ Implemented |
| `GET /metrics/*` | Metrics endpoints | ✅ Implemented |
| `GET /_raw/*` | Raw resource access | Missing |
| `GET /csrftoken/*` | CSRF tokens | Missing |
| `WS /shell/*` | WebSocket terminal | ✅ Implemented |
| `GET /log/file/*` | Download logs | Missing |

### From headlamp API
| Feature | Description | k13d Status |
|---------|-------------|-------------|
| `/plugins/*` | Plugin management | Missing |
| `/helm/*` | Helm operations | ✅ Implemented |
| `WS /exec` | Pod exec WebSocket | ✅ Implemented |
| `/portforward/*` | Port forward management | ✅ Implemented |
| `/cluster/*` | Multi-cluster management | Missing |

---

## Summary

| Category | kubernetes-dashboard | headlamp | kubectl-ai | k13d Status |
|----------|---------------------|----------|------------|-------------|
| LLM Providers | N/A | N/A | 7+ providers | ✅ 5 providers (OpenAI, Ollama, Gemini, Bedrock, Azure) |
| AI Safety | N/A | N/A | Full | ✅ Implemented |
| Terminal | WebSocket | WebSocket | N/A | ✅ Implemented |
| Plugins | No | Yes (full) | No | Not Implemented |
| Helm | No | Yes | No | ✅ Implemented |
| Multi-cluster | No | Yes | No | Not Implemented |
| Metrics Viz | Yes (sparkline) | Yes | No | API only |
| Resource Graph | No | Yes | No | Not Implemented |
| Session Persist | N/A | No | Yes | ✅ Implemented |
| Retry/Backoff | N/A | N/A | Yes | ✅ Implemented |
| OIDC Auth | Yes | Yes (full) | No | Not Implemented |
| CRD Management | Yes | Yes | No | Partial |
| Node Operations | Yes | Yes | No | ✅ Implemented |
| Deployment Ops | Yes (full) | Partial | No | ✅ Implemented |
| MCP Support | N/A | N/A | Yes | ✅ Implemented |

**Implemented Features:** ~30 major features
**Remaining Features:** ~15 major features

---

## Implementation Progress

### Completed (Phase 1-2)
1. ✅ AI safety validation with tool approval flow
2. ✅ In-browser terminal (WebSocket)
3. ✅ Deployment operations (scale, restart, pause, resume, rollback, history)
4. ✅ StatefulSet/DaemonSet operations
5. ✅ CronJob operations (trigger, suspend)
6. ✅ Node operations (cordon, drain, pods)
7. ✅ Helm integration (releases, repos, search)
8. ✅ MCP support
9. ✅ Tool execution visualization

### Recently Completed (2026-01-22)
- [x] Google Gemini provider
- [x] AWS Bedrock provider (Claude) with tool calling
- [x] Azure OpenAI provider with tool calling
- [x] Session persistence for AI conversations
- [x] Retry logic with exponential backoff

### Upcoming (Phase 3-5)
1. Metrics visualization (sparklines, charts)
2. Resource graph visualization
3. Plugin system
4. Multi-cluster support
5. OIDC authentication
6. Advanced search
7. YAML editor with Monaco
8. Web UI integration for session management
