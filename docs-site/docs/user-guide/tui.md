# TUI Dashboard

The Terminal User Interface (TUI) provides a k9s-like experience for managing Kubernetes clusters directly from your terminal.

## Overview

```
┌─────────────────────────────────────────────────────────────────┐
│ k13d │ Context: minikube │ Namespace: default │ ? for help     │
├─────────────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────┐ ┌─────────────────────────┐ │
│ │ Pods (8)                        │ │ AI Assistant            │ │
│ │                                 │ │                         │ │
│ │ NAME            READY  STATUS   │ │ Ask me anything about   │ │
│ │ nginx-abc123    1/1    Running  │ │ your cluster...         │ │
│ │ api-def456      2/2    Running  │ │                         │ │
│ │ db-ghi789       1/1    Running  │ │ > Why is nginx failing? │ │
│ │                                 │ │                         │ │
│ └─────────────────────────────────┘ └─────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│ :pods   Filter: /nginx                                          │
└─────────────────────────────────────────────────────────────────┘
```

## Getting Started

### Launch TUI Mode

```bash
# Default mode
k13d

# With specific kubeconfig
k13d --kubeconfig ~/.kube/my-cluster

# With specific context
k13d --context production
```

## Navigation

### Basic Navigation

| Key | Action |
|-----|--------|
| ++j++ / ++arrow-down++ | Move down |
| ++k++ / ++arrow-up++ | Move up |
| ++tab++ | Switch focus between panels |
| ++esc++ | Close modal / Return to main view |
| ++q++ | Quit (with confirmation) |

### Resource Navigation

| Key | Action |
|-----|--------|
| ++colon++ | Command mode (e.g., `:pods`, `:svc`) |
| ++slash++ | Filter current table |
| ++g++ | Go to first item |
| ++shift+g++ | Go to last item |
| ++ctrl+f++ | Page down |
| ++ctrl+b++ | Page up |

### Panel Focus

| Key | Action |
|-----|--------|
| ++tab++ | Cycle focus between panels |
| ++arrow-left++ / ++arrow-right++ | Switch panels |
| ++1++ - ++9++ | Quick switch to specific panel |

## Resource Commands

### Viewing Resources

| Command | Description |
|---------|-------------|
| `:pods` | View pods |
| `:svc` or `:services` | View services |
| `:deploy` or `:deployments` | View deployments |
| `:rs` or `:replicasets` | View replica sets |
| `:sts` or `:statefulsets` | View stateful sets |
| `:ds` or `:daemonsets` | View daemon sets |
| `:jobs` | View jobs |
| `:cronjobs` | View cron jobs |
| `:cm` or `:configmaps` | View config maps |
| `:secrets` | View secrets |
| `:ingress` | View ingresses |
| `:pv` | View persistent volumes |
| `:pvc` | View persistent volume claims |
| `:nodes` | View nodes |
| `:ns` or `:namespaces` | View namespaces |
| `:events` | View events |
| `:helm` | View Helm releases |

### Namespace Commands

| Command | Description |
|---------|-------------|
| `:ns default` | Switch to default namespace |
| `:ns kube-system` | Switch to kube-system |
| `:ns all` | View all namespaces |

### Management Commands

| Command | Description |
|---------|-------------|
| `:alias` | View all configured resource aliases |
| `:model` | Open AI model profile selector |
| `:model <name>` | Switch directly to a named model profile |
| `:plugins` | View available plugins with shortcuts |
| `:health` | Check system status |
| `:audit` | View audit log |

## Autocomplete

When typing a command, k13d provides smart autocomplete:

- **Single match**: Dimmed hint text appears (press ++tab++ to accept)
- **Multiple matches**: Dropdown overlay appears above command bar
  - ++arrow-up++ / ++arrow-down++ to navigate
  - ++tab++ or ++enter++ to select
  - ++esc++ to dismiss
  - Custom aliases from `aliases.yaml` included in results

## Resource Aliases

Define custom command shortcuts in `~/.config/k13d/aliases.yaml`:

```yaml title="~/.config/k13d/aliases.yaml"
aliases:
  pp: pods
  dep: deployments
  svc: services
  sec: secrets
  cm: configmaps
  ds: daemonsets
  sts: statefulsets
```

### Usage

- Type `:pp` to navigate to Pods (alias resolves automatically)
- Type `:alias` to see all active aliases (built-in + custom)
- Custom aliases integrate with autocomplete — type `:p` and press ++tab++
- If a custom alias conflicts with a built-in, the custom alias wins

## Resource Actions

### General Actions

| Key | Action | Description |
|-----|--------|-------------|
| ++y++ | YAML | View resource YAML |
| ++d++ | Describe | Describe resource |
| ++e++ | Edit | Edit resource (opens $EDITOR) |
| ++ctrl+d++ | Delete | Delete resource (with confirmation) |

### Pod-Specific Actions

| Key | Action | Description |
|-----|--------|-------------|
| ++l++ | Logs | Stream pod logs |
| ++shift+l++ | Previous Logs | View previous container logs |
| ++x++ | Exec | Open shell in container |
| ++p++ | Port Forward | Start port forwarding |

### Deployment Actions

| Key | Action | Description |
|-----|--------|-------------|
| ++s++ | Scale | Scale replicas |
| ++r++ | Restart | Rollout restart |
| ++h++ | History | View rollout history |
| ++u++ | Undo | Rollback to previous version |

### Node Actions

| Key | Action | Description |
|-----|--------|-------------|
| ++c++ | Cordon | Mark node unschedulable |
| ++shift+c++ | Uncordon | Mark node schedulable |
| ++shift+d++ | Drain | Drain node |

## AI Assistant

### Using the AI Panel

1. Press ++tab++ to focus on the AI panel
2. Type your question
3. Press ++enter++ to send
4. View the response in the output area

### Chat History

Previous Q&A sessions are preserved within each TUI session. Scroll up to review past conversations, separated by visual dividers.

### Model Switching

Switch AI models on the fly:

- Type `:model` to see all configured profiles
- Type `:model gpt-4o` to switch directly
- Active model marked with `*` in selector

### AI Actions on Resources

| Key | Action | Description |
|-----|--------|-------------|
| ++shift+l++ | AI Analyze | Deep analysis of selected resource |
| ++h++ | Explain This | Get explanation of resource |

### Tool Approval

When AI requests to execute a command:

```
┌─────────────────────────────────────────┐
│ Tool Approval Required                   │
│                                         │
│ kubectl get pods -n production          │
│                                         │
│ [Y] Approve  [N] Reject  [A] Always     │
└─────────────────────────────────────────┘
```

| Key | Action |
|-----|--------|
| ++y++ | Approve this command |
| ++n++ | Reject this command |
| ++a++ | Always approve read-only commands |

## Filtering and Search

### Filter Syntax

```
/nginx           # Contains "nginx"
/^nginx          # Starts with "nginx"
/nginx$          # Ends with "nginx"
/nginx.*running  # Regex pattern
```

### Quick Filter

| Key | Action |
|-----|--------|
| ++slash++ | Start filter mode |
| ++esc++ | Clear filter |
| ++enter++ | Apply filter |

## Status Bar

The status bar shows:

```
:pods [1/25] │ Filter: nginx │ Context: production │ NS: default
```

- Current resource type
- Selection index / total count
- Active filter
- Kubernetes context
- Current namespace

## Plugins

Extend k13d with external CLI tools via `~/.config/k13d/plugins.yaml`. Plugins bind keyboard shortcuts to commands that run with the selected resource's context.

### Quick Start

Create `~/.config/k13d/plugins.yaml`:

```yaml
plugins:
  dive:
    shortCut: "Ctrl-I"
    description: "Dive into container image layers"
    scopes: [pods]
    command: dive
    args: [$IMAGE]
```

Then in the TUI:

1. Navigate to Pods (`:pods`)
2. Select a pod with `j`/`k`
3. Press ++ctrl+i++ to launch `dive` with the pod's container image

### Using Plugins

| Action | How |
|--------|-----|
| View all plugins | Type `:plugins` |
| Run a plugin | Press the plugin's shortcut while on a matching resource |
| Confirm execution | Press `Execute` in the confirmation modal (if `confirm: true`) |

### Example Plugins

```yaml title="~/.config/k13d/plugins.yaml"
plugins:
  # Analyze container image layers
  dive:
    shortCut: "Ctrl-I"
    description: "Dive into container image layers"
    scopes: [pods]
    command: dive
    args: [$IMAGE]

  # Debug pod with ephemeral container
  debug:
    shortCut: "Shift-D"
    description: "Debug pod with ephemeral container"
    scopes: [pods]
    command: kubectl
    args: [debug, -n, $NAMESPACE, $NAME, -it, --image=busybox]
    confirm: true

  # Background port-forward
  port-forward:
    shortCut: "Shift-F"
    description: "Port-forward to localhost:8080"
    scopes: [pods, services]
    command: kubectl
    args: [port-forward, -n, $NAMESPACE, $NAME, "8080:80"]
    background: true
    confirm: true

  # Multi-pod log streaming
  stern:
    shortCut: "Shift-S"
    description: "Stream logs with stern (by app label)"
    scopes: [deployments]
    command: stern
    args: [-n, $NAMESPACE, $LABELS.app]

  # Open in external tool (all resources)
  lens:
    shortCut: "Ctrl-O"
    description: "Open in Lens"
    scopes: ["*"]
    command: lens
    args: [--context, $CONTEXT]
    background: true
```

!!! tip
    Plugins with `background: true` run without suspending the TUI. Use this for port-forwarding or opening external applications.

See [Configuration > Plugins](../getting-started/configuration.md#plugins-pluginsyaml) for the full reference including all available variables.

## Custom Hotkeys

Define custom keyboard shortcuts in `~/.config/k13d/hotkeys.yaml` to execute external commands:

```yaml title="~/.config/k13d/hotkeys.yaml"
hotkeys:
  stern-logs:
    shortCut: "Shift-L"
    description: "Stern multi-pod logs"
    scopes: [pods, deployments]
    command: stern
    args: ["-n", "$NAMESPACE", "$NAME"]

  port-forward-8080:
    shortCut: "Ctrl-P"
    description: "Port forward to 8080"
    scopes: [pods, services]
    command: kubectl
    args: [port-forward, -n, "$NAMESPACE", "$NAME", "8080:8080"]
```

Hotkeys support `$NAMESPACE`, `$NAME`, and `$CONTEXT` variable expansion. Set `dangerous: true` to require confirmation before execution.

## Per-Resource Sort Defaults

Configure default sort in `~/.config/k13d/views.yaml`:

```yaml title="~/.config/k13d/views.yaml"
views:
  pods:
    sortColumn: AGE
    sortAscending: false       # Newest first
  deployments:
    sortColumn: NAME
    sortAscending: true        # Alphabetical
  services:
    sortColumn: TYPE
    sortAscending: true
```

Sort preferences are applied automatically when navigating to each resource. k13d detects column types (numeric, age, ready format, string) for proper sorting.

## Context Switching

Switch between Kubernetes clusters:

1. Type `:context` or `:ctx`
2. Select from available contexts (current marked with `*`)
3. Press ++enter++ to switch

On switch, k13d reconnects to the new cluster, reloads namespaces, and refreshes all resource data.

## Customization

### Theme Configuration

k13d uses a Tokyo Night-inspired theme by default.

### Configuration File

```yaml
# ~/.config/k13d/config.yaml

ui:
  theme: tokyo-night  # or: light, dark, nord
  refresh_interval: 5s
  show_header: true
  show_status: true
```

## Keyboard Shortcuts Reference

### Global

| Key | Action |
|-----|--------|
| ++question++ | Show help |
| ++ctrl+c++ | Quit |
| ++esc++ | Cancel / Back |

### Navigation

| Key | Action |
|-----|--------|
| ++j++ / ++k++ | Up / Down |
| ++g++ / ++shift+g++ | First / Last |
| ++ctrl+f++ / ++ctrl+b++ | Page Down / Up |
| ++tab++ | Switch panel |

### Actions

| Key | Action |
|-----|--------|
| ++y++ | YAML |
| ++d++ | Describe |
| ++l++ | Logs |
| ++x++ | Exec |
| ++e++ | Edit |
| ++s++ | Scale |
| ++r++ | Restart |
| ++ctrl+d++ | Delete |

### AI

| Key | Action |
|-----|--------|
| ++shift+l++ | AI Analyze |
| ++h++ | Explain This |

## Tips and Tricks

### 1. Quick Namespace Switch

```
:ns default
:ns -          # Previous namespace
```

### 2. Resource Shortcuts

```
:po           # pods
:svc          # services
:deploy       # deployments
:no           # nodes
:alias        # view all aliases
:model        # switch AI model
:plugins      # view plugins
```

### 3. Filter + Action

```
/error        # Filter to errors
Shift+L       # AI analyze all matching
```

### 4. Vim-Style Navigation

If you're familiar with Vim:

- `j/k` for up/down
- `gg` for top
- `G` for bottom
- `/` for search

## Next Steps

- [Web Dashboard](web.md) - Web UI alternative
- [Keyboard Shortcuts](shortcuts.md) - Complete shortcut reference
- [AI Assistant](../concepts/ai-assistant.md) - AI features
