# TUI (Terminal User Interface) Architecture Guide

This document explains the internal architecture of k13d's Terminal User Interface, built with [tview](https://github.com/rivo/tview) and [tcell](https://github.com/gdamore/tcell).

## Table of Contents

- [Overview](#overview)
- [Main Application Structure](#main-application-structure)
- [Component Hierarchy](#component-hierarchy)
- [Keyboard Navigation](#keyboard-navigation)
- [AI Assistant Integration](#ai-assistant-integration)
- [Resource Views](#resource-views)
- [State Management](#state-management)
- [Concurrency Patterns](#concurrency-patterns)
- [File Structure](#file-structure)

---

## Overview

k13d's TUI follows the **k9s** design patterns, providing:
- Vim-style navigation (`j/k/g/G`)
- Command mode (`:pods`, `:deploy`)
- Live filtering (`/pattern/`)
- Resource drill-down with back navigation
- Integrated AI assistant panel

---

## Main Application Structure

### Core App Struct

**File: `pkg/ui/app.go`**

```go
type App struct {
    *tview.Application

    // Core dependencies
    config   *config.Config
    k8s      *k8s.Client
    aiClient *ai.Client

    // UI Components
    pages       *tview.Pages        // Page container
    header      *tview.TextView     // Top bar with cluster info
    briefing    *BriefingPanel      // Cluster health panel
    table       *tview.Table        // Main resource table
    statusBar   *tview.TextView     // Bottom status bar
    flash       *tview.TextView     // Flash messages
    cmdInput    *tview.InputField   // Command input (:)
    cmdHint     *tview.TextView     // Autocomplete hints
    cmdDropdown *tview.List         // Command suggestions
    aiPanel     *tview.TextView     // AI response area
    aiInput     *tview.InputField   // AI question input

    // State (protected by mutex)
    mx                  sync.RWMutex
    currentResource     string
    currentNamespace    string
    namespaces          []string
    showAIPanel         bool
    filterText          string
    tableHeaders        []string
    tableRows           [][]string
    selectedRows        map[int]bool
    sortColumn          int
    sortAscending       bool

    // Navigation history (stack)
    navMx           sync.Mutex
    navigationStack []navHistory

    // Atomic guards (k9s pattern)
    inUpdate    int32
    running     int32
    stopping    int32
    hasToolCall int32
    cancelFn    context.CancelFunc

    // AI tool approval
    aiMx                sync.RWMutex
    pendingDecisions    []PendingDecision
    pendingToolApproval chan bool
}
```

### Key Design Patterns

| Pattern | Purpose | Usage |
|---------|---------|-------|
| **Atomic Guards** | Lock-free checks for hot paths | `inUpdate`, `running`, `hasToolCall` |
| **RWMutex** | Read-heavy state protection | `currentResource`, `namespaces` |
| **Navigation Stack** | Back button support | `navigationStack` |
| **QueueUpdateDraw** | Thread-safe UI updates | All goroutine→UI communication |
| **Panic Recovery** | Graceful error handling | `Run()` method |

---

## Component Hierarchy

```
App (tview.Application)
│
└── pages (tview.Pages)
    │
    └── "main" (mainFlex - tview.Flex vertical)
        │
        ├── header (tview.TextView) ─────────────────────── 4 lines
        │   ├── Logo + Tagline + Version
        │   ├── Context: xxx | Cluster: xxx | Namespace: xxx
        │   ├── Resource: pods (25 items)
        │   └── [1]ns1 [2]ns2 [3]ns3 ... (quick-select preview)
        │
        ├── flash (tview.TextView) ──────────────────────── 1 line
        │   └── Flash messages (errors, success)
        │
        ├── briefing (BriefingPanel) ────────────────────── 5 lines (optional)
        │   ├── Health Score: 95/100 [████████░░]
        │   ├── Pods: 45 running, 2 pending, 1 failed
        │   ├── Nodes: 3/3 ready
        │   ├── CPU: 45% | Memory: 62%
        │   └── ⚠ 1 pod in CrashLoopBackOff
        │
        ├── contentFlex (tview.Flex horizontal)
        │   │
        │   ├── table (tview.Table) ─────────────────────── Main view
        │   │   ├── [Header] NAMESPACE | NAME | STATUS | READY | AGE
        │   │   ├── [Row 1]  default   | nginx | Running | 1/1  | 5m
        │   │   ├── [Row 2]  default   | redis | Running | 1/1  | 3m
        │   │   └── [Row N]  ...
        │   │
        │   └── aiContainer (tview.Flex vertical) ───────── AI Panel
        │       ├── aiPanel (tview.TextView) ────────────── AI responses
        │       │   ├── "[cyan]🤖 Agentic Mode"
        │       │   ├── "[yellow]Question: why is pod failing?"
        │       │   ├── "Analyzing pod status..."
        │       │   └── "🔧 kubectl get pods -n default"
        │       │
        │       └── aiInput (tview.InputField) ──────────── AI input
        │           └── "Ask AI: _"
        │
        ├── statusBar (tview.TextView) ──────────────────── 1 line
        │   └── "[l]ogs [d]escribe [y]aml [s]hell [S]cale [R]estart"
        │
        └── cmdFlex (tview.Flex horizontal)
            ├── cmdInput (tview.InputField) ─────────────── Command input
            │   └── ": pods_"
            │
            ├── cmdHint (tview.TextView) ────────────────── Autocomplete
            │   └── "pods, deploy, svc, nodes..."
            │
            └── cmdDropdown (tview.List) ────────────────── Hidden dropdown
```

### Focus Flow

```
┌─────────────────────────────────────────────────────┐
│                                                     │
│  ┌─────────────┐    Tab    ┌─────────────┐         │
│  │   Table     │ ────────► │  AI Input   │         │
│  │  (default)  │ ◄──────── │             │         │
│  └─────────────┘    Esc    └─────────────┘         │
│        │                          │                │
│        │ :                        │ Enter          │
│        ▼                          ▼                │
│  ┌─────────────┐           ┌─────────────┐         │
│  │ Cmd Input   │           │  AI Panel   │         │
│  │  (:pods)    │           │ (responses) │         │
│  └─────────────┘           └─────────────┘         │
│        │                          │                │
│        │ Enter/Esc               │ Y/N (approval) │
│        ▼                          ▼                │
│     Back to Table           Execute/Cancel         │
│                                                     │
└─────────────────────────────────────────────────────┘
```

---

## Keyboard Navigation

### Global Keybindings

| Key | Action | Description |
|-----|--------|-------------|
| `j` / `↓` | Move down | Select next row |
| `k` / `↑` | Move up | Select previous row |
| `g` | Go to top | Select first row |
| `G` | Go to bottom | Select last row |
| `Ctrl+D` / `Ctrl+F` | Page down | Move 10 rows down |
| `Ctrl+U` / `Ctrl+B` | Page up | Move 10 rows up |
| `Enter` | Drill down | Navigate to related resource |
| `Esc` | Go back | Pop navigation stack |
| `Tab` | Toggle focus | Switch between table and AI panel |
| `:` | Command mode | Enter resource command (`:pods`) |
| `/` | Filter mode | Start live filtering |
| `?` | Help | Show help dialog |
| `q` | Quit | Exit application |
| `r` | Refresh | Refresh current view |

### Resource-Specific Actions

| Key | Resource | Action |
|-----|----------|--------|
| `l` | Pods | View logs |
| `p` | Pods | View previous logs |
| `s` | Pods | Shell into pod |
| `a` | Pods | Attach to container |
| `k` / `Ctrl+K` | Pods | Kill pod |
| `o` | Pods | Show node |
| `S` | Deployments/StatefulSets | Scale replicas |
| `R` | Deployments/StatefulSets | Rollout restart |
| `t` | CronJobs | Trigger job |
| `u` | Namespaces | Use namespace |
| `d` | All | Describe resource |
| `y` | All | View YAML |
| `e` | All | Edit resource |
| `F` | All | Port forward |
| `Ctrl+D` | All | Delete (with confirmation) |

### Namespace Quick-Select

| Key | Action |
|-----|--------|
| `0` | All namespaces |
| `1-9` | Select namespace by index |
| `n` | Cycle to next namespace |

### Sorting (Shift + Column Key)

| Key | Sort By |
|-----|---------|
| `N` | Name |
| `A` | Age |
| `T` | Status |
| `P` | Namespace |
| `C` | Restarts |
| `D` | Ready |

### AI Panel Keybindings

| Key | Action |
|-----|--------|
| `Tab` | Switch to AI input |
| `Enter` | Submit question / Approve tool |
| `Esc` | Cancel / Return to table |
| `Y` | Approve tool execution |
| `N` | Reject tool execution |
| `1-9` | Execute specific pending decision |
| `A` | Execute all pending decisions |

---

## AI Assistant Integration

### AIPanel Component

**File: `pkg/ui/ai_panel.go`**

```go
type AIPanel struct {
    *tview.Flex
    outputView *tview.TextView      // AI responses
    inputField *tview.InputField    // User input
    statusBar  *tview.TextView      // Status indicator

    agent *agent.Agent

    isShowingApproval bool
    currentApproval   *agent.ChoiceRequest
    autoScroll        bool

    onSubmit func(string)
    onFocus  func()
}
```

### AI Interaction Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Step 1: User Question                                                    │
│                                                                          │
│   User types: "Why is nginx pod failing?"                               │
│   → Collect context (namespace, resource, selected row)                 │
│   → Build prompt with Kubernetes context                                │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ Step 2: AI Processing                                                    │
│                                                                          │
│   AI Panel shows:                                                        │
│   ┌─────────────────────────────────────────────────────────┐           │
│   │ [cyan]🤖 Agentic Mode[white]                            │           │
│   │ [yellow]Question:[white] Why is nginx pod failing?      │           │
│   │ [gray]Thinking...                                       │           │
│   └─────────────────────────────────────────────────────────┘           │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ Step 3: Tool Call (if needed)                                            │
│                                                                          │
│   AI decides to run: kubectl describe pod nginx -n default              │
│   ┌─────────────────────────────────────────────────────────┐           │
│   │ ━━━ DECISION REQUIRED ━━━                               │           │
│   │                                                          │           │
│   │ ? [1] Confirm: kubectl describe pod nginx -n default    │           │
│   │                                                          │           │
│   │ Press Y to approve, N to cancel, 1-9 for specific       │           │
│   └─────────────────────────────────────────────────────────┘           │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ Step 4: Execution & Response                                             │
│                                                                          │
│   After approval (Y/Enter):                                              │
│   ┌─────────────────────────────────────────────────────────┐           │
│   │ 🔧 Executing: kubectl describe pod nginx -n default     │           │
│   │                                                          │           │
│   │ [green]✓ Command executed successfully[white]           │           │
│   │                                                          │           │
│   │ Based on the output, the pod is failing because:        │           │
│   │ - Image pull error: nginx:invalid-tag not found         │           │
│   │ - Suggested fix: Update image to nginx:latest           │           │
│   └─────────────────────────────────────────────────────────┘           │
└─────────────────────────────────────────────────────────────────────────┘
```

### Safety Analysis

All AI-suggested commands pass through the safety analyzer:

| Command Type | Examples | Behavior |
|--------------|----------|----------|
| **Read-only** | `get`, `describe`, `logs` | Auto-approve (configurable) |
| **Write** | `apply`, `create`, `patch` | Require confirmation |
| **Dangerous** | `delete`, `drain`, `cordon` | Extra warnings shown |
| **Interactive** | `exec -it`, `attach` | Not auto-executed |

---

## Resource Views

### Supported Resources (30+)

**Workloads:**
- `pods` (po) - Pod list with status, ready, restarts
- `deployments` (deploy) - Deployment list with replicas
- `daemonsets` (ds) - DaemonSet list
- `statefulsets` (sts) - StatefulSet list
- `jobs` (job) - Job list with completions
- `cronjobs` (cj) - CronJob list with schedule
- `replicasets` (rs) - ReplicaSet list

**Services & Networking:**
- `services` (svc) - Service list with type, ports
- `ingresses` (ing) - Ingress list with hosts
- `endpoints` (ep) - Endpoint list
- `networkpolicies` (netpol) - NetworkPolicy list

**Config & Storage:**
- `configmaps` (cm) - ConfigMap list
- `secrets` (sec) - Secret list with type
- `persistentvolumes` (pv) - PV list with capacity
- `persistentvolumeclaims` (pvc) - PVC list with status
- `storageclasses` (sc) - StorageClass list

**RBAC:**
- `serviceaccounts` (sa) - ServiceAccount list
- `roles` (role) - Role list
- `rolebindings` (rb) - RoleBinding list
- `clusterroles` (cr) - ClusterRole list
- `clusterrolebindings` (crb) - ClusterRoleBinding list

**Cluster:**
- `nodes` (no) - Node list with status, version
- `namespaces` (ns) - Namespace list
- `events` (ev) - Event list
- `customresourcedefinitions` (crd) - CRD list

### Fetch Flow

```
handleCommand(":pods")
        │
        ▼
setResource("pods")
        │
        ▼
navigateTo("pods", namespace, "")
        │
        ├── updateHeader()      ─── Update header with resource info
        ├── updateStatusBar()   ─── Update keybinding hints
        └── refresh()           ─── Async fetch and render
                │
                ▼
        fetchResources(ctx)
                │
                ▼
        fetchPods(ctx, ns)
                │
                ├── k8s.ListPods()
                ├── Format headers and rows
                └── Store in tableHeaders, tableRows
                        │
                        ▼
        applyFilterText()      ─── Apply live filter if active
                │
                ▼
        QueueUpdateDraw()      ─── Thread-safe UI update
                │
                ▼
        table.SetCell()        ─── Render cells with colors
```

### Drill-Down Navigation

When you press `Enter` on a resource, k13d navigates to related resources:

| From | Navigate To |
|------|-------------|
| Service | Pods (matching selector) |
| Deployment | Pods |
| ReplicaSet | Pods |
| StatefulSet | Pods |
| DaemonSet | Pods |
| Job | Pods |
| CronJob | Jobs |
| Node | Pods on node |
| Namespace | Switch & show Pods |
| Pod | Logs view |

Press `Esc` to go back (navigation history is maintained).

---

## State Management

### Thread-Safe State Updates

```go
// Pattern 1: Read-only access (common case)
a.mx.RLock()
resource := a.currentResource
a.mx.RUnlock()

// Pattern 2: Modify state
a.mx.Lock()
a.currentResource = "pods"
a.mx.Unlock()

// Pattern 3: Atomic lock-free checks (hot paths)
if atomic.LoadInt32(&a.hasToolCall) == 1 {
    a.approveToolCall(true)
}

// Pattern 4: QueueUpdateDraw for all UI updates from goroutines
go func() {
    // Long operation...
    a.QueueUpdateDraw(func() {
        a.table.SetCell(row, col, cell)
    })
}()
```

### Navigation Stack

```go
type navHistory struct {
    resource  string
    namespace string
    filter    string
}

// Push to stack (drill-down)
a.navigationStack = append(a.navigationStack, navHistory{
    resource:  a.currentResource,
    namespace: a.currentNamespace,
    filter:    a.filterText,
})

// Pop from stack (go back)
if len(a.navigationStack) > 0 {
    prev := a.navigationStack[len(a.navigationStack)-1]
    a.navigationStack = a.navigationStack[:len(a.navigationStack)-1]
    a.navigateTo(prev.resource, prev.namespace, prev.filter)
}
```

### Filter State

```go
// Live filtering with debounce (100ms)
a.cmdInput.SetChangedFunc(func(text string) {
    filterTimer.Reset(100 * time.Millisecond)
})

// After debounce timeout:
a.mx.Lock()
a.filterText = text
a.filterRegex = strings.HasPrefix(text, "/") && strings.HasSuffix(text, "/")
a.mx.Unlock()
a.applyFilterText()
```

---

## Concurrency Patterns

### QueueUpdateDraw Wrapper

All UI updates from goroutines must use `QueueUpdateDraw()`:

```go
func (a *App) QueueUpdateDraw(f func()) {
    go func() {
        // Recheck state before queuing (avoid deadlock)
        if atomic.LoadInt32(&a.stopping) == 1 {
            return
        }
        a.Application.QueueUpdateDraw(f)
    }()
}
```

**Why the goroutine wrapper?**
- Prevents deadlock when called from tview input handlers
- Input handlers run on the main event loop
- Direct `QueueUpdateDraw()` from input handler → deadlock
- Goroutine wrapper schedules update asynchronously → safe

### AI Tool Approval Channel

```go
// Non-blocking send (prevent deadlock)
select {
case a.pendingToolApproval <- approved:
    // Success
default:
    // Channel full or no receiver, ignore
}

// Receiver side (with timeout)
select {
case approved := <-a.pendingToolApproval:
    if approved {
        a.executeToolCall()
    }
case <-time.After(30 * time.Second):
    a.cancelToolCall()
case <-ctx.Done():
    return
}
```

### Fetch with Backoff

```go
func (a *App) refresh() {
    go func() {
        backoff := 100 * time.Millisecond
        maxBackoff := 5 * time.Second

        for {
            err := a.fetchResources(ctx)
            if err == nil {
                break
            }

            time.Sleep(backoff)
            backoff *= 2
            if backoff > maxBackoff {
                backoff = maxBackoff
            }
        }
    }()
}
```

---

## File Structure

```
pkg/ui/
├── app.go (2166 lines)              # Main App struct, UI setup
│   ├── New()                        # Constructor
│   ├── Run()                        # Main event loop
│   ├── setupUI()                    # Component initialization
│   └── inputCapture()               # Keyboard handler
│
├── app_navigation.go (521 lines)    # Navigation logic
│   ├── navigateTo()                 # Resource switching
│   ├── drillDown()                  # Enter key handler
│   ├── goBack()                     # Esc key handler
│   └── updateHeader()               # Header rendering
│
├── app_actions.go                   # Action handlers
│   ├── showLogs()                   # Log viewer
│   ├── showDescribe()               # Describe viewer
│   ├── showYAML()                   # YAML viewer
│   ├── showScale()                  # Scale dialog
│   └── showDelete()                 # Delete confirmation
│
├── app_fetch.go                     # Resource fetching
│   ├── fetchResources()             # Dispatcher
│   ├── fetchPods()                  # Pod-specific fetch
│   ├── fetchDeployments()           # Deployment-specific
│   └── applyFilterText()            # Filter application
│
├── ai_panel.go (669 lines)          # AI assistant panel
│   ├── NewAIPanel()                 # Constructor
│   ├── SetAgent()                   # Bind agent
│   ├── Submit()                     # Question submission
│   ├── ShowApproval()               # Tool approval UI
│   └── AppendResponse()             # Streaming response
│
├── briefing.go                      # Cluster health panel
│   ├── BriefingPanel                # Component struct
│   ├── Update()                     # Refresh health data
│   └── Render()                     # Render health score
│
├── vim_viewer.go (409 lines)        # Vim-style text viewer
│   ├── VimViewer                    # Component struct
│   ├── SetContent()                 # Load text content
│   ├── Search()                     # /pattern search
│   └── inputCapture()               # j/k/g/G navigation
│
├── logo.go (359 lines)              # ASCII art logo
│
├── actions/
│   └── actions.go                   # Key action registry
│       ├── KeyAction                # Action definition
│       └── KeyActions               # Action registry
│
├── models/
│   ├── table.go                     # Table data model
│   │   ├── Table                    # Data container
│   │   └── TableListener            # Change observer
│   └── resource.go                  # Resource model
│
├── resources/                       # Resource-specific fetch
│   ├── pods.go
│   ├── deployments.go
│   ├── services.go
│   ├── nodes.go
│   └── types.go
│
├── render/                          # Rendering helpers
│   ├── pod.go                       # Pod formatting
│   ├── deployment.go                # Deployment formatting
│   └── render.go                    # Generic utilities
│
└── views/                           # (Future: view components)
    ├── base.go
    ├── stack.go
    └── registrar.go
```

---

## Vim Viewer Component

**File: `pkg/ui/vim_viewer.go`**

Used for logs, YAML, describe output:

```go
type VimViewer struct {
    *tview.TextView
    searchPattern string
    searchRegex   *regexp.Regexp
    searchMatches []int
    currentMatch  int
    searchMode    bool
    content       string
    lines         []string
    totalLines    int
}
```

### Keybindings

| Key | Action |
|-----|--------|
| `/pattern` | Search |
| `n` | Next match |
| `N` | Previous match |
| `j` / `k` | Down / Up |
| `g` / `G` | Top / Bottom |
| `Ctrl+D` / `Ctrl+U` | Page down / up |
| `Esc` | Close viewer |

---

## Briefing Panel

**File: `pkg/ui/briefing.go`**

Shows cluster health at a glance:

```go
type BriefingData struct {
    HealthScore      int      // 0-100
    HealthStatus     string   // "healthy", "warning", "critical"
    TotalPods        int
    RunningPods      int
    PendingPods      int
    FailedPods       int
    TotalNodes       int
    ReadyNodes       int
    CPUPercent       float64
    MemoryPercent    float64
    Alerts           []string // Warning messages
}
```

### Display

```
┌─ Cluster Health ─────────────────────────────────────────────────┐
│ Health: 95/100 [██████████████████░░] healthy                    │
│ Pods: 45 running, 2 pending, 1 failed                            │
│ Nodes: 3/3 ready | CPU: 45% | Memory: 62%                        │
│ ⚠ 1 pod in CrashLoopBackOff                                      │
└──────────────────────────────────────────────────────────────────┘
```

Toggle with `Shift+B`.

---

## Summary

The k13d TUI architecture follows k9s patterns for:
- **Stability**: Mutex-protected state, atomic guards
- **Responsiveness**: QueueUpdateDraw, non-blocking channels
- **Usability**: Vim keybindings, command mode, drill-down
- **AI Integration**: Tool calling, approval workflow, streaming

Key files:
- `app.go` - Main application and UI setup
- `app_navigation.go` - Resource navigation logic
- `ai_panel.go` - AI assistant integration
- `vim_viewer.go` - Log/YAML viewer

---

## Next Steps

- [User Guide](./USER_GUIDE.md) - How to use k13d
- [MCP Guide](./MCP_GUIDE.md) - AI tool integration
- [Architecture Guide](./ARCHITECTURE.md) - System overview
