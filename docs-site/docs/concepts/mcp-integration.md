# MCP Integration

This guide explains how MCP works in k13d and how to extend AI capabilities with custom tools.

## Table of Contents

- [What is MCP?](#what-is-mcp)
- [k13d MCP Modes](#k13d-mcp-modes)
- [MCP Server Mode](#mcp-server-mode)
- [MCP Client Mode](#mcp-client-mode)
- [Architecture Overview](#architecture-overview)
- [How It Works](#how-it-works)
- [Agentic Loop Integration](#agentic-loop-integration)
- [Configuration](#configuration)
- [Built-in Tools](#built-in-tools)
- [Adding MCP Servers](#adding-mcp-servers)
- [Creating Custom MCP Servers](#creating-custom-mcp-servers)
- [Troubleshooting](#troubleshooting)

---

## What is MCP?

**Model Context Protocol (MCP)** is an open protocol developed by Anthropic that standardizes how AI models interact with external tools and data sources. k13d implements MCP to extend its AI capabilities beyond the built-in kubectl and bash tools.

### Key Benefits

- **Extensibility**: Add new tools without modifying k13d core
- **Standardization**: Use any MCP-compatible server
- **Isolation**: Tools run in separate processes
- **Security**: Fine-grained control over tool capabilities

---

## k13d MCP Modes

k13d supports **both** MCP Server and MCP Client modes:

| Mode | Command | Description |
|------|---------|-------------|
| **MCP Server** | `k13d --mcp` | Exposes k13d tools to external MCP clients (Claude Desktop, Cursor, VS Code) |
| **MCP Client** | (default) | Connects to external MCP servers for additional tools |

```
┌─────────────────────────────────────────────────────────────────────┐
│                           k13d                                       │
│                                                                      │
│  ┌─────────────────────┐          ┌─────────────────────┐           │
│  │   MCP Server Mode   │          │   MCP Client Mode   │           │
│  │   (k13d --mcp)      │          │   (default)         │           │
│  │                     │          │                     │           │
│  │ Exposes tools:      │          │ Connects to:        │           │
│  │ - kubectl           │          │ - thinking server   │           │
│  │ - kubectl_get       │          │ - kubernetes server │           │
│  │ - kubectl_describe  │          │ - custom servers    │           │
│  │ - kubectl_logs      │          │                     │           │
│  │ - kubectl_apply     │          │                     │           │
│  │ - bash              │          │                     │           │
│  └──────────┬──────────┘          └──────────┬──────────┘           │
│             │                                │                       │
└─────────────┼────────────────────────────────┼───────────────────────┘
              │ stdio                          │ stdio
              ▼                                ▼
      ┌─────────────┐                  ┌─────────────┐
      │ Claude      │                  │ External    │
      │ Desktop,    │                  │ MCP         │
      │ Cursor,     │                  │ Servers     │
      │ VS Code     │                  │             │
      └─────────────┘                  └─────────────┘
```

---

## MCP Server Mode

Run k13d as an MCP server to expose Kubernetes management tools to external AI clients.

### Quick Start

```bash
# Start k13d as MCP server
k13d --mcp
```

### Integration with Claude Desktop

Add to your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "k13d": {
      "command": "k13d",
      "args": ["--mcp"]
    }
  }
}
```

### Integration with Cursor

Add to Cursor MCP settings:

```json
{
  "mcp": {
    "servers": {
      "k13d": {
        "command": "k13d",
        "args": ["--mcp"]
      }
    }
  }
}
```

### Available Tools (Server Mode)

| Tool | Description |
|------|-------------|
| `kubectl` | Execute any kubectl command |
| `kubectl_get` | Get Kubernetes resources with filtering |
| `kubectl_describe` | Describe a resource in detail |
| `kubectl_logs` | Get pod logs |
| `kubectl_apply` | Apply YAML manifests |
| `bash` | Execute shell commands |

### Example: Using k13d from Claude Desktop

Once configured, you can ask Claude:

> "Get all pods in the kube-system namespace"

Claude will use k13d's `kubectl_get` tool:

```
Using tool: kubectl_get
Arguments: {"resource": "pods", "namespace": "kube-system"}

Result:
NAME                                     READY   STATUS    RESTARTS   AGE
coredns-5d78c9869d-xxxxx                1/1     Running   0          10d
etcd-master                             1/1     Running   0          10d
kube-apiserver-master                   1/1     Running   0          10d
...
```

---

## MCP Client Mode

k13d can also act as an **MCP Client** to connect to external MCP servers for additional capabilities.

**Important**: k13d is an **MCP Client**, not an MCP Server. This distinction is crucial:

| Role | Description | k13d |
|------|-------------|------|
| **MCP Client** | Connects to MCP servers, discovers tools, invokes them | **This is k13d** |
| **MCP Server** | Provides tools, executes them when called | External processes |

### How k13d Uses MCP

1. **Spawns MCP Servers**: k13d starts external MCP server processes as child processes
2. **Communicates via stdio**: Uses JSON-RPC 2.0 over stdin/stdout pipes
3. **Discovers Tools**: Calls `tools/list` to learn what tools each server provides
4. **Invokes Tools**: When AI decides to use a tool, k13d calls `tools/call` on the appropriate server
5. **Returns Results**: Tool results are passed back to the AI for reasoning

```
┌───────────────────────────────────────────────────────────────┐
│                    k13d (MCP Client)                           │
│                                                                │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐       │
│   │ AI Agent    │───▶│ MCP Manager │───▶│ Tool Router │       │
│   └─────────────┘    └─────────────┘    └─────────────┘       │
│                             │                                  │
│                             │ spawn + JSON-RPC 2.0 stdio       │
└─────────────────────────────┼──────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
  │ MCP Server  │     │ MCP Server  │     │ MCP Server  │
  │ (thinking)  │     │ (kubernetes)│     │ (custom)    │
  └─────────────┘     └─────────────┘     └─────────────┘
     External            External            External
     Process             Process             Process
```

---

## Architecture Overview

### High-Level System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                               k13d Binary                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌────────────┐    ┌────────────┐    ┌────────────┐                         │
│  │  TUI Mode  │    │  Web Mode  │    │  CLI Mode  │                         │
│  │  (tview)   │    │  (HTTP)    │    │  (direct)  │                         │
│  └─────┬──────┘    └─────┬──────┘    └─────┬──────┘                         │
│        │                 │                 │                                 │
│        └─────────────────┼─────────────────┘                                 │
│                          │                                                   │
│                 ┌────────▼────────┐                                         │
│                 │   AI Agent      │  ← State Machine                        │
│                 │   (agentic      │    (Idle → Running → ToolAnalysis       │
│                 │    loop)        │     → WaitingForApproval → Done)        │
│                 └────────┬────────┘                                         │
│                          │                                                   │
│         ┌────────────────┼────────────────┐                                 │
│         │                │                │                                 │
│  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐                         │
│  │ LLM Provider│  │Tool Registry│  │   Safety    │                         │
│  │  (OpenAI,   │  │ (kubectl,   │  │  Analyzer   │                         │
│  │   Gemini,   │  │  bash, MCP) │  │             │                         │
│  │   Ollama)   │  └──────┬──────┘  └─────────────┘                         │
│  └─────────────┘         │                                                   │
│                          │                                                   │
│         ┌────────────────┴────────────────┐                                 │
│         │         Tool Router             │                                 │
│         │   (routes by tool.Type)         │                                 │
│         └────────────────┬────────────────┘                                 │
│                          │                                                   │
│         ┌────────────────┼────────────────┐                                 │
│         │                │                │                                 │
│  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐                         │
│  │  kubectl    │  │    bash     │  │ MCP Client  │                         │
│  │  Executor   │  │  Executor   │  │  (stdio)    │                         │
│  └─────────────┘  └─────────────┘  └──────┬──────┘                         │
│                                           │ JSON-RPC 2.0                    │
└───────────────────────────────────────────┼─────────────────────────────────┘
                                            │
                    ┌───────────────────────┼───────────────────────┐
                    ▼                       ▼                       ▼
            ┌─────────────┐         ┌─────────────┐         ┌─────────────┐
            │ MCP Server  │         │ MCP Server  │         │ MCP Server  │
            │ (sequential │         │ (database)  │         │ (custom)    │
            │  thinking)  │         │             │         │             │
            └─────────────┘         └─────────────┘         └─────────────┘
```

### Key Components

| Component | Location | Purpose |
|-----------|----------|---------|
| **MCP Client** | `pkg/mcp/client.go` | Manages server processes, JSON-RPC communication, tool discovery |
| **Tool Registry** | `pkg/ai/tools/tools.go` | Central registry for all tools (kubectl, bash, MCP) |
| **MCP Executor Adapter** | `pkg/mcp/client.go` | Bridges MCP Client to Tool Registry interface |
| **AI Agent** | `pkg/ai/agent/agent.go` | State machine managing conversation flow |
| **Conversation Loop** | `pkg/ai/agent/conversation.go` | Main agentic loop with tool handling, approval, safety |
| **LLM Provider** | `pkg/ai/providers/` | Handles tool calling protocol with LLM |
| **Safety Analyzer** | `pkg/ai/safety/analyzer.go` | Analyzes command safety (read-only, dangerous, etc.) |

---

## How It Works

### 1. MCP Server Connection Flow

When k13d starts or an MCP server is enabled:

```
┌─────────────────────────────────────────────────────────────────┐
│  1. k13d spawns MCP server process                              │
│     exec.Command("npx", "-y", "@modelcontextprotocol/server-*") │
├─────────────────────────────────────────────────────────────────┤
│  2. Establishes stdio communication (stdin/stdout pipes)        │
├─────────────────────────────────────────────────────────────────┤
│  3. Sends "initialize" request (JSON-RPC 2.0)                   │
│     → Protocol version: 2024-11-05                              │
│     → Client info: {name: "k13d", version: "1.0.0"}            │
├─────────────────────────────────────────────────────────────────┤
│  4. Receives server capabilities and info                       │
├─────────────────────────────────────────────────────────────────┤
│  5. Sends "notifications/initialized" notification              │
├─────────────────────────────────────────────────────────────────┤
│  6. Calls "tools/list" to discover available tools              │
├─────────────────────────────────────────────────────────────────┤
│  7. Registers tools in the Tool Registry                        │
│     → Tagged with server name for routing                       │
└─────────────────────────────────────────────────────────────────┘
```

### 2. JSON-RPC Message Format

**Initialize Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "k13d",
      "version": "1.0.0"
    }
  }
}
```

**Tool Call Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "sequential_thinking",
    "arguments": {
      "thought": "Analyzing pod status...",
      "thoughtNumber": 1,
      "totalThoughts": 3,
      "nextThoughtNeeded": true
    }
  }
}
```

**Tool Call Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Step 1 complete. Ready for next analysis..."
      }
    ],
    "isError": false
  }
}
```

---

## Agentic Loop Integration

### Agent State Machine

The AI Agent operates as a state machine that manages the conversation flow:

```
    ┌─────────┐
    │  Idle   │◄────────────────────────┐
    └────┬────┘                         │
         │ User Message                 │
         ▼                              │
    ┌─────────┐                         │
    │ Running │◄─────────────────┐      │
    └────┬────┘                  │      │
         │ LLM Response          │      │
         ▼                       │      │
    ┌──────────────┐             │      │
    │ToolAnalysis  │             │      │
    └────┬─────────┘             │      │
         │                       │      │
         ├─ Auto-approve ────────┘      │
         │  (read-only)                 │
         ▼                              │
    ┌──────────────────┐                │
    │WaitingForApproval│                │
    └────┬─────────────┘                │
         │                              │
         ├─ Approved ──► Execute ───────┤
         │                              │
         ├─ Rejected ───────────────────┤
         │                              │
         └─ Timeout ────────────────────┤
                                        │
    ┌─────────┐                         │
    │  Done   │─────────────────────────┤
    └─────────┘                         │
                                        │
    ┌─────────┐                         │
    │  Error  │─────────────────────────┘
    └─────────┘
```

### Complete Tool Execution Flow

Here's the detailed flow from user question to MCP tool execution:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 1: User Input                                                          │
│                                                                             │
│   User: "Why is my nginx pod failing?"                                     │
│   → Agent.handleUserMessage()                                               │
│   → Add to session history                                                  │
│   → State: Idle → Running                                                   │
└────────────────────────────────────────────────┬────────────────────────────┘
                                                 │
                                                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 2: LLM Call with Tools                                                 │
│                                                                             │
│   Agent.callLLM()                                                           │
│   ├─ Build prompt with conversation history                                │
│   ├─ Convert Tool Registry → ToolDefinitions[]                             │
│   │   (kubectl, bash, MCP tools all included)                              │
│   └─ toolProvider.AskWithTools(prompt, tools, streamCb, toolCb)            │
└────────────────────────────────────────────────┬────────────────────────────┘
                                                 │
                                                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 3: LLM Provider Processing                                             │
│                                                                             │
│   POST /chat/completions (with tools array)                                 │
│   → LLM decides to use tools:                                               │
│     {                                                                       │
│       "tool_calls": [{                                                      │
│         "id": "call_abc123",                                                │
│         "function": {                                                       │
│           "name": "kubectl",                                                │
│           "arguments": "{\"command\": \"get pods -n default\"}"            │
│         }                                                                   │
│       }]                                                                    │
│     }                                                                       │
└────────────────────────────────────────────────┬────────────────────────────┘
                                                 │
                                                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 4: Tool Call Handling                                                  │
│                                                                             │
│   Agent.handleToolCall(call)                                                │
│   ├─ Extract command from arguments                                        │
│   ├─ Safety Analysis (safetyAnalyzer.Analyze)                              │
│   │   → IsReadOnly: true/false                                             │
│   │   → IsDangerous: true/false                                            │
│   │   → Warnings: ["This will delete..."]                                  │
│   ├─ Emit tool call request to UI                                          │
│   └─ Check if approval needed:                                             │
│       if (needsApproval) {                                                 │
│         requestAndWaitForApproval()  // Shows UI, waits 30s               │
│       }                                                                     │
└────────────────────────────────────────────────┬────────────────────────────┘
                                                 │
                                                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 5: Tool Registry Execution (Routing)                                   │
│                                                                             │
│   toolRegistry.Execute(ctx, toolCall)                                       │
│   ├─ Look up tool by name                                                  │
│   ├─ Check tool.Type:                                                       │
│   │   ├─ ToolTypeKubectl → executor.Execute(kubectl, args)                │
│   │   ├─ ToolTypeBash    → executor.Execute(bash, args)                   │
│   │   └─ ToolTypeMCP     → mcpExecutor.CallTool(name, args) ◄──────────── │
│   └─ Return ToolResult                                                      │
└────────────────────────────────────────────────┬────────────────────────────┘
                                                 │ (MCP path)
                                                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 6: MCP Client Execution                                                │
│                                                                             │
│   MCPToolExecutorAdapter.CallTool(toolName, args)                           │
│   ├─ client.CallTool(ctx, toolName, args)                                  │
│   │   ├─ Find ServerConnection with this tool                              │
│   │   └─ conn.callTool(ctx, name, args)                                    │
│   │       └─ sendRequest(ctx, "tools/call", params)                        │
│   │           ├─ Write JSON-RPC request to stdin                           │
│   │           └─ Read JSON-RPC response from stdout                        │
│   └─ Extract text content from result.Content[]                            │
└────────────────────────────────────────────────┬────────────────────────────┘
                                                 │
                                                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 7: Result Back to LLM                                                  │
│                                                                             │
│   Tool result returned to LLM provider                                      │
│   → LLM may call more tools or generate final response                     │
│   → Loop continues until no more tool calls                                 │
│   → Final response streamed to UI                                          │
│   → State: Running → Done → Idle                                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Code References

**Agent Main Loop** (`pkg/ai/agent/conversation.go:28-105`):
```go
func (a *Agent) Run(ctx context.Context) error {
    for {
        switch a.State() {
        case StateIdle:
            // Wait for user input
        case StateRunning:
            // Call LLM and process response
            a.callLLM()
        case StateToolAnalysis:
            // Analyze pending tool calls
        case StateWaitingForApproval:
            // Wait for user decision
        case StateDone:
            // Conversation turn complete
        }
    }
}
```

**Tool Call Handler** (`pkg/ai/agent/conversation.go:256-335`):
```go
func (a *Agent) handleToolCall(call providers.ToolCall) providers.ToolResult {
    // 1. Create internal tool call info
    toolCall := &ToolCallInfo{...}

    // 2. Analyze safety
    safetyReport := a.safetyAnalyzer.Analyze(toolCall.Command)

    // 3. Request approval if needed
    if needsApproval {
        approved := a.requestAndWaitForApproval(toolCall)
        if !approved { return cancelled }
    }

    // 4. Execute via registry (routes to MCP if needed)
    result := a.toolRegistry.Execute(a.ctx, registryCall)

    // 5. Return result to LLM
    return providers.ToolResult{...}
}
```

**MCP Tool Execution** (`pkg/mcp/client.go:229-244`):
```go
func (c *Client) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*CallToolResult, error) {
    // Find which server has this tool
    for _, conn := range c.servers {
        for _, tool := range conn.tools {
            if tool.Name == toolName {
                return conn.callTool(ctx, toolName, args)
            }
        }
    }
    return nil, fmt.Errorf("tool not found: %s", toolName)
}
```

---

## Configuration

### Config File

MCP servers are configured in `~/.config/k13d/config.yaml`:

```yaml
mcp:
  servers:
    - name: sequential-thinking
      enabled: true
      command: npx
      args:
        - "-y"
        - "@modelcontextprotocol/server-sequential-thinking"
      description: "Step-by-step reasoning tool"

    - name: kubernetes
      enabled: true
      command: npx
      args:
        - "-y"
        - "@anthropic/mcp-server-kubernetes"
      description: "Kubernetes management tools"
      env:
        KUBECONFIG: "/home/user/.kube/config"

    - name: github
      enabled: true
      command: npx
      args:
        - "-y"
        - "@modelcontextprotocol/server-github"
      description: "GitHub integration"
      env:
        GITHUB_TOKEN: "${GITHUB_TOKEN}"

    - name: postgres
      enabled: false
      command: npx
      args:
        - "-y"
        - "@modelcontextprotocol/server-postgres"
        - "postgresql://user:pass@localhost/db"
      description: "PostgreSQL database tools"
```

### Web UI Configuration

1. Go to **Settings** → **MCP** tab
2. Click **Add MCP Server**
3. Fill in:
   - **Name**: Unique identifier (e.g., "kubernetes")
   - **Command**: Executable (e.g., "npx", "docker")
   - **Arguments**: Command arguments (comma-separated)
   - **Description**: What the server does
4. Toggle **Enabled** to connect

---

## Built-in Tools

k13d includes these built-in tools (not MCP):

| Tool | Type | Description |
|------|------|-------------|
| `kubectl` | ToolTypeKubectl | Execute kubectl commands |
| `bash` | ToolTypeBash | Execute shell commands (with safety checks) |

These are always available and don't require MCP configuration.

### Tool Type Routing

```go
const (
    ToolTypeKubectl ToolType = "kubectl"
    ToolTypeBash    ToolType = "bash"
    ToolTypeRead    ToolType = "read_file"
    ToolTypeWrite   ToolType = "write_file"
    ToolTypeMCP     ToolType = "mcp"  // MCP server provided tools
)
```

When `toolRegistry.Execute()` is called:
- `kubectl` and `bash` → Local executor
- `mcp` → MCP Client → External server process

---

## Adding MCP Servers

### Sequential Thinking (Default)

```yaml
- name: sequential-thinking
  command: npx
  args: ["-y", "@modelcontextprotocol/server-sequential-thinking"]
```

Provides step-by-step reasoning capabilities for complex problem solving.

### Kubernetes MCP Server

```yaml
- name: kubernetes
  command: npx
  args: ["-y", "@anthropic/mcp-server-kubernetes"]
```

Provides:
- `get_pods` - List pods in namespace
- `get_deployments` - List deployments
- `describe_pod` - Get pod details
- `get_logs` - Get pod logs
- `apply_manifest` - Apply YAML manifest

### GitHub MCP Server

```yaml
- name: github
  command: npx
  args: ["-y", "@modelcontextprotocol/server-github"]
  env:
    GITHUB_TOKEN: "${GITHUB_TOKEN}"
```

Provides:
- `search_repositories` - Search GitHub repos
- `get_file_contents` - Read file from repo
- `create_issue` - Create GitHub issue
- `create_pull_request` - Create PR

### Filesystem MCP Server

```yaml
- name: filesystem
  command: npx
  args: ["-y", "@modelcontextprotocol/server-filesystem", "/allowed/path"]
```

Provides:
- `read_file` - Read file contents
- `write_file` - Write to file
- `list_directory` - List directory contents

### Docker MCP Server

```yaml
- name: docker-mcp
  command: docker
  args: ["run", "-i", "--rm", "mcp/docker-server"]
```

---

## Creating Custom MCP Servers

### Python Example

```python
#!/usr/bin/env python3
"""Custom MCP server for k13d."""

import json
import sys

def send_response(id, result):
    response = {"jsonrpc": "2.0", "id": id, "result": result}
    print(json.dumps(response), flush=True)

def handle_request(request):
    method = request.get("method")
    id = request.get("id")
    params = request.get("params", {})

    if method == "initialize":
        send_response(id, {
            "protocolVersion": "2024-11-05",
            "capabilities": {"tools": {}},
            "serverInfo": {"name": "custom-server", "version": "1.0.0"}
        })

    elif method == "tools/list":
        send_response(id, {
            "tools": [
                {
                    "name": "hello_world",
                    "description": "Says hello",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "name": {"type": "string", "description": "Name to greet"}
                        }
                    }
                }
            ]
        })

    elif method == "tools/call":
        name = params.get("name")
        args = params.get("arguments", {})

        if name == "hello_world":
            result = f"Hello, {args.get('name', 'World')}!"
            send_response(id, {
                "content": [{"type": "text", "text": result}]
            })

def main():
    for line in sys.stdin:
        try:
            request = json.loads(line.strip())
            handle_request(request)
        except json.JSONDecodeError:
            pass

if __name__ == "__main__":
    main()
```

### Go Example

```go
package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
)

type Request struct {
    JSONRPC string                 `json:"jsonrpc"`
    ID      int64                  `json:"id"`
    Method  string                 `json:"method"`
    Params  map[string]interface{} `json:"params"`
}

type Response struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      int64       `json:"id"`
    Result  interface{} `json:"result,omitempty"`
    Error   interface{} `json:"error,omitempty"`
}

func main() {
    scanner := bufio.NewScanner(os.Stdin)

    for scanner.Scan() {
        var req Request
        if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
            continue
        }

        var result interface{}

        switch req.Method {
        case "initialize":
            result = map[string]interface{}{
                "protocolVersion": "2024-11-05",
                "capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
                "serverInfo":      map[string]interface{}{"name": "go-mcp", "version": "1.0.0"},
            }
        case "tools/list":
            result = map[string]interface{}{
                "tools": []map[string]interface{}{
                    {
                        "name":        "get_time",
                        "description": "Get current time",
                        "inputSchema": map[string]interface{}{"type": "object"},
                    },
                },
            }
        case "tools/call":
            result = map[string]interface{}{
                "content": []map[string]interface{}{
                    {"type": "text", "text": "Current time: ..."},
                },
            }
        }

        resp := Response{JSONRPC: "2.0", ID: req.ID, Result: result}
        data, _ := json.Marshal(resp)
        fmt.Println(string(data))
    }
}
```

### Register Custom Server

```yaml
mcp:
  servers:
    - name: my-custom-server
      command: python3
      args: ["/path/to/my_server.py"]
      description: "My custom MCP tools"
```

---

## Troubleshooting

### Server Not Connecting

```bash
# Test server manually
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | npx -y @modelcontextprotocol/server-sequential-thinking
```

### Tools Not Appearing

1. Check server is enabled in config
2. Verify command path is correct
3. Check k13d logs for connection errors
4. Ensure server responds to `tools/list`

### Tool Execution Fails

1. Check tool arguments match schema
2. Verify server has necessary permissions
3. Check environment variables are set
4. Look at server stderr for errors

### Debug Mode

Enable debug logging:

```yaml
debug: true
log_level: debug
```

Check logs at `~/.config/k13d/k13d.log`

---

## Security Considerations

1. **Trust**: Only add MCP servers from trusted sources
2. **Permissions**: MCP servers run with k13d's permissions
3. **Environment**: Be careful with environment variables containing secrets
4. **Network**: Some servers may make network requests
5. **Sandboxing**: Consider running servers in containers

### Safety Analyzer

All tool commands (including MCP) pass through the safety analyzer:

```go
type Report struct {
    Command          string
    Type             CommandType  // read, write, dangerous, interactive
    RequiresApproval bool
    IsDangerous      bool
    IsInteractive    bool
    IsReadOnly       bool
    Warnings         []string
}
```

- **Read-only** commands (get, describe, logs) can be auto-approved
- **Write** commands (apply, create, patch) require confirmation
- **Dangerous** commands (delete, drain) show extra warnings

---

## Next Steps

- [Configuration Guide](./CONFIGURATION_GUIDE.md)
- [User Guide](./USER_GUIDE.md)
- [Architecture Guide](./ARCHITECTURE.md)
- [MCP Specification](https://modelcontextprotocol.io/docs)
