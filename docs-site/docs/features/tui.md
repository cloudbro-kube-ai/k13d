# TUI Features

Complete feature reference for k13d Terminal User Interface (TUI).

---

## Dashboard Overview

The TUI provides a k9s-style terminal dashboard for Kubernetes management.

### Main Interface

![TUI Full Screen](../images/tui-full-screen.png)

The TUI interface provides:

- **Left Panel**: Resource list with status indicators
- **Right Panel**: AI Assistant for natural language queries
- **Bottom**: Status bar with context and namespace info

### Full Screen View

![TUI Full Screen 2](../images/tui-full-screen2.png)

Different resource views with real-time updates.

---

## Navigation

### Vim-Style Keybindings

| Key | Action | Description |
|-----|--------|-------------|
| `j` / `↓` | Down | Move selection down |
| `k` / `↑` | Up | Move selection up |
| `g` | Top | Go to first item |
| `G` | Bottom | Go to last item |
| `Ctrl+f` | Page Down | Scroll page down |
| `Ctrl+b` | Page Up | Scroll page up |
| `Ctrl+d` | Half Down | Scroll half page down |
| `Ctrl+u` | Half Up | Scroll half page up |

### Panel Navigation

| Key | Action | Description |
|-----|--------|-------------|
| `Tab` | Next Panel | Switch to next panel |
| `Shift+Tab` | Prev Panel | Switch to previous panel |
| `←` / `→` | Left/Right | Switch panels |
| `1-9` | Panel N | Quick switch to panel N |

### Command Mode

| Command | Description |
|---------|-------------|
| `:pods` or `:po` | View pods |
| `:svc` or `:services` | View services |
| `:deploy` | View deployments |
| `:rs` | View replica sets |
| `:sts` | View statefulsets |
| `:ds` | View daemonsets |
| `:jobs` | View jobs |
| `:cronjobs` | View cron jobs |
| `:cm` | View configmaps |
| `:secrets` | View secrets |
| `:ingress` | View ingresses |
| `:pv` | View persistent volumes |
| `:pvc` | View persistent volume claims |
| `:nodes` | View nodes |
| `:ns` | View namespaces |
| `:events` | View events |
| `:helm` | View Helm releases |

### Management Commands

| Command | Description |
|---------|-------------|
| `:alias` | View all configured resource aliases |
| `:model` | Open AI model profile selector |
| `:model <name>` | Switch directly to a named model profile |
| `:plugins` | View available plugins with shortcuts |
| `:health` or `:status` | Check system status |
| `:audit` | View audit log |

### Autocomplete

When typing a command, k13d shows autocomplete suggestions:

- **Single match**: Dimmed hint text appears next to cursor (press `Tab` to complete)
- **Multiple matches**: Dropdown overlay appears above the command bar
  - **Up/Down arrows**: Navigate suggestions
  - **Tab or Enter**: Select suggestion
  - **Esc**: Dismiss dropdown
  - Custom aliases from `aliases.yaml` are included in suggestions

---

## Resource Actions

### General Actions

| Key | Action | Description |
|-----|--------|-------------|
| `y` | YAML | View resource YAML manifest |
| `d` | Describe | Show resource description |
| `e` | Edit | Edit resource in $EDITOR |
| `Ctrl+D` | Delete | Delete resource (with confirmation) |
| `Enter` | Details | Show detailed view |

### Pod Actions

| Key | Action | Description |
|-----|--------|-------------|
| `l` | Logs | View pod logs |
| `L` | Previous Logs | View previous container logs |
| `x` | Exec | Open shell in container |
| `p` | Port Forward | Start port forwarding |
| `f` | Follow | Follow logs in real-time |

### Deployment Actions

| Key | Action | Description |
|-----|--------|-------------|
| `s` | Scale | Scale replica count |
| `r` | Restart | Rollout restart |
| `h` | History | View rollout history |
| `u` | Undo | Rollback to previous version |

### Node Actions

| Key | Action | Description |
|-----|--------|-------------|
| `c` | Cordon | Mark node unschedulable |
| `C` | Uncordon | Mark node schedulable |
| `D` | Drain | Drain node (evict pods) |

---

## AI Assistant

### AI Panel

![TUI AI Panel](../images/tui-assistant-pannel.png)

The AI panel is accessible by pressing `Tab` to switch focus.

| Feature | Description |
|---------|-------------|
| **Input Field** | Type questions in natural language |
| **Output View** | Streaming AI responses |
| **Tool Execution** | AI executes kubectl/bash commands |
| **Context** | AI receives selected resource context |
| **Chat History** | Previous Q&A preserved within session |

### Chat History

AI conversations are preserved within each TUI session:

- Previous Q&A sessions are kept above, separated by visual dividers (`────────────────────────────`)
- Scroll up in the AI panel to review past conversations
- History is maintained for the duration of the TUI session

### AI Conversation

![TUI AI Conversation](../images/tui-ask-answer-test.png)

Example AI interaction showing:

- Natural language question input
- AI response with analysis
- Tool execution results

### LLM Model Switching

Switch between configured AI model profiles:

- **`:model`** - Opens a modal showing all profiles (active marked with `*`)
- **`:model gpt-4o`** - Switch directly to a named profile

Model profiles are defined in `~/.config/k13d/config.yaml` under the `models` section.

### AI Actions

| Key | Action | Description |
|-----|--------|-------------|
| `Shift+L` | AI Analyze | Deep analysis of selected resource |
| `h` | Explain | Get explanation of resource |

### Tool Approval

![TUI Decision Required](../images/tui-decision-required.png)

When AI needs to execute a command, an approval dialog appears:

| Key | Action |
|-----|--------|
| `Y` | Approve this command |
| `N` | Reject this command |
| `A` | Always approve read-only commands |

---

## Filtering & Search

### Filter Mode

| Key | Action |
|-----|--------|
| `/` | Start filter mode |
| `Enter` | Apply filter |
| `Esc` | Clear filter |

**Filter Syntax:**

```
/nginx          # Contains "nginx"
/^nginx         # Starts with "nginx"
/nginx$         # Ends with "nginx"
/nginx.*running # Regex pattern
```

---

## Resource Aliases

Define custom command aliases in `~/.config/k13d/aliases.yaml`:

```yaml
aliases:
  pp: pods
  dep: deployments
  sec: secrets
  cm: configmaps
```

- Type `:pp` to navigate to Pods (resolved via alias)
- Use `:alias` to view all active aliases (built-in + custom)
- Custom aliases appear in autocomplete suggestions

---

## Per-Resource Sort Defaults

Configure default sort column and direction in `~/.config/k13d/views.yaml`:

```yaml
views:
  pods:
    sortColumn: AGE
    sortAscending: false
  deployments:
    sortColumn: NAME
    sortAscending: true
```

Sort preferences are applied automatically when navigating to each resource type.

---

## Plugin System

k13d supports external plugins that extend the TUI with custom commands.

- **`:plugins`** - Show all configured plugins with shortcuts and scopes
- Plugin keyboard shortcuts are active when viewing matching resource types
- **Foreground plugins**: Temporarily suspend TUI, restore after command finishes
- **Background plugins**: Run silently without interrupting workflow

Configure in `~/.config/k13d/plugins.yaml`:

```yaml
plugins:
  dive:
    shortCut: Ctrl-I
    description: "Dive into container image layers"
    scopes: [pods]
    command: dive
    args: [$IMAGE]
    background: false
    confirm: false
```

---

## YAML Viewer

View resource YAML manifests with syntax highlighting.

| Feature | Description |
|---------|-------------|
| **Syntax Highlighting** | Color-coded YAML |
| **Scrolling** | Navigate with j/k |
| **Copy** | Copy to clipboard |
| **Exit** | Press `Esc` or `q` |

---

## Log Viewer

Real-time log streaming with ANSI color support.

| Feature | Description |
|---------|-------------|
| **Real-time Streaming** | Auto-scroll with new logs |
| **ANSI Colors** | Full color support |
| **Container Selection** | Choose container for multi-container pods |
| **Previous Logs** | View crashed container logs |
| **Follow Mode** | Toggle auto-follow |

| Key | Action |
|-----|--------|
| `f` | Toggle follow mode |
| `w` | Wrap lines |
| `/` | Search logs |
| `Esc` | Exit log viewer |

---

## Terminal/Shell

Execute commands in pod containers.

| Feature | Description |
|---------|-------------|
| **Shell Access** | /bin/bash or /bin/sh |
| **Container Selection** | Choose container |
| **Exit** | Type `exit` or Ctrl+D |

---

## Port Forward

Forward container ports to local machine.

| Feature | Description |
|---------|-------------|
| **Local Port** | Specify local port |
| **Container Port** | Target container port |
| **Status** | Active/Stopped indicator |
| **Stop** | Press `q` to stop forwarding |

---

## Resource Highlighting

Visual indicators for resource status.

| Color | Status |
|-------|--------|
| 🟢 Green | Running, Ready, Healthy |
| 🟡 Yellow | Pending, Progressing |
| 🔴 Red | Failed, Error, CrashLoopBackOff |
| 🔵 Blue | Selected item |
| ⚪ Gray | Terminated, Completed |

---

## Help System

Built-in help for keybindings and commands.

| Key | Action |
|-----|--------|
| `?` | Show help modal |
| `Esc` | Close help |

---

## i18n Support

Multi-language interface support.

| Language | Configuration |
|----------|--------------|
| English | `language: en` |
| 한국어 | `language: ko` |
| 中文 | `language: zh` |
| 日本語 | `language: ja` |

Configure in `~/.config/k13d/config.yaml`:

```yaml
language: ko  # Korean
```

---

## Tokyo Night Theme

k13d TUI uses a Tokyo Night-inspired color scheme:

| Element | Color |
|---------|-------|
| Background | Dark blue-gray (#1a1b26) |
| Text | Light gray (#c0caf5) |
| Selection | Blue highlight (#7aa2f7) |
| Errors | Red (#f7768e) |
| Success | Green (#9ece6a) |
| Warning | Yellow (#e0af68) |

---

## Quick Reference

```
┌──────────────────────────────────────────┐
│           k13d TUI Quick Reference        │
├──────────────────────────────────────────┤
│ Navigation                               │
│   j/k          Move up/down              │
│   g/G          First/Last item           │
│   Tab          Switch panel              │
│   :            Command mode              │
│   /            Filter                    │
│                                          │
│ Actions                                  │
│   y            View YAML                 │
│   d            Describe                  │
│   l            Logs                      │
│   x            Exec/Shell                │
│   s            Scale                     │
│   r            Restart                   │
│   Ctrl+D       Delete                    │
│                                          │
│ AI                                       │
│   Shift+L      AI Analyze                │
│   h            Explain This              │
│   Y/N          Approve/Reject            │
│                                          │
│ Commands                                 │
│   :alias       View aliases              │
│   :model       Switch AI model           │
│   :plugins     View plugins              │
│                                          │
│ Global                                   │
│   ?            Help                      │
│   Esc          Cancel/Back               │
│   q            Quit                      │
└──────────────────────────────────────────┘
```
