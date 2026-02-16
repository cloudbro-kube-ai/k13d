# AI Assistant Features

The k13d AI Assistant is an agentic AI that understands your Kubernetes cluster and can execute commands on your behalf.

---

## Overview

![AI Assistant Panel](../images/webui-assistant-pannel.png)

| Capability | Description |
|------------|-------------|
| **Natural Language** | Ask questions in plain language |
| **Tool Execution** | Runs kubectl, bash commands |
| **Context Awareness** | Sees YAML, Events, Logs |
| **Safety First** | Dangerous commands require approval |
| **Multi-language** | English, Korean, Chinese, Japanese |

---

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
```

---

## Tool Calling

### MCP Tool Call Debug Mode

![MCP Tool Call Debug](../images/webui-mcp-tool-call-debugmode.png)

Enable debug mode to see raw tool calls:

- Tool call requests
- Raw API responses
- Execution timing

### Available Tools

| Tool | Description | Examples |
|------|-------------|----------|
| **kubectl** | Kubernetes CLI | `kubectl get pods`, `kubectl describe` |
| **bash** | Shell commands | `date`, `grep`, file operations |
| **MCP** | External tools | GitHub, databases, custom tools |

### Safety Classification

| Level | Commands | Approval |
|-------|----------|----------|
| **Read-only** | get, describe, logs | Auto-approve |
| **Write** | apply, create, patch | Requires approval |
| **Dangerous** | delete, drain, taint | Warning + approval |

### AST-Based Analysis

The AI uses Abstract Syntax Tree parsing to detect:

- Piped commands (`kubectl get pods | xargs rm`)
- Command chains (`kubectl get pods && rm -rf /`)
- Redirects and subshells
- Dangerous patterns

---

## Approval Workflow

### Web UI Approval

![Web Approval Dialog](../images/webui-decision-required.png)

1. AI requests tool execution
2. Dialog shows exact command
3. Safety warning (if dangerous)
4. User clicks Approve or Reject
5. Command executes on approval

![Web Approval Detail](../images/webui-decision-required-2.png)

### TUI Approval

| Key | Action |
|-----|--------|
| `Y` | Approve this command |
| `N` | Reject this command |
| `A` | Always approve read-only |

---

## Context Awareness

### Resource Context

When you select a resource, the AI receives:

| Context | Description |
|---------|-------------|
| **YAML** | Full resource manifest |
| **Events** | Recent Kubernetes events |
| **Logs** | Container logs (for Pods) |
| **Metrics** | CPU/Memory usage |
| **Related** | Owner references, selectors |

### Example: Troubleshooting

```
User: "Why is this pod failing?"

AI receives:
- Pod YAML (spec, status, containers)
- Events (ImagePullBackOff, FailedScheduling)
- Logs (error messages, stack traces)
- Container status (restarts, exit codes)

AI responds with specific diagnosis.
```

---

## AI Conversation Example

### TUI AI Interaction

The TUI AI assistant supports the same capabilities as the Web UI:

- Natural language question input
- AI analysis with streaming responses
- Tool execution with approval workflow

---

## Beginner Mode

Simplified explanations for Kubernetes newcomers.

Enable in config:

```yaml
beginner_mode: true
```

| Normal Mode | Beginner Mode |
|-------------|---------------|
| "Pod in CrashLoopBackOff" | "Your container keeps crashing. This happens when the program inside has an error and Kubernetes keeps trying to restart it." |
| "OOMKilled" | "Your container ran out of memory. It tried to use more RAM than allowed and was stopped by the system." |

---

## Chat History (TUI)

In the TUI, AI conversations are preserved within each session:

- Previous Q&A sessions are kept above the latest response
- Separated by visual dividers (`────────────────────────────`)
- Scroll up in the AI panel to review past conversations
- History maintained for the duration of the TUI session

---

## Model Switching (TUI)

Switch between configured AI model profiles directly from the TUI:

| Command | Action |
|---------|--------|
| `:model` | Opens a modal with all configured profiles (active marked with `*`) |
| `:model gpt-4o` | Switch directly to a named profile |

Model profiles are defined in `~/.config/k13d/config.yaml`:

```yaml
models:
  - name: gpt-4
    provider: openai
    model: gpt-4
  - name: local-llama
    provider: ollama
    model: llama3.2
    endpoint: http://localhost:11434

active_model: gpt-4
```

---

## Streaming Responses

Real-time response streaming for immediate feedback.

| Feature | Description |
|---------|-------------|
| **SSE Streaming** | Server-Sent Events for real-time updates |
| **Live Cursor** | Shows AI is still thinking |
| **Tool Progress** | Shows which tool is executing |
| **Partial Results** | See response as it's generated |

---

## LLM Providers

### Supported Providers

![LLM Settings](../images/webui-settings-llm.png)

| Provider | Tool Calling | Best For |
|----------|:------------:|----------|
| **OpenAI** | ✅ | Production, best tool support |
| **Anthropic** | ✅ | Claude models |
| **Google Gemini** | ✅ | Fast responses |
| **Azure OpenAI** | ✅ | Enterprise |
| **Ollama** | ✅ | Local, air-gapped |
| **Embedded SLLM** | ⚠️ | Zero-dependency |

### Configuration

```yaml
llm:
  provider: openai
  model: gpt-4
  api_key: ${OPENAI_API_KEY}
```

---

## MCP Integration

Extend AI capabilities with Model Context Protocol servers.

![MCP Settings](../images/webui-settings-mcp.png)

### MCP Client Mode (Default)

k13d connects to external MCP servers:

```yaml
mcp:
  servers:
    - name: sequential-thinking
      command: npx
      args: ["-y", "@modelcontextprotocol/server-sequential-thinking"]
    - name: github
      command: npx
      args: ["-y", "@modelcontextprotocol/server-github"]
```

### MCP Server Mode

Run k13d as MCP server for Claude Desktop:

```bash
k13d --mcp
```

---

## Example Conversations

### Troubleshooting

```
User: Why is my api-server deployment not working?

AI: I'll investigate the api-server deployment.

[Tool: kubectl get deployment api-server -o yaml]
[Tool: kubectl get pods -l app=api-server]
[Tool: kubectl get events --field-selector involvedObject.name=api-server]

The api-server deployment has 0/3 ready replicas because:

1. **Image Pull Error**: The image "api-server:v2.0" doesn't exist
2. All pods are in "ImagePullBackOff" status

To fix this, update to an existing image:
```

### Operations

```
User: Scale nginx to 5 replicas

AI: I'll scale the nginx deployment to 5 replicas.

[Approval Required: kubectl scale deployment nginx --replicas=5]

User: [Approves]

AI: Successfully scaled nginx to 5 replicas.
[Tool: kubectl get deployment nginx]

NAME    READY   UP-TO-DATE   AVAILABLE
nginx   5/5     5            5
```

### Explanation

```
User: Explain this HPA configuration

AI: This HorizontalPodAutoscaler is configured to:

**Target:** frontend deployment
**Scaling Behavior:**
- Min replicas: 2
- Max replicas: 10
- Scale up when: CPU > 50%
- Stabilization window: 300 seconds

This means Kubernetes will:
1. Keep at least 2 pods running
2. Add pods when CPU exceeds 50%
3. Wait 5 minutes before scaling down
4. Never exceed 10 pods
```

---

## Best Practices

### Be Specific

```
# Good
"Scale nginx deployment in production namespace to 5 replicas"

# Less good
"Scale nginx"
```

### Provide Context

```
# Good
"Check why pod nginx-abc123 in default namespace is failing"

# Less good
"Check the pod"
```

### Use for Investigation First

```
# Good workflow
1. "Show me pods with high memory usage"
2. "Describe the api-server pod"
3. "Now scale it to 3 replicas"

# Less good
1. "Scale everything to 10 replicas"
```
