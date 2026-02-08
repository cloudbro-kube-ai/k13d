# k13d CLI Improvement Plan

## Analysis Summary

Compared k13d with k9s and web UI to identify missing features and UI/UX improvements.

---

## Priority 1: High Impact (Must Have)

### 1. Resource Filter/Search (`/` key)
- **Current**: Only command mode (`:`) exists
- **k9s**: Fuzzy filter with regex support
- **Implementation**: Add filter input that filters table rows in real-time
- **Status**: TODO

### 2. Delete Resource with Confirmation
- **Current**: No delete functionality
- **k9s**: Delete dialog with propagation options (Foreground/Background/Orphan), force flag
- **Implementation**: Add `d` key for delete with confirmation modal
- **Status**: TODO

### 3. Pod Exec/Shell Access (`s` key)
- **Current**: No shell access
- **k9s**: Interactive shell spawning with container selection
- **Implementation**: Add shell command that spawns external terminal
- **Status**: TODO

### 4. Port Forwarding (`Shift+F`)
- **Current**: No port forward support
- **k9s**: Dialog-based port forward configuration
- **Implementation**: Add port forward dialog for pods/services
- **Status**: TODO

### 5. Context/Cluster Switching
- **Current**: Uses default context only
- **k9s**: Context picker, cluster switching
- **Implementation**: Add context selector modal
- **Status**: TODO

---

## Priority 2: Core Features

### 6. Live Resource Watching (Delta Updates)
- **Current**: Full table reload on refresh
- **k9s**: Delta-based updates with listener pattern
- **Implementation**: Use watch API for incremental updates
- **Status**: TODO

### 7. YAML Syntax Highlighting
- **Current**: Plain text YAML display
- **k9s**: Colorized YAML with key/value distinction
- **Implementation**: Add regex-based colorization
- **Status**: TODO

### 8. Streaming Log Tail
- **Current**: Static log fetch (last N lines)
- **k9s**: Live log streaming with container selection
- **Implementation**: Use log watch API
- **Status**: TODO

### 9. Metrics Display (CPU/Memory)
- **Current**: No metrics shown
- **k9s**: Pulse view with charts
- **Web UI**: Resource usage reports
- **Implementation**: Add metrics columns or dedicated view
- **Status**: TODO

### 10. Health Status Command
- **Current**: No health check
- **Web UI**: /api/health endpoint
- **Implementation**: Add `:health` command
- **Status**: TODO

---

## Priority 3: Polish & UX

### 11. Breadcrumb Navigation
- **Current**: Single view with modal overlays
- **k9s**: Full breadcrumb trail (Pods > nginx-xxx > logs)
- **Implementation**: Add crumb bar below header
- **Status**: TODO

### 12. Hotkey Customization
- **Current**: Hardcoded keybindings
- **k9s**: YAML-based hotkey definitions
- **Implementation**: Load from config file
- **Status**: DONE (hotkeys.yaml support)

### 13. Column Sorting
- **Current**: Fixed order
- **k9s**: Click header to sort
- **Implementation**: Add sort by column
- **Status**: DONE (sort with visual indicators + per-resource defaults via views.yaml)

### 14. Resource Age Formatting
- **Current**: Simple duration (1h, 2d)
- **k9s**: Relative time with color coding (older = yellow/red)
- **Implementation**: Color-code old resources
- **Status**: TODO

### 15. Audit Log Viewer
- **Current**: No audit viewing
- **Web UI**: Full audit log API
- **Implementation**: Add `:audit` command
- **Status**: TODO

---

## UI/UX Improvements

### Visual
- [ ] Add loading spinner animation instead of static "Loading..."
- [ ] Add row highlighting on hover/focus
- [ ] Add alternating row colors for readability
- [ ] Show resource count in status bar
- [ ] Add "last refresh" timestamp

### Interaction
- [ ] Add vim-like j/k navigation (partially done)
- [ ] Add page up/down for fast scrolling
- [ ] Add Ctrl+U/Ctrl+D for half-page scroll
- [ ] Add search highlight in YAML view
- [ ] Add copy-to-clipboard for selected row

### Feedback
- [ ] Show operation progress (delete, scale, etc.)
- [ ] Add sound/bell on errors (optional)
- [ ] Flash message improvements (icons, auto-dismiss timing)

---

## Implementation Order

```
Phase 1 (v0.2.0):
[x] Autocomplete with hints
[x] AI input field
[x] Filter/Search (/)
[x] Delete with confirmation

Phase 2 (v0.3.0):
[x] Pod exec/shell
[x] Port forwarding
[x] Context switching
[x] Health status

Phase 3 (v0.4.0):
[ ] Live watching
[x] YAML highlighting
[ ] Metrics display
[x] Streaming logs

Phase 4 (v0.5.0):
[x] Hotkey customization
[x] Audit viewer
[x] Column sorting
[ ] Breadcrumbs

Phase 5 (v0.7.0 - Current):
[x] Screen ghosting fix (draw throttle + safety sync)
[x] AI chat history preservation
[x] Autocomplete dropdown overlay
[x] Resource aliases (aliases.yaml)
[x] Per-resource sort defaults (views.yaml)
[x] LLM model switching (:model command)
[x] Plugin system TUI integration
```

---

## Architecture Recommendations (from k9s)

### Model-View-Controller Pattern
- **Models**: Data management with listeners
- **Views**: UI rendering with action handlers
- **Render**: Resource-specific formatters

### Listener Pattern
```go
type TableListener interface {
    TableDataChanged(*TableData)
    TableLoadFailed(error)
}
```

### Action System
```go
type KeyAction struct {
    Description string
    Handler     ActionHandler
    Dangerous   bool  // Requires confirmation
    Visible     bool  // Show in help
}
```

---

## Files to Create/Modify

```
pkg/ui/
├── app.go          # Main app (modify)
├── app_test.go     # Tests (done)
├── filter.go       # NEW: Filter input
├── dialog.go       # NEW: Modal dialogs
├── exec.go         # NEW: Shell execution
├── portforward.go  # NEW: Port forwarding
├── context.go      # NEW: Context switching
├── yaml.go         # NEW: YAML colorizer
├── metrics.go      # NEW: Metrics display
└── resources/      # Resource renderers (exists)
```
