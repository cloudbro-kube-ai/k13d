# TUI Features

Complete feature reference for k13d Terminal User Interface (TUI).

---

## Dashboard Overview

The TUI provides a k9s-style terminal dashboard for Kubernetes management.

### Main Interface

![TUI Help](../images/tui_help.png)

The TUI interface provides:

- **Left Panel**: Resource list with status indicators
- **Right Panel**: AI Assistant for natural language queries
- **Bottom**: Status bar with context, namespace, sort, and filter info

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

![TUI Autocomplete Dropdown](../images/tui_auto_complete.png)

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

### LLM Model Switching

Switch between configured AI model profiles:

- **`:model`** - Opens a modal showing all profiles (active marked with `*`)
- **`:model gpt-4o`** - Switch directly to a named profile

Model profiles are defined in `~/.config/k13d/config.yaml` under the `models` section.

![TUI LLM Settings](../images/tui_llm_setting.png)

### AI Actions

| Key | Action | Description |
|-----|--------|-------------|
| `Shift+L` | AI Analyze | Deep analysis of selected resource |
| `h` | Explain | Get explanation of resource |

### Tool Approval

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

Define custom command aliases in `~/.config/k13d/aliases.yaml` for quick resource navigation.

```yaml title="~/.config/k13d/aliases.yaml"
aliases:
  pp: pods
  dep: deployments
  svc: services
  sec: secrets
  cm: configmaps
  ds: daemonsets
  sts: statefulsets
  rs: replicasets
  cj: cronjobs
```

### How It Works

1. Type `:pp` in the command bar → resolves to `:pods` and navigates to Pods view
2. Custom aliases appear in autocomplete suggestions alongside built-in commands
3. Use `:alias` to view all active aliases (built-in + custom)
4. If a custom alias conflicts with a built-in, the custom alias takes precedence

### Alias Display

Type `:alias` to see the full alias table:

```
┌─ Aliases (Esc to close) ────────────────────────┐
│                                                  │
│   ALIAS     RESOURCE          SOURCE             │
│   ────      ────────          ──────             │
│   po        pods              built-in           │
│   svc       services          built-in           │
│   deploy    deployments       built-in           │
│   pp        pods              custom             │
│   sec       secrets           custom             │
│   cm        configmaps        custom             │
│                                                  │
│ Press Esc to close                               │
└──────────────────────────────────────────────────┘
```

See [Configuration > Aliases](../getting-started/configuration.md#resource-aliases-aliasesyaml) for more details.

---

## Custom Hotkeys

Define custom keyboard shortcuts in `~/.config/k13d/hotkeys.yaml` that execute external CLI commands.

```yaml title="~/.config/k13d/hotkeys.yaml"
hotkeys:
  stern-logs:
    shortCut: "Shift-L"
    description: "Stern multi-pod logs"
    scopes: [pods, deployments]
    command: stern
    args: ["-n", "$NAMESPACE", "$NAME"]
    dangerous: false

  port-forward-8080:
    shortCut: "Ctrl-P"
    description: "Port forward to 8080"
    scopes: [pods, services]
    command: kubectl
    args: [port-forward, -n, "$NAMESPACE", "$NAME", "8080:8080"]
    dangerous: false
```

Hotkeys support `$NAMESPACE`, `$NAME`, and `$CONTEXT` variable expansion. Set `dangerous: true` to require confirmation before execution.

See [Configuration > Hotkeys](../getting-started/configuration.md#custom-hotkeys-hotkeysyaml) for the full reference.

---

## Per-Resource Sort Defaults

Configure default sort column and direction in `~/.config/k13d/views.yaml`. Sort preferences are applied automatically when navigating to each resource type.

```yaml title="~/.config/k13d/views.yaml"
views:
  pods:
    sortColumn: AGE
    sortAscending: false         # Newest first
  deployments:
    sortColumn: NAME
    sortAscending: true          # Alphabetical
  services:
    sortColumn: TYPE
    sortAscending: true
  nodes:
    sortColumn: STATUS
    sortAscending: true
```

k13d automatically detects column types for proper sorting:

- **Numeric**: RESTARTS, COUNT, DESIRED, CURRENT, AVAILABLE
- **Ready format**: READY (e.g., `1/2`, `3/3`) — compares by ratio
- **Age format**: AGE (e.g., `5d`, `3h`, `10m`) — converts to seconds
- **String**: NAME, STATUS, TYPE — case-insensitive alphabetical

See [Configuration > Views](../getting-started/configuration.md#per-resource-sort-defaults-viewsyaml) for more details.

---

## Plugin System

k13d supports external plugins that extend the TUI with custom commands, following the k9s plugin pattern.

### How It Works

1. Define plugins in `~/.config/k13d/plugins.yaml`
2. Each plugin binds a keyboard shortcut to an external CLI command
3. Plugins are **scoped** to specific resource types (or all with `"*"`)
4. When you press the shortcut on a matching resource, the command runs with resource context (name, namespace, image, etc.) automatically injected

### Viewing Plugins

Type `:plugins` in the TUI to see all configured plugins:

```
┌─ Plugins (Esc to close) ────────────────────────────────────┐
│ Configured Plugins                                          │
│                                                             │
│   NAME            SHORTCUT     SCOPES               DESC   │
│   ────            ────────     ──────               ─────  │
│   dive            Ctrl-I       pods                 Dive…  │
│   debug           Shift-D      pods                 Debug… │
│   port-forward    Shift-F      pods, services       Port…  │
│   stern           Shift-S      deployments          Stre…  │
│   lens            Ctrl-O       *                    Open…  │
│                                                             │
│   Total: 5 plugins loaded                                   │
│                                                             │
│ Config: ~/.config/k13d/plugins.yaml                         │
│ Variables: $NAMESPACE, $NAME, $CONTEXT, $IMAGE,             │
│            $LABELS.key, $ANNOTATIONS.key                    │
│                                                             │
│ Press Esc to close                                          │
└─────────────────────────────────────────────────────────────┘
```

### Plugin Configuration

```yaml title="~/.config/k13d/plugins.yaml"
plugins:
  dive:
    shortCut: "Ctrl-I"
    description: "Dive into container image layers"
    scopes: [pods]
    command: dive
    args: [$IMAGE]

  debug:
    shortCut: "Shift-D"
    description: "Debug pod with ephemeral container"
    scopes: [pods]
    command: kubectl
    args: [debug, -n, $NAMESPACE, $NAME, -it, --image=busybox]
    confirm: true

  port-forward:
    shortCut: "Shift-F"
    description: "Port-forward to localhost:8080"
    scopes: [pods, services]
    command: kubectl
    args: [port-forward, -n, $NAMESPACE, $NAME, "8080:80"]
    background: true
    confirm: true

  stern:
    shortCut: "Shift-S"
    description: "Stream logs with stern (by app label)"
    scopes: [deployments]
    command: stern
    args: [-n, $NAMESPACE, $LABELS.app]

  lens:
    shortCut: "Ctrl-O"
    description: "Open in Lens"
    scopes: ["*"]
    command: lens
    args: [--context, $CONTEXT]
    background: true
```

### Execution Modes

| Mode | Behavior | Use Case |
|------|----------|----------|
| **Foreground** (`background: false`) | Suspends TUI, command gets full terminal | Interactive tools: `dive`, `kubectl exec` |
| **Background** (`background: true`) | Runs silently, TUI continues | External windows: Lens, port-forward |

### Confirmation

Plugins with `confirm: true` show a modal before execution, displaying the expanded command with all variables resolved. This is recommended for destructive or resource-intensive operations.

### Available Variables

| Variable | Description |
|----------|-------------|
| `$NAMESPACE` | Resource namespace |
| `$NAME` | Resource name |
| `$CONTEXT` | Current Kubernetes context |
| `$IMAGE` | Container image (pods only) |
| `$LABELS.key` | Label value by key |
| `$ANNOTATIONS.key` | Annotation value by key |

See the [Configuration Guide](../getting-started/configuration.md#plugins-pluginsyaml) for the full reference.

---

## YAML Viewer

View resource YAML manifests with syntax highlighting. Press `y` on any selected resource to open.

### Features

| Feature | Description |
|---------|-------------|
| **Syntax Highlighting** | Color-coded YAML with Tokyo Night theme |
| **Vim-Style Scrolling** | Navigate with `j`/`k`, `g`/`G`, `Ctrl-f`/`Ctrl-b` |
| **Line Wrapping** | Press `w` to toggle line wrap |
| **Search** | Press `/` to search within YAML content |
| **Exit** | Press `Esc` or `q` to return to resource table |

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll down / up |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `Ctrl+f` | Page down |
| `Ctrl+b` | Page up |
| `w` | Toggle line wrap |
| `/` | Search |
| `Esc` / `q` | Close viewer |

---

## Log Viewer

Real-time log streaming with ANSI color support. Press `l` on a pod to open.

### Features

| Feature | Description |
|---------|-------------|
| **Real-time Streaming** | Auto-scroll with new log entries as they arrive |
| **ANSI Color Support** | Full color rendering for application logs |
| **Container Selection** | Prompt to choose container for multi-container pods |
| **Previous Logs** | Press `Shift+L` to view logs from crashed/restarted containers |
| **Follow Mode** | Toggle auto-follow with `f` (enabled by default) |
| **Line Wrap** | Toggle line wrapping with `w` for long log lines |
| **Search** | Press `/` to search within log output |
| **Download** | Log files can be downloaded with pod name and timestamp in filename |

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `f` | Toggle follow mode (auto-scroll) |
| `w` | Toggle line wrap |
| `/` | Search within logs |
| `g` | Jump to beginning |
| `G` | Jump to end |
| `Ctrl+f` | Page down |
| `Ctrl+b` | Page up |
| `Esc` | Exit log viewer |

### Multi-Container Pods

When a pod has multiple containers, k13d displays a container selector before streaming logs. Select the desired container with `j`/`k` and press `Enter`.

---

## Terminal/Shell

Execute commands directly inside pod containers. Press `x` on a pod to open a shell session.

### Features

| Feature | Description |
|---------|-------------|
| **Shell Detection** | Automatically tries `/bin/bash` first, falls back to `/bin/sh` |
| **Container Selection** | For multi-container pods, prompts to select a container |
| **Full Terminal** | TUI suspends to provide full terminal access with stdin/stdout/stderr |
| **Exit** | Type `exit` or press `Ctrl+D` to return to k13d |

### How It Works

1. Press `x` on a selected pod
2. If multi-container, select the target container
3. TUI suspends and opens an interactive shell
4. After exiting the shell, TUI restores automatically

---

## Port Forward

Forward container ports to your local machine. Press `p` on a pod to start port forwarding.

### Features

| Feature | Description |
|---------|-------------|
| **Port Configuration** | Specify local and container ports via input dialog |
| **Background Running** | Port forwarding runs in the background while you continue using k13d |
| **Status Indicator** | Shows active/stopped status for forwarded ports |
| **Multiple Forwards** | Run multiple port forwards simultaneously |
| **Stop** | Press `q` or `Esc` to stop the selected port forward |

### Usage

1. Select a pod and press `p`
2. Enter the port mapping (e.g., `8080:80` for local 8080 → container 80)
3. Port forwarding starts in the background
4. Access the service at `http://localhost:8080`

---

## Context Switching

Switch between Kubernetes clusters directly from the TUI.

### Usage

- Type `:context` or `:ctx` to open the context switcher
- A modal displays all available contexts from your kubeconfig
- Current context is marked with `*`
- Select a context and press `Enter` to switch

### What Happens on Switch

1. Kubernetes client reconnects to the new cluster
2. Namespace selector resets and reloads available namespaces
3. All resource data refreshes for the new cluster
4. Status bar updates to show the new context name

---

## Resource Highlighting

Visual indicators for resource status using the Tokyo Night color scheme.

| Color | Status | Examples |
|-------|--------|----------|
| Green | Healthy | Running, Ready, Active, Bound |
| Yellow | Warning | Pending, Progressing, Waiting |
| Red | Error | Failed, Error, CrashLoopBackOff, ImagePullBackOff |
| Blue | Selected | Currently highlighted row |
| Gray | Inactive | Terminated, Completed, Succeeded |

---

## Help System

Built-in help for keybindings and commands. Press `?` at any time to view.

### Help Modal

The help modal shows context-sensitive keyboard shortcuts:

- **Global shortcuts**: Available everywhere (navigation, quit, help)
- **Resource actions**: Available when a resource is selected (YAML, describe, delete)
- **Resource-specific**: Actions specific to the current resource type (logs for pods, scale for deployments)
- **AI shortcuts**: AI-related actions (analyze, explain)
- **Plugin shortcuts**: Custom plugin shortcuts for the current resource scope

| Key | Action |
|-----|--------|
| `?` | Show help modal |
| `j` / `k` | Scroll within help |
| `Esc` / `q` | Close help |

![TUI Help Modal](../images/tui_help.png)

---

## i18n Support

Multi-language interface support for the TUI and AI responses.

### Supported Languages

| Language | Code | Description |
|----------|------|-------------|
| English | `en` | Default language |
| 한국어 (Korean) | `ko` | Full TUI + AI response translation |
| 中文 (Chinese) | `zh` | Full TUI + AI response translation |
| 日本語 (Japanese) | `ja` | Full TUI + AI response translation |

### Configuration

```yaml title="~/.config/k13d/config.yaml"
language: ko  # Korean
```

### What Gets Translated

- **TUI interface**: Menu labels, help text, status messages, keyboard shortcut descriptions
- **AI responses**: When the language is set to non-English, the AI assistant automatically responds in the configured language
- **Error messages**: Common error messages and warnings

### Fallback Behavior

If a translation key is not found for the configured language, k13d falls back to English. If the English translation is also missing, the raw key name is displayed.

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
