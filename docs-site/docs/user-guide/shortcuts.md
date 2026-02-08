# Keyboard Shortcuts

Complete reference for k13d keyboard shortcuts in both TUI and Web interfaces.

## TUI Keyboard Shortcuts

### Global

| Key | Action | Description |
|-----|--------|-------------|
| ++question++ | Help | Show help modal |
| ++ctrl+c++ | Quit | Exit k13d |
| ++esc++ | Cancel | Close modal / Cancel action |
| ++colon++ | Command | Enter command mode |
| ++slash++ | Filter | Start filtering |

### Navigation

| Key | Action | Description |
|-----|--------|-------------|
| ++j++ | Down | Move selection down |
| ++k++ | Up | Move selection up |
| ++arrow-down++ | Down | Move selection down |
| ++arrow-up++ | Up | Move selection up |
| ++g++ | First | Go to first item |
| ++shift+g++ | Last | Go to last item |
| ++ctrl+f++ | Page Down | Scroll page down |
| ++ctrl+b++ | Page Up | Scroll page up |
| ++ctrl+d++ | Half Down | Scroll half page down |
| ++ctrl+u++ | Half Up | Scroll half page up |

### Panel Navigation

| Key | Action | Description |
|-----|--------|-------------|
| ++tab++ | Next Panel | Switch to next panel |
| ++shift+tab++ | Prev Panel | Switch to previous panel |
| ++arrow-left++ | Left Panel | Focus left panel |
| ++arrow-right++ | Right Panel | Focus right panel |
| ++1++ - ++9++ | Panel N | Quick switch to panel N |

### Resource Actions

#### General

| Key | Action | Description |
|-----|--------|-------------|
| ++y++ | YAML | View resource YAML |
| ++d++ | Describe | Describe resource |
| ++e++ | Edit | Edit resource in $EDITOR |
| ++ctrl+d++ | Delete | Delete resource (confirm) |
| ++enter++ | Details | Show resource details |

#### Pods

| Key | Action | Description |
|-----|--------|-------------|
| ++l++ | Logs | View pod logs |
| ++shift+l++ | Prev Logs | View previous container logs |
| ++x++ | Exec | Execute shell in pod |
| ++p++ | Port Forward | Start port forwarding |
| ++f++ | Follow | Follow logs |

#### Deployments

| Key | Action | Description |
|-----|--------|-------------|
| ++s++ | Scale | Scale replicas |
| ++r++ | Restart | Rollout restart |
| ++h++ | History | View rollout history |
| ++u++ | Undo | Rollback to previous |

#### Nodes

| Key | Action | Description |
|-----|--------|-------------|
| ++c++ | Cordon | Mark unschedulable |
| ++shift+c++ | Uncordon | Mark schedulable |
| ++shift+d++ | Drain | Drain node |

### AI Assistant

| Key | Action | Description |
|-----|--------|-------------|
| ++shift+l++ | AI Analyze | Deep analysis of resource |
| ++h++ | Explain | Explain this resource |
| ++enter++ | Send | Send message to AI |
| ++y++ | Approve | Approve tool request |
| ++n++ | Reject | Reject tool request |
| ++a++ | Always | Always approve read-only |

### Command Mode

| Command | Description |
|---------|-------------|
| `:pods` | View pods |
| `:svc` | View services |
| `:deploy` | View deployments |
| `:rs` | View replica sets |
| `:sts` | View statefulsets |
| `:ds` | View daemonsets |
| `:jobs` | View jobs |
| `:cronjobs` | View cronjobs |
| `:cm` | View configmaps |
| `:secrets` | View secrets |
| `:ingress` | View ingresses |
| `:pv` | View persistent volumes |
| `:pvc` | View persistent volume claims |
| `:nodes` | View nodes |
| `:ns` | View namespaces |
| `:events` | View events |
| `:helm` | View Helm releases |
| `:ns <name>` | Switch namespace |
| `:ctx <name>` | Switch context |
| `:q` | Quit |
| `:help` | Show help |

### Management Commands

| Command | Description |
|---------|-------------|
| `:alias` | View all resource aliases (built-in + custom) |
| `:model` | Open AI model profile selector |
| `:model <name>` | Switch directly to named model profile |
| `:plugins` | View available plugins with shortcuts |
| `:health` | Check system status |
| `:audit` | View audit log |

### Filter Mode

| Key | Action |
|-----|--------|
| Any text | Filter pattern |
| ++enter++ | Apply filter |
| ++esc++ | Clear filter |

## Web Keyboard Shortcuts

### Global

| Key | Action | Description |
|-----|--------|-------------|
| ++ctrl+k++ | Search | Focus global search |
| ++ctrl+slash++ | AI Panel | Toggle AI assistant |
| ++esc++ | Close | Close modal/panel |
| ++question++ | Help | Show shortcuts |

### Navigation

| Key | Action | Description |
|-----|--------|-------------|
| ++arrow-down++ | Down | Move selection down |
| ++arrow-up++ | Up | Move selection up |
| ++enter++ | Select | Open selected item |
| ++ctrl+enter++ | New Tab | Open in new tab |

### Resource Actions

| Key | Action | Description |
|-----|--------|-------------|
| ++ctrl+y++ | YAML | View YAML |
| ++ctrl+d++ | Describe | Describe resource |
| ++delete++ | Delete | Delete resource |
| ++ctrl+e++ | Edit | Edit resource |

### AI Assistant

| Key | Action | Description |
|-----|--------|-------------|
| ++enter++ | Send | Send message |
| ++shift+enter++ | Newline | Add newline |
| ++ctrl+l++ | Clear | Clear chat |

### Quick Access

| Key | Action |
|-----|--------|
| ++ctrl+1++ | Dashboard |
| ++ctrl+2++ | Pods |
| ++ctrl+3++ | Services |
| ++ctrl+4++ | Deployments |
| ++ctrl+5++ | ConfigMaps |

## Vim-Style Bindings

k13d supports Vim-style navigation:

| Vim | k13d Action |
|-----|-------------|
| `j` | Move down |
| `k` | Move up |
| `gg` | Go to first |
| `G` | Go to last |
| `/` | Search/filter |
| `n` | Next match |
| `N` | Previous match |
| `:` | Command mode |
| `q` | Quit |

## Customization

### Custom Keybindings

Configure in `~/.config/k13d/config.yaml`:

```yaml
keybindings:
  # Override default bindings
  describe: "i"           # Use 'i' instead of 'd'
  yaml: "v"               # Use 'v' instead of 'y'

  # Custom actions
  custom:
    - key: "ctrl+shift+p"
      action: "port-forward"
    - key: "ctrl+shift+l"
      action: "logs-follow"
```

### Disable Shortcuts

```yaml
keybindings:
  disable:
    - "ctrl+d"  # Disable delete shortcut
```

## Quick Reference Card

### Essential Shortcuts

```
┌──────────────────────────────────────────┐
│           k13d Quick Reference            │
├──────────────────────────────────────────┤
│ Navigation                               │
│   j/k          Move up/down              │
│   Tab          Switch panel              │
│   :            Command mode              │
│   /            Filter                    │
│                                          │
│ Actions                                  │
│   y            View YAML                 │
│   d            Describe                  │
│   l            Logs                      │
│   x            Exec                      │
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

## Next Steps

- [TUI Dashboard](tui.md) - Full TUI guide
- [Web Dashboard](web.md) - Full Web guide
- [Configuration](../getting-started/configuration.md) - Customize keybindings
