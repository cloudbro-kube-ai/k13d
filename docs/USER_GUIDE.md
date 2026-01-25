# k13d User Guide

Welcome to the **k13d** user guide! This document will help you master the ultimate Kubernetes AI Explorer with **k9s-compatible key bindings**.

## Core Navigation (k9s Compatible)

k13d uses Vim-style navigation with k9s-compatible shortcuts to keep your hands on the keyboard.

| Key | Action |
|-----|--------|
| `j` / `k` or `↑` / `↓` | Move selection up/down |
| `g` | Jump to top of list |
| `G` | Jump to bottom of list |
| `Ctrl+U` | Page up (10 rows) |
| `Ctrl+F` | Page down (10 rows) |
| `Enter` | Drill down to related resources |
| `Esc` | Go back to previous view |
| `Tab` | Switch focus to AI Assistant |
| `0-9` | Quick namespace switch |

### Resource Drill-Down (Enter Key)

When you press `Enter` on a resource, k13d navigates to related resources:

| From Resource | Navigates To |
|---------------|--------------|
| Service | Pods (matching selector) |
| Deployment | Pods |
| ReplicaSet | Pods |
| StatefulSet | Pods |
| DaemonSet | Pods |
| Job | Pods |
| CronJob | Jobs |
| Node | Pods running on node |
| Namespace | Switch to namespace, show Pods |
| Pod | Logs view |

Press `Esc` to go back to the previous view (navigation history is maintained).

## Resource Commands

Switch between resources using the command bar (`:` prefix):

| Command | Resource |
|---------|----------|
| `:pods` or `:po` | Pods |
| `:deploy` or `:deployments` | Deployments |
| `:svc` or `:services` | Services |
| `:ds` or `:daemonsets` | DaemonSets |
| `:sts` or `:statefulsets` | StatefulSets |
| `:rs` or `:replicasets` | ReplicaSets |
| `:jobs` or `:job` | Jobs |
| `:cj` or `:cronjobs` | CronJobs |
| `:hpa` | Horizontal Pod Autoscalers |
| `:netpol` or `:networkpolicies` | Network Policies |
| `:cm` or `:configmaps` | ConfigMaps |
| `:sec` or `:secrets` | Secrets |
| `:ing` or `:ingresses` | Ingresses |
| `:pv` or `:persistentvolumes` | Persistent Volumes |
| `:pvc` or `:persistentvolumeclaims` | Persistent Volume Claims |
| `:sc` or `:storageclasses` | Storage Classes |
| `:sa` or `:serviceaccounts` | Service Accounts |
| `:roles` or `:role` | Roles |
| `:rb` or `:rolebindings` | Role Bindings |
| `:cr` or `:clusterroles` | Cluster Roles |
| `:crb` or `:clusterrolebindings` | Cluster Role Bindings |
| `:nodes` or `:no` | Nodes |
| `:ns` or `:namespaces` | Namespaces |
| `:ctx` or `:context` | Kubernetes Contexts |
| `:events` or `:ev` | Events |
| `:crd` | Custom Resource Definitions |
| `:pdb` | Pod Disruption Budgets |
| `:quota` | Resource Quotas |
| `:limits` | Limit Ranges |
| `:ep` | Endpoints |

## Dashboard Actions (k9s Compatible)

### General Actions

| Key | Action |
|-----|--------|
| `d` | Describe resource (detailed info like `kubectl describe`) |
| `y` | View YAML manifest |
| `e` | Edit resource in $EDITOR |
| `/` | Filter current table (supports regex: `/pattern/`) |
| `r` | Refresh current view |
| `c` | Switch Kubernetes context |
| `?` | Show help |
| `q` | Quit |

### Pod Actions

| Key | Action |
|-----|--------|
| `l` | View logs |
| `p` | View previous container logs |
| `s` | Shell into Pod (`/bin/bash` or `/bin/sh`) |
| `a` | Attach to container |
| `o` | Show node where pod is running |
| `k` or `Ctrl+K` | Kill (force delete) pod |
| `Shift+F` | Port forward |

### Workload Actions (Deployments, StatefulSets, DaemonSets)

| Key | Action |
|-----|--------|
| `Shift+S` | Scale replicas |
| `Shift+R` | Rollout restart |
| `z` | Show related resources (ReplicaSets for Deployments) |

### CronJob Actions

| Key | Action |
|-----|--------|
| `t` | Trigger (manually create job from cronjob) |

### Namespace Actions

| Key | Action |
|-----|--------|
| `u` | Use namespace (switch to selected namespace) |

### Dangerous Actions

| Key | Action |
|-----|--------|
| `Ctrl+D` | Delete resource (with confirmation) |

## Multi-Select

Select multiple resources for bulk operations:

| Key | Action |
|-----|--------|
| `Space` | Toggle selection on current row |
| `Ctrl+Space` | Clear all selections |

Selected rows are marked with `●` and highlighted in cyan.

## Filtering

### Substring Filter
Type `/` followed by text to filter:
```
/nginx
```

### Regex Filter
Use `/pattern/` for regex matching:
```
/nginx-[0-9]+/
/^api-.*/
```

## AI Assistant Synergy

The AI Assistant is fully integrated with the Dashboard.

### Asking Questions

Press `Tab` to switch focus to the AI input field. You can ask questions like:
- "Why is this pod failing?"
- "How do I scale this deployment?"
- "Explain what this resource does"

### Decision Required

When the AI suggests kubectl commands that modify resources, k13d shows a **Decision Required** prompt:

```
━━━ DECISION REQUIRED ━━━

? [1] Confirm: kubectl scale deployment nginx --replicas=5
⚠ [2] DANGEROUS: kubectl delete pod nginx-abc123 --force
   • This command will delete resources
   • Force flag may cause data loss

Press 1-9 to execute, A to execute all, Esc to cancel
```

- **Yellow `?`**: Command requires confirmation
- **Red `⚠`**: Dangerous command with warnings
- Press **1-9** to execute a specific command
- Press **A** to execute all commands (with extra confirmation for dangerous ones)
- Press **Esc** to cancel all

### Command Safety Analysis

The AI automatically analyzes suggested commands:

| Category | Examples | Behavior |
|----------|----------|----------|
| Read-only | `get`, `describe`, `logs` | No confirmation needed |
| Write | `apply`, `create`, `scale` | Confirmation required |
| Dangerous | `delete --all`, `drain --force` | Extra warnings shown |
| Interactive | `exec -it`, `port-forward` | Not auto-executed |

### Jumping to Dashboard

If the AI mentions a resource, you can jump to it using the command bar:
- `:view pods nginx-bot-123 my-namespace`

## Web UI Mode

Start the web server with:
```bash
# Basic web mode
k13d -web -port 8080

# With authentication options
k13d -web -port 8080 --auth-mode local --admin-user admin --admin-password secret

# Token-based auth (Kubernetes Dashboard style)
k13d -web -port 8080 --auth-mode token

# Disable authentication (development only)
k13d -web -port 8080 --no-auth
```

### Authentication Modes

| Mode | Description | Best For |
|------|-------------|----------|
| `token` | Kubernetes ServiceAccount token | In-cluster deployments |
| `local` | Local username/password | Development, small teams |
| `ldap` | LDAP/Active Directory | Enterprise environments |

### Web UI Features

| Feature | Description |
|---------|-------------|
| **SSE Streaming Chat** | AI responses stream in real-time with a blinking cursor |
| **Auto-Refresh** | Automatically refresh resource data at configurable intervals |
| **Manual Refresh** | Click the refresh button for immediate data update |
| **Settings Panel** | Configure streaming, auto-refresh, LLM provider, and more |

### Auto-Refresh Controls

Located in the header bar:
- **Refresh Button** (circular arrow): Click for immediate refresh
- **Auto Toggle**: Enable/disable automatic refresh
- **Interval Selector**: Choose refresh interval (10s, 30s, 1m, 2m, 5m)
- **Last Update Time**: Shows when data was last refreshed

### Settings (Web UI)

Access settings via the **Settings** button in the header:

**General Tab:**
- **Language**: English or Korean
- **Log Level**: Debug, Info, Warning, Error
- **Enable Streaming**: Toggle SSE streaming for AI chat
- **Auto Refresh**: Toggle and interval configuration

**LLM Tab:**
- Provider selection (OpenAI, Ollama, Azure OpenAI, AWS Bedrock)
- Model name
- Endpoint URL
- API Key
- Connection test button

**Authentication Tab (Admin only):**
- LDAP/Active Directory settings
- SSO/OAuth configuration
- User management

### Using Local LLM (Ollama)

For air-gapped environments or offline use:

```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull a recommended model
ollama pull qwen2.5:3b

# Configure k13d
cat > ~/.config/k13d/config.yaml << EOF
llm:
  provider: ollama
  model: qwen2.5:3b
  endpoint: http://localhost:11434/v1
EOF
```

## Settings & Customization

Type `:health` or `:status` to check system status including:
- Kubernetes connectivity
- AI provider status
- Current configuration

Configuration is stored in `~/.kube-ai-dashboard/config.yaml`. See the [Configuration Guide](CONFIGURATION_GUIDE.md) for details.

## Auditing

Internal actions and AI tool calls are recorded in the Audit Log for transparency and security. View it by typing `:audit`.

---

## Quick Reference Card

```
Navigation          Resource Actions      AI Assistant
─────────────────   ─────────────────     ─────────────
↑/↓ or j/k: Move    d: Describe           Tab: Focus AI
g/G: Top/Bottom     y: YAML               Enter: Send
Enter: Drill down   e: Edit               Esc: Cancel
Esc: Go back        l: Logs (pods)        1-9: Execute cmd
Ctrl+U/F: Page      s: Shell (pods)       A: Execute all
0-9: Namespace      Ctrl+D: Delete
/: Filter           S: Scale
:: Command          R: Restart
?: Help             F: Port-forward
q: Quit             t: Trigger (cj)
```

---

Need more help? Check out our [Support Policy](../SUPPORT.md) or join our community discussions!
