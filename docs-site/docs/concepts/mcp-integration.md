# MCP Integration

k13d supports the Model Context Protocol (MCP) for extending AI capabilities with external tools.

## What is MCP?

**Model Context Protocol (MCP)** is an open protocol developed by Anthropic that standardizes how AI models interact with external tools and data sources.

### Key Benefits

- **Extensibility**: Add new tools without modifying k13d core
- **Standardization**: Use any MCP-compatible server
- **Isolation**: Tools run in separate processes
- **Security**: Fine-grained control over tool capabilities

## k13d MCP Modes

k13d supports **both** MCP Server and MCP Client modes:

| Mode | Command | Description |
|------|---------|-------------|
| **MCP Server** | `k13d --mcp` | Exposes k13d tools to external clients |
| **MCP Client** | (default) | Connects to external MCP servers |

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
│  │ - bash              │          │ - custom servers    │           │
│  └──────────┬──────────┘          └──────────┬──────────┘           │
│             │ stdio                          │ stdio                 │
└─────────────┼────────────────────────────────┼───────────────────────┘
              ▼                                ▼
      ┌─────────────┐                  ┌─────────────┐
      │ Claude      │                  │ External    │
      │ Desktop,    │                  │ MCP         │
      │ Cursor      │                  │ Servers     │
      └─────────────┘                  └─────────────┘
```

## MCP Server Mode

Run k13d as an MCP server to expose Kubernetes tools to external AI clients.

### Quick Start

```bash
k13d --mcp
```

### Integration with Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

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

### Available Tools (Server Mode)

| Tool | Description |
|------|-------------|
| `kubectl` | Execute any kubectl command |
| `kubectl_get` | Get Kubernetes resources with filtering |
| `kubectl_describe` | Describe a resource in detail |
| `kubectl_logs` | Get pod logs |
| `kubectl_apply` | Apply YAML manifests |
| `bash` | Execute shell commands |

## MCP Client Mode

k13d can connect to external MCP servers for additional capabilities.

### Architecture

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
```

## Configuration

Configure MCP servers in `~/.config/k13d/config.yaml`:

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

    - name: github
      enabled: true
      command: npx
      args:
        - "-y"
        - "@modelcontextprotocol/server-github"
      env:
        GITHUB_TOKEN: "${GITHUB_TOKEN}"
```

## Popular MCP Servers

### Sequential Thinking

For step-by-step reasoning:

```yaml
- name: sequential-thinking
  command: npx
  args: ["-y", "@modelcontextprotocol/server-sequential-thinking"]
```

### GitHub

For GitHub integration:

```yaml
- name: github
  command: npx
  args: ["-y", "@modelcontextprotocol/server-github"]
  env:
    GITHUB_TOKEN: "${GITHUB_TOKEN}"
```

### Filesystem

For file operations:

```yaml
- name: filesystem
  command: npx
  args: ["-y", "@modelcontextprotocol/server-filesystem", "/allowed/path"]
```

## Creating Custom MCP Servers

### Python Example

```python
#!/usr/bin/env python3
import json
import sys

def send_response(id, result):
    response = {"jsonrpc": "2.0", "id": id, "result": result}
    print(json.dumps(response), flush=True)

def handle_request(request):
    method = request.get("method")
    id = request.get("id")

    if method == "initialize":
        send_response(id, {
            "protocolVersion": "2024-11-05",
            "capabilities": {"tools": {}},
            "serverInfo": {"name": "custom-server", "version": "1.0.0"}
        })

    elif method == "tools/list":
        send_response(id, {
            "tools": [{
                "name": "hello_world",
                "description": "Says hello",
                "inputSchema": {"type": "object", "properties": {
                    "name": {"type": "string"}
                }}
            }]
        })

    elif method == "tools/call":
        args = request.get("params", {}).get("arguments", {})
        result = f"Hello, {args.get('name', 'World')}!"
        send_response(id, {"content": [{"type": "text", "text": result}]})

for line in sys.stdin:
    try:
        handle_request(json.loads(line.strip()))
    except json.JSONDecodeError:
        pass
```

Register in config:

```yaml
mcp:
  servers:
    - name: my-server
      command: python3
      args: ["/path/to/my_server.py"]
```

## Security Considerations

1. **Trust**: Only add MCP servers from trusted sources
2. **Permissions**: MCP servers run with k13d's permissions
3. **Environment**: Be careful with environment variables
4. **Sandboxing**: Consider running servers in containers

## Troubleshooting

### Server Not Connecting

Test server manually:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{...}}' | npx -y @modelcontextprotocol/server-sequential-thinking
```

### Tools Not Appearing

1. Check server is enabled in config
2. Verify command path is correct
3. Check k13d logs for connection errors

### Debug Mode

Enable debug logging:

```yaml
debug: true
log_level: debug
```

## Next Steps

- [LLM Providers](../ai-llm/providers.md) - Configure AI providers
- [Tool Calling](../ai-llm/tool-calling.md) - How tools are executed
- [Configuration](../getting-started/configuration.md) - Full config options
