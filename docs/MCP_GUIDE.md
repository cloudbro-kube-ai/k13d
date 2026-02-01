# MCP (Model Context Protocol) Guide

This guide explains how MCP works in k13d and how to extend AI capabilities with custom tools.

## Table of Contents

- [What is MCP?](#what-is-mcp)
- [Architecture](#architecture)
- [How It Works](#how-it-works)
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

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                           k13d                                   │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │  AI Agent    │───▶│  Tool Router │───▶│  MCP Client  │      │
│  └──────────────┘    └──────────────┘    └──────────────┘      │
│                              │                   │              │
│                              │                   │ JSON-RPC 2.0 │
│                              ▼                   ▼              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Tool Registry                         │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────────────────────┐  │   │
│  │  │ kubectl │  │  bash   │  │   MCP Tools (dynamic)   │  │   │
│  │  └─────────┘  └─────────┘  └─────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
            ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
            │ MCP Server  │ │ MCP Server  │ │ MCP Server  │
            │ (kubectl)   │ │ (database)  │ │ (custom)    │
            └─────────────┘ └─────────────┘ └─────────────┘
                    │               │               │
                    ▼               ▼               ▼
            ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
            │ Kubernetes  │ │  Database   │ │  External   │
            │   Cluster   │ │             │ │   Service   │
            └─────────────┘ └─────────────┘ └─────────────┘
```

---

## How It Works

### 1. Connection Flow

When k13d starts or an MCP server is enabled:

```
1. k13d spawns MCP server process (e.g., npx @anthropic/mcp-server-kubernetes)
2. Establishes stdio communication (JSON-RPC 2.0)
3. Sends "initialize" request with protocol version
4. Receives server capabilities and info
5. Sends "notifications/initialized" notification
6. Calls "tools/list" to discover available tools
7. Registers tools in the Tool Registry
```

### 2. Tool Execution Flow

When the AI decides to use an MCP tool:

```
1. AI generates tool call: { name: "mcp_kubernetes_get_pods", arguments: {...} }
2. Tool Router identifies it as an MCP tool (prefix "mcp_")
3. MCP Client finds the server that provides this tool
4. Sends "tools/call" JSON-RPC request to server
5. Server executes the tool and returns result
6. Result is passed back to AI for interpretation
```

### 3. JSON-RPC Messages

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
    "name": "get_pods",
    "arguments": {
      "namespace": "default"
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
        "text": "NAME                    READY   STATUS    RESTARTS   AGE\nnginx-7c5d8bc4c-xyz    1/1     Running   0          5m"
      }
    ]
  }
}
```

---

## Configuration

### Config File

MCP servers are configured in `~/.config/k13d/config.yaml`:

```yaml
mcp:
  servers:
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

| Tool | Description |
|------|-------------|
| `kubectl` | Execute kubectl commands |
| `bash` | Execute shell commands (with safety checks) |

These are always available and don't require MCP configuration.

---

## Adding MCP Servers

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
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | npx -y @anthropic/mcp-server-kubernetes
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

---

## Next Steps

- [Configuration Guide](./CONFIGURATION_GUIDE.md)
- [User Guide](./USER_GUIDE.md)
- [MCP Specification](https://modelcontextprotocol.io/docs)
