# k13d Stability Architecture

This document describes the internal stability patterns that ensure k13d operates reliably in production environments. It serves as a guide for contributors and maintainers working on the TUI codebase.

## Table of Contents

- [Lock Ordering Convention](#lock-ordering-convention)
- [Goroutine Lifecycle](#goroutine-lifecycle)
- [Error Recovery Strategy](#error-recovery-strategy)
- [Watcher & Refresh Synchronization](#watcher--refresh-synchronization)
- [Shutdown Sequence](#shutdown-sequence)
- [Thread-Safe UI Updates](#thread-safe-ui-updates)

---

## Lock Ordering Convention

k13d uses multiple mutexes to protect concurrent access to shared state. **Lock ordering is critical** to prevent deadlocks.

### Lock Hierarchy (Must Follow This Order)

```
watchMu → navMx → mx
```

**Rule**: Always acquire locks from left to right. **Never nest locks in reverse order.**

### Lock Definitions

| Lock | Type | Purpose | Scope |
|------|------|---------|-------|
| `watchMu` | `sync.Mutex` | Protects watcher lifecycle (start/stop) | `pkg/ui/app.go` |
| `navMx` | `sync.Mutex` | Protects navigation stack (back/forward history) | `pkg/ui/app.go` |
| `mx` | `sync.RWMutex` | Protects app state (resource, namespace, filters, sorts) | `pkg/ui/app.go` |
| `pfMx` | `sync.Mutex` | Protects port-forward tracking | `pkg/ui/app.go` |
| `aiMx` | `sync.RWMutex` | Protects AI decision approval state | `pkg/ui/app.go` |
| `cancelLock` | `sync.Mutex` | Protects context cancellation | `pkg/ui/app.go` |

### Lock Ordering Examples

#### ✅ Correct: watchMu → mx

```go
func (a *App) startWatch() {
    a.watchMu.Lock()
    defer a.watchMu.Unlock()

    // Later acquire mx (allowed: watchMu before mx)
    a.mx.RLock()
    resource := a.currentResource
    a.mx.RUnlock()

    // ... start watcher ...
}
```

#### ❌ WRONG: mx → watchMu (Deadlock Risk!)

```go
// NEVER DO THIS - violates lock ordering
func badFunction() {
    a.mx.Lock()              // Acquire mx first
    a.watchMu.Lock()         // Then watchMu - WRONG ORDER!
    // ...
}
```

### The `navigateTo()` Centralized State Transition

`navigateTo()` is the **single entry point** for all resource/namespace transitions. It enforces the lock ordering convention:

```go
func (a *App) navigateTo(resource, namespace, filter string) {
    // 1. Stop watch first (watchMu)
    a.stopWatch()

    // 2. Update state (mx)
    a.mx.Lock()
    a.currentResource = resource
    a.currentNamespace = namespace
    a.filterText = filter
    a.sortColumn = -1
    a.sortAscending = true
    a.mx.Unlock()

    // 3. Refresh and start new watch (no locks held)
    a.safeGo("navigateTo-refresh", func() {
        a.updateHeader()
        a.updateStatusBar()
        a.refresh()
        a.startWatch() // Start watch for new resource
    })
}
```

**Why this works:**
1. `stopWatch()` acquires `watchMu` alone
2. State mutation acquires `mx` **after** `watchMu` is released
3. `startWatch()` runs in goroutine, acquiring `watchMu` again (no deadlock risk)

---

## Goroutine Lifecycle

k13d uses goroutines extensively for async operations. All goroutines use the `safeGo()` wrapper for panic recovery.

### The `safeGo()` Pattern (k9s RunE Pattern)

```go
// safeGo wraps goroutines with panic recovery
func (a *App) safeGo(name string, fn func()) {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                a.logger.Error("goroutine panic recovered",
                    "name", name,
                    "error", r,
                    "stack", string(debug.Stack()))
                a.flashMsg(fmt.Sprintf("Internal error in %s. The operation has been recovered. Please try again or check logs for details.", name), true)
            }
        }()
        fn()
    }()
}
```

### Goroutine Usage Examples

```go
// Resource loading
a.safeGo("loadAPIResources", a.loadAPIResources)

// AI queries
a.safeGo("askAI", func() {
    a.askAI(question)
})

// Refresh after mutation
a.safeGo("refresh-after-execute", a.refresh)
```

**Key Points:**
- **Always** use `safeGo()` instead of raw `go` statements
- Provide descriptive names for debugging (shows in logs)
- Panics are logged with full stack trace
- User sees friendly error message in TUI

### Context Propagation and Cancellation

Use `context.WithCancel` or `context.WithTimeout` for operations that need to be aborted:

```go
a.safeGo("editResource-fetch", func() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    yaml, err := a.k8s.GetResourceYAML(ctx, ns, name, gvr)
    if ctx.Err() != nil {
        a.logger.Debug("Fetch cancelled", "resource", resource)
        return // Silent return on cancellation
    }
    // ... use yaml ...
})
```

### Atomic Guards (Lock-Free Coordination)

k13d uses `sync/atomic` for lightweight state checks without mutexes:

| Atomic Flag | Purpose |
|-------------|---------|
| `stopping` | Set to 1 immediately when `Stop()` is called |
| `running` | Set to 1 after `Application.Run()` starts |
| `inUpdate` | Deduplicates concurrent refresh() calls |
| `hasToolCall` | Tracks pending AI tool approvals |
| `needsSync` | Triggers full terminal re-sync |

**Example:**

```go
func (a *App) refresh() {
    // Prevent duplicate concurrent refreshes
    if !atomic.CompareAndSwapInt32(&a.inUpdate, 0, 1) {
        return // Already refreshing
    }
    defer atomic.StoreInt32(&a.inUpdate, 0)

    // Check if shutting down
    if atomic.LoadInt32(&a.stopping) == 1 {
        return
    }

    // ... perform refresh ...
}
```

---

## Error Recovery Strategy

k13d uses a multi-layered approach to error handling:

### 1. Panic Recovery (safeGo)

All goroutines recover from panics:

```go
a.safeGo("operation", func() {
    // If this panics, it's caught by safeGo's defer/recover
    dangerousOperation()
})
```

### 2. Kubernetes API Failure Handling

Use exponential backoff for transient failures:

```go
func (a *App) refresh() {
    b := backoff.NewExponentialBackOff()
    b.MaxElapsedTime = 30 * time.Second

    var pods []k8s.Pod
    err := backoff.Retry(func() error {
        var err error
        pods, err = a.k8s.ListPods(ctx, namespace)
        return err
    }, b)

    if err != nil {
        if ctx.Err() != nil {
            // Context cancelled - silently return
            return
        }
        // Real error - show to user
        a.flashMsg(fmt.Sprintf("Failed to load pods: %v", err), true)
        return
    }
    // ... populate table ...
}
```

### 3. Graceful Degradation

When non-critical features fail, continue operation:

```go
// AI client initialization failure - continue without AI
newClient, err := ai.NewClient(&a.config.LLM)
if err != nil {
    a.logger.Warn("AI client init failed", "error", err)
    a.aiClient = nil // Disable AI, but TUI still works
} else {
    a.aiClient = newClient
}
```

### 4. User-Friendly Error Messages

Always provide actionable error messages:

```go
// ❌ BAD: Generic, unhelpful
a.flashMsg("Error: %v", err)

// ✅ GOOD: Specific, actionable
a.flashMsg(fmt.Sprintf("Failed to load %s: %v. Check cluster connectivity and permissions.", resource, err), true)
```

---

## Watcher & Refresh Synchronization

k13d uses Kubernetes watch API for real-time resource updates. The watch/refresh cycle is carefully synchronized to avoid race conditions.

### Watch → Debounce → onChange → Refresh Flow

```
┌──────────────┐
│   K8s API    │
│   Watch      │
└──────┬───────┘
       │
       │ Resource changed event
       ▼
┌──────────────────┐
│  ResourceWatcher │
│   (debounced)    │
└──────┬───────────┘
       │
       │ onChange callback (100ms debounce)
       ▼
┌──────────────┐
│ a.refresh()  │ ← Fetches latest data and updates table
└──────────────┘
```

### Debouncing (k9s Pattern)

Rapid K8s API events are debounced to avoid UI thrashing:

```go
type ResourceWatcher struct {
    debounce time.Duration // Default: 100ms
    timer    *time.Timer
}

func (w *ResourceWatcher) eventLoop(ctx context.Context) {
    for event := range w.eventChan {
        // Reset timer on each event
        w.timer.Reset(w.debounce)
    }
}

func (w *ResourceWatcher) debounceLoop(ctx context.Context) {
    for {
        select {
        case <-w.timer.C:
            w.onChange() // Call callback after debounce period
        case <-ctx.Done():
            return
        }
    }
}
```

### Stop Guard with `isStopped()`

The watcher checks if it's been stopped before invoking callbacks:

```go
func (w *ResourceWatcher) isStopped() bool {
    w.mu.RLock()
    defer w.mu.RUnlock()
    return w.stopped
}

func (w *ResourceWatcher) debounceLoop(ctx context.Context) {
    for {
        select {
        case <-w.timer.C:
            if !w.isStopped() {
                w.onChange()
            }
        case <-ctx.Done():
            return
        }
    }
}
```

### Resource Switch Sequence

When switching resources (e.g., `:pods` → `:svc`):

```
1. stopWatch()      → Cancel old watcher context
                    → Set watcher.stopped = true
                    → Drain remaining events

2. State change     → Update currentResource, currentNamespace
                    → Reset filters, sorts

3. startWatch()     → Create new watcher for new resource
                    → Start new watch goroutines
```

**Critical:** Always call `stopWatch()` **before** changing state to prevent stale events from updating the wrong resource view.

---

## Shutdown Sequence

k13d follows a graceful shutdown sequence to avoid resource leaks and crashes.

### Shutdown Flow

```
User presses 'q'
       │
       ▼
┌──────────────────────────────────────┐
│ 1. Set stopping = 1 (atomic)         │ ← Prevent new operations
└──────┬───────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────┐
│ 2. Cancel in-flight operations       │ ← Call cancelFn()
└──────┬───────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────┐
│ 3. Stop watcher (stopWatch)          │ ← Cancel watch context
└──────┬───────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────┐
│ 4. Cleanup resources                 │ ← Close channels, stop goroutines
└──────┬───────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────┐
│ 5. tview.Application.Stop()          │ ← Exit event loop
└──────────────────────────────────────┘
```

### Shutdown Code

```go
func (a *App) Stop() {
    // 1. Set atomic flag immediately (guards against new operations)
    atomic.StoreInt32(&a.stopping, 1)

    // 2. Cancel all in-flight contexts
    a.cancelLock.Lock()
    if a.cancelFn != nil {
        a.cancelFn()
    }
    a.cancelLock.Unlock()

    // 3. Stop watcher
    a.stopWatch()

    // 4. Stop tview application (blocks until event loop exits)
    if a.Application != nil {
        a.Application.Stop()
    }
}
```

### Checking for Shutdown in Operations

All long-running operations check the `stopping` flag:

```go
func (a *App) refresh() {
    // Exit early if shutting down
    if atomic.LoadInt32(&a.stopping) == 1 {
        return
    }

    // Prevent duplicate refreshes
    if !atomic.CompareAndSwapInt32(&a.inUpdate, 0, 1) {
        return
    }
    defer atomic.StoreInt32(&a.inUpdate, 0)

    // ... perform refresh ...
}
```

### Avoiding Deadlocks on Shutdown

**Problem:** Goroutines blocking on mutexes during shutdown.

**Solution:** Use atomic flags (`stopping`) to exit early **before** acquiring locks:

```go
func (a *App) startWatch() {
    // Check stopping flag BEFORE acquiring watchMu
    if atomic.LoadInt32(&a.stopping) == 1 {
        return
    }

    a.watchMu.Lock()
    defer a.watchMu.Unlock()
    // ... start watch ...
}
```

---

## Thread-Safe UI Updates

tview is **not thread-safe**. All UI updates from goroutines must go through `QueueUpdateDraw()`.

### QueueUpdateDraw Pattern

```go
// From a goroutine (e.g., safeGo callback)
a.safeGo("fetchData", func() {
    data := fetchData() // Background operation

    // Update UI on main thread
    a.QueueUpdateDraw(func() {
        a.table.SetCell(0, 0, tview.NewTableCell(data))
    })
})
```

### QueueUpdateDraw Implementation

```go
func (a *App) QueueUpdateDraw(f func()) {
    // Guard: Don't queue if stopping or app is nil
    if a.Application == nil || atomic.LoadInt32(&a.stopping) == 1 {
        return
    }

    // IMPORTANT: Run in goroutine to avoid deadlock when called from input handlers
    go func() {
        a.Application.QueueUpdateDraw(f)
    }()
}
```

**Why the extra goroutine?**

tview's `QueueUpdateDraw` can block if called from within an input handler (because the event loop is waiting for the handler to return). Wrapping it in a goroutine makes it non-blocking.

### Direct Update (queueUpdateDrawDirect)

Use when **already outside** the event loop (e.g., from `safeGo` callback):

```go
func (a *App) queueUpdateDrawDirect(f func()) {
    if a.Application == nil || atomic.LoadInt32(&a.stopping) == 1 {
        return
    }
    a.Application.QueueUpdateDraw(f) // No extra goroutine needed
}
```

### AI Streaming Updates (Throttled)

AI responses stream token-by-token. Throttle updates to avoid UI thrashing:

```go
func (a *App) appendAIResponse(text string) {
    now := time.Now().UnixNano()
    last := atomic.LoadInt64(&a.lastAIDraw)

    // Throttle: Only update UI every 50ms
    if now-last < 50*time.Millisecond.Nanoseconds() {
        return
    }

    if atomic.CompareAndSwapInt64(&a.lastAIDraw, last, now) {
        a.QueueUpdateDraw(func() {
            // Update AI panel with new text
            a.aiPanel.SetText(a.aiPanel.GetText(true) + text)
        })
    }
}
```

---

## Testing Stability

k13d includes regression tests for stability patterns:

### Test Files

| File | Purpose |
|------|---------|
| `pkg/ui/stability_regression_test.go` | Tests deadlock scenarios, lock ordering, shutdown |
| `pkg/ui/feature_test.go` | E2E tests with concurrency checks |
| `pkg/ui/golden_test.go` | Screen snapshot tests |

### Running Stability Tests

```bash
# Always use -race flag for TUI tests
go test -race ./pkg/ui/ -run Stability

# Check for deadlocks
go test -race -timeout=10s ./pkg/ui/ -run TestNavigationDeadlock
```

### Example: Testing Lock Ordering

```go
func TestNavigationDeadlock(t *testing.T) {
    app := NewTestApp()
    defer app.Stop()

    // Spawn 100 concurrent navigations
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            resource := []string{"pods", "svc", "deploy"}[idx%3]
            app.navigateTo(resource, "", "")
        }(i)
    }

    // Should complete without deadlock
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        // Success
    case <-time.After(5 * time.Second):
        t.Fatal("Deadlock detected: navigation did not complete")
    }
}
```

---

## Common Pitfalls & Solutions

### Pitfall 1: Acquiring Locks in Wrong Order

**Problem:**
```go
func badFunction() {
    a.mx.Lock()      // Acquire mx first
    a.watchMu.Lock() // Then watchMu - WRONG!
}
```

**Solution:**
Follow the lock hierarchy: `watchMu → navMx → mx`

---

### Pitfall 2: Forgetting to Release Locks

**Problem:**
```go
func badFunction() {
    a.mx.Lock()
    if err != nil {
        return // OOPS: forgot to unlock!
    }
    a.mx.Unlock()
}
```

**Solution:**
Always use `defer`:
```go
func goodFunction() {
    a.mx.Lock()
    defer a.mx.Unlock()

    if err != nil {
        return // Lock released automatically
    }
}
```

---

### Pitfall 3: UI Updates Without QueueUpdateDraw

**Problem:**
```go
a.safeGo("fetch", func() {
    data := fetch()
    a.table.SetCell(0, 0, tview.NewTableCell(data)) // CRASH: not thread-safe!
})
```

**Solution:**
```go
a.safeGo("fetch", func() {
    data := fetch()
    a.QueueUpdateDraw(func() {
        a.table.SetCell(0, 0, tview.NewTableCell(data))
    })
})
```

---

### Pitfall 4: Not Checking `stopping` Flag

**Problem:**
```go
func (a *App) longOperation() {
    a.mx.Lock()
    defer a.mx.Unlock()
    // If Stop() was called, this will deadlock or cause errors
    // ... long operation ...
}
```

**Solution:**
```go
func (a *App) longOperation() {
    if atomic.LoadInt32(&a.stopping) == 1 {
        return // Exit early
    }

    a.mx.Lock()
    defer a.mx.Unlock()
    // ... operation ...
}
```

---

## Debugging Tips

### Enable Debug Logging

```yaml
# config.yaml
log_level: debug
```

### Check for Data Races

```bash
go test -race ./pkg/ui/
```

### Analyze Deadlocks

```bash
# Run with GODEBUG to see mutex contention
GODEBUG=schedtrace=1000,scheddetail=1 k13d
```

### Profile the TUI

```bash
go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=.
go tool pprof cpu.prof
```

---

## Summary

k13d's stability architecture follows these principles:

1. **Strict Lock Ordering**: `watchMu → navMx → mx`
2. **Centralized State Transitions**: `navigateTo()` is the single entry point
3. **Panic Recovery**: All goroutines use `safeGo()`
4. **Graceful Shutdown**: Atomic `stopping` flag prevents new operations
5. **Thread-Safe UI**: All UI updates via `QueueUpdateDraw()`
6. **Watch/Refresh Sync**: Debounced watchers with stop guards
7. **Error Recovery**: Exponential backoff for transient failures

These patterns ensure k13d remains responsive and stable even under high load, rapid navigation, and concurrent operations.

---

## References

- **k9s Stability**: github.com/derailed/k9s (RunE pattern, watch debouncing)
- **tview Threading**: github.com/rivo/tview (QueueUpdateDraw documentation)
- **Go Concurrency**: golang.org/doc/effective_go#concurrency
- **Lock Ordering**: "The Little Book of Semaphores" by Allen B. Downey
