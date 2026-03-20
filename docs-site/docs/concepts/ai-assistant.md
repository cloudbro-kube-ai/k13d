# AI Assistant

k13d integrates an intelligent AI assistant that helps you manage Kubernetes clusters with natural language commands and intelligent analysis.

!!! note "Config path note"
    This page uses `~/.config/k13d/...` examples when showing `config.yaml`. That is now also the default on macOS.

## Overview

The AI assistant provides:

- **Natural Language Commands**: Ask questions in plain English
- **Context-Aware Analysis**: AI has access to YAML, events, and logs
- **Tool Integration**: Uses a kubectl-first toolset, with bash and MCP tools as opt-in extensions
- **Safety Checks**: Required Decision for allowed changes, hard blocks for unsupported interactive commands
- **Beginner Mode**: Simple explanations for complex resources

## How It Works

```
┌─────────────────────────────────────────────────────────────────┐
│                    User Question                                  │
│             "Why is my nginx pod failing?"                        │
└─────────────────────────────┬───────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    AI Agent                                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │ Understand  │─►│ Plan Tools  │─►│   Execute   │              │
│  │  Context    │  │   to Use    │  │    Tools    │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└─────────────────────────────┬───────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
        ┌──────────┐   ┌──────────┐   ┌──────────┐
        │ kubectl  │   │   bash   │   │   MCP    │
        │  tools   │   │  tools   │   │  tools   │
        └──────────┘   └──────────┘   └──────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    AI Response                                    │
│  "The nginx pod is failing because the image 'nginx:invalid'    │
│   cannot be pulled. The error indicates 'ImagePullBackOff'.     │
│   Would you like me to fix this by updating the image tag?"     │
└─────────────────────────────────────────────────────────────────┘
```

## Features

### Context-Aware Analysis

When you select a resource and ask the AI to analyze it:

1. **YAML Manifest** - Full resource definition
2. **Events** - Recent Kubernetes events
3. **Logs** - Container logs (for Pods)
4. **Metrics** - Resource usage data

### Built-in Tools

| Tool | Default | Description | Examples |
|------|---------|-------------|----------|
| `kubectl` | On | Execute kubectl commands | `kubectl get pods`, `kubectl describe` |
| `bash` | Off | Run shell commands as a last resort | `date`, `grep`, file operations |
| `MCP` | Off | External tool integration from configured MCP servers | GitHub, databases, custom tools |

### Safety Features

Commands are classified by risk level:

| Level | Action | Examples |
|-------|--------|----------|
| **Read-only** | Allowed, but `Decision Required` by default | `kubectl get`, `describe` |
| **Write** | Requires approval by default | `kubectl apply`, `create`, `scale` |
| **Dangerous** | Warning + approval, or full block if configured | `kubectl delete`, `drain` |
| **Hard-blocked** | Not approvable | `kubectl edit`, `kubectl port-forward`, `kubectl exec -it`, bash-wrapped `kubectl` |

## Usage

### TUI Mode

1. Press ++tab++ to focus on the AI Panel
2. Type your question
3. Press ++enter++ to send
4. Review tool approval requests (Y/N)

### Web Mode

1. Click the AI Assistant panel
2. Type your question in the input field
3. Click "Send" or press ++enter++
4. Approve/reject tool requests in the popup

## Example Conversations

### Troubleshooting

```
You: Why is my nginx deployment not starting?

AI: Let me check your nginx deployment...
[Tool: kubectl get deployment nginx -o yaml]
[Tool: kubectl get events --field-selector involvedObject.name=nginx]

The nginx deployment has 0/3 replicas ready. The issue is:
- Image pull error: "nginx:v999" does not exist
- All pods are in "ImagePullBackOff" state

To fix this, you can update the image to a valid tag:
kubectl set image deployment/nginx nginx=nginx:latest
```

### Resource Explanation

```
You: Explain this HPA configuration

AI: This HorizontalPodAutoscaler (HPA) is configured to:
- Target: frontend deployment
- Min replicas: 2
- Max replicas: 10
- Scale trigger: CPU utilization > 50%
- Scale-down stabilization: 300 seconds

This means the HPA will add pods when CPU usage exceeds 50%
and remove pods (with a 5-minute delay) when usage drops.
```

### Quick Actions

```
You: Scale my api deployment to 5 replicas

AI: I'll scale the api deployment to 5 replicas.
[Approval Required: kubectl scale deployment api --replicas=5]
```

## Configuration

### LLM Provider Settings

Configure your LLM provider in `~/.config/k13d/config.yaml`:

```yaml
llm:
  provider: openai
  model: gpt-4
  api_key: ${OPENAI_API_KEY}
  enable_bash_tool: false
  enable_mcp_tools: false

# Or use Ollama for local LLM
llm:
  provider: ollama
  model: llama3.2
  endpoint: http://localhost:11434
```

### Beginner Mode

Enable beginner mode for simpler explanations:

```yaml
beginner_mode: true
```

## Best Practices

1. **Be Specific**: "Check nginx pod logs for errors" is better than "check pods"
2. **Provide Context**: Include namespace and resource names
3. **Review Approvals**: Always review tool commands before approving
4. **Expect kubectl-first behavior**: Turn on bash or MCP only when you truly need them
5. **Use Beginner Mode**: Enable for learning or simpler explanations

## Next Steps

- [LLM Providers](../ai-llm/providers.md) - Configure different AI providers
- [Tool Calling](../ai-llm/tool-calling.md) - How AI executes commands
- [MCP Integration](mcp-integration.md) - Extend AI capabilities
