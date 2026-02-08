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
