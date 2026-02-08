# Tool Calling

k13d uses AI tool calling to execute kubectl commands, bash scripts, and MCP tools based on natural language requests.

## Overview

Tool calling allows the AI to:

1. **Understand Intent**: Parse natural language requests
2. **Select Tools**: Choose appropriate tools for the task
3. **Execute Commands**: Run kubectl, bash, or MCP commands
4. **Report Results**: Explain outcomes to the user

```
User: "Scale nginx to 5 replicas"
         │
         ▼
┌─────────────────────────────────────────┐
│ AI Agent                                 │
│                                         │
│ 1. Parse intent: scale deployment       │
│ 2. Select tool: kubectl                 │
│ 3. Build command: kubectl scale         │
│    deployment nginx --replicas=5        │
│ 4. Request approval                     │
│ 5. Execute command                      │
│ 6. Report result                        │
└─────────────────────────────────────────┘
         │
         ▼
Result: "Scaled deployment nginx to 5 replicas"
```

## Available Tools

### kubectl

Execute any kubectl command:

```json
{
  "name": "kubectl",
  "description": "Execute kubectl commands",
  "parameters": {
    "command": "get pods -n default"
  }
}
```

Examples:
- `kubectl get pods`
- `kubectl describe deployment nginx`
- `kubectl logs pod-name`
- `kubectl apply -f manifest.yaml`

### bash

Execute shell commands:

```json
{
  "name": "bash",
  "description": "Execute shell commands",
  "parameters": {
    "command": "date"
  }
}
```

Examples:
- `date`
- `grep pattern file`
- `jq '.items[]' data.json`

### MCP Tools

Dynamic tools from MCP servers:

```json
{
  "name": "sequential_thinking",
  "description": "Step-by-step reasoning",
  "parameters": {
    "thought": "Analyzing the issue...",
    "thoughtNumber": 1,
    "totalThoughts": 3
  }
}
```

## Tool Execution Flow

### 1. User Request

```
User: "Check why the api pod is crashing"
```

### 2. AI Planning

AI decides which tools to use:

```
Thought: I need to check pod status and logs
Tools: kubectl get pods, kubectl describe pod, kubectl logs
```

### 3. Tool Selection

AI generates tool calls:

```json
{
  "tool_calls": [
    {
      "id": "call_1",
      "function": {
        "name": "kubectl",
        "arguments": "{\"command\": \"get pods | grep api\"}"
      }
    }
  ]
}
```

### 4. Safety Check

Command is analyzed for safety:

```
Command: kubectl get pods | grep api
Category: READ_ONLY
Auto-approve: Yes
```

### 5. Approval (if needed)

For write/dangerous commands:

```
┌─────────────────────────────────────────┐
│ Approval Required                        │
│                                         │
│ kubectl delete pod api-xyz              │
│                                         │
│ [Approve] [Reject]                      │
└─────────────────────────────────────────┘
```

### 6. Execution

Command is executed:

```
$ kubectl get pods | grep api
api-abc123   1/1     Running   5 (10m ago)   2d
api-def456   0/1     CrashLoopBackOff   12   1h
```

### 7. Result Processing

AI interprets results and may call more tools:

```
I see api-def456 is in CrashLoopBackOff. Let me check the logs...
[Tool: kubectl logs api-def456 --tail=50]
```

## Safety System

### Command Classification

| Type | Examples | Approval |
|------|----------|----------|
| **Read** | get, describe, logs | Auto-approve |
| **Write** | apply, create, patch | Requires approval |
| **Dangerous** | delete, drain | Warning + approval |
| **Interactive** | exec, attach | Requires approval |

### AST Parsing

k13d parses commands as an Abstract Syntax Tree:

```
"kubectl get pods | xargs rm -rf /"
         │
         ▼
┌─────────────────────┐
│ AST Parser          │
├─────────────────────┤
│ - Pipeline detected │
│ - rm -rf found      │
│ - DANGEROUS!        │
└─────────────────────┘
```

### Blocked Patterns

Configure blocked command patterns:

```yaml
authorization:
  tool_approval:
    blocked_patterns:
      - "rm -rf /"
      - "kubectl delete ns kube-system"
      - ":(){ :|:& };:"  # Fork bomb
```

## Configuration

### Tool Approval Settings

```yaml
authorization:
  tool_approval:
    # Auto-approve read-only commands
    auto_approve_read_only: true

    # Require approval for write commands
    require_approval_for_write: true

    # Require approval for unknown commands
    require_approval_for_unknown: true

    # Block dangerous commands entirely
    block_dangerous: false

    # Approval timeout in seconds
    approval_timeout_seconds: 60
```

### Custom Tool Definitions

Add custom tools in config:

```yaml
tools:
  custom:
    - name: my_tool
      description: "My custom tool"
      command: "/path/to/script.sh"
      require_approval: true
```

## Examples

### Simple Query

```
User: "How many pods are running?"

AI: [Tool: kubectl get pods --no-headers | wc -l]
    You have 15 pods running in the default namespace.
```

### Multi-Tool Query

```
User: "Why is my deployment failing?"

AI: [Tool: kubectl get deployment nginx -o yaml]
    [Tool: kubectl get events --field-selector involvedObject.name=nginx]
    [Tool: kubectl get pods -l app=nginx]

    Your nginx deployment is failing because:
    1. Image 'nginx:invalid' cannot be pulled
    2. All 3 pods are in ImagePullBackOff state

    To fix this, update the image tag:
    kubectl set image deployment/nginx nginx=nginx:latest
```

### Action Request

```
User: "Scale the api deployment to 10 replicas"

AI: [Tool: kubectl scale deployment api --replicas=10]
    [Approval Required]

User: [Approves]

AI: Successfully scaled deployment 'api' to 10 replicas.
```

## Tool Response Format

### Success Response

```json
{
  "success": true,
  "output": "deployment.apps/nginx scaled",
  "exit_code": 0
}
```

### Error Response

```json
{
  "success": false,
  "error": "Error from server (NotFound): deployments.apps \"nginx\" not found",
  "exit_code": 1
}
```

## Debugging

### Enable Debug Logging

```yaml
debug: true
log_level: debug
```

### View Tool Calls

In TUI:
```
:debug tools
```

In Web: Settings → Debug → Tool Calls

### Audit Log

All tool calls are logged:

```sql
SELECT * FROM audit_logs
WHERE action = 'execute'
ORDER BY timestamp DESC;
```

## Best Practices

### 1. Be Specific

```
# Good
"Scale nginx deployment in production namespace to 5 replicas"

# Less good
"Scale nginx"
```

### 2. Review Approvals

Always review commands before approving.

### 3. Use Read-Only for Exploration

```
"Show me all pods with high memory usage"
# Uses only read commands
```

### 4. Incremental Actions

```
"First show me the current state, then let's decide on changes"
```

## Limitations

### 1. No Interactive Commands

Tools cannot handle interactive commands like `vim` or `less`.

### 2. Command Timeout

Commands timeout after 60 seconds by default.

### 3. Output Size

Large outputs are truncated.

### 4. No File Editing

Cannot directly edit files in containers.

## Next Steps

- [LLM Providers](providers.md) - Provider configuration
- [MCP Integration](../concepts/mcp-integration.md) - External tools
- [Security](../concepts/security.md) - Safety features
