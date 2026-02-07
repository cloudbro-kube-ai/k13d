# k13d Configuration Guide

This guide provides examples for configuring **k13d**, including LLM providers and the Multi-Context Protocol (MCP).

## LLM Configuration
Stored in `~/.config/k13d/config.yaml`.

### OpenAI
```yaml
llm:
  provider: openai
  model: gpt-4o
  endpoint: https://api.openai.com/v1
  api_key: sk-...
```

### Ollama (Local)
```yaml
llm:
  provider: ollama
  model: llama3.1
  endpoint: http://localhost:11434/v1
```

---

## MCP Configuration

k13d is an **MCP Client** that spawns external MCP servers as child processes and communicates via JSON-RPC 2.0 over stdio. MCP configuration is stored in `~/.config/k13d/config.yaml` alongside LLM settings. This allows the AI Assistant to use external tools.

### Example MCP Configuration in `config.yaml`
```yaml
# In ~/.config/k13d/config.yaml
mcp:
  servers:
    - name: kubernetes
      enabled: true
      command: npx
      args: ["-y", "@modelcontextprotocol/server-kubernetes"]
      description: "Kubernetes management tools"

    - name: google-search
      enabled: true
      command: npx
      args: ["-y", "@modelcontextprotocol/server-google-search"]
      description: "Google search integration"
      env:
        GOOGLE_API_KEY: "your-google-api-key"
        GOOGLE_SEARCH_ENGINE_ID: "your-cse-id"

    - name: filesystem
      enabled: false
      command: npx
      args: ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/dir"]
      description: "Filesystem access tools"
```

**Note**: k13d spawns these MCP servers as child processes and communicates via JSON-RPC 2.0 over stdio. See [MCP Guide](./MCP_GUIDE.md) for more details.

## Assistant Navigation
- **Switch to Assistant**: Press `TAB` or `Right Arrow`.
- **Focus Chat History**: Press `Up Arrow` while in the text input field.
- **Scroll Chat**: Use `Up`/`Down` or `PgUp`/`PgDn` once the chat history is focused.
- **Return to Typing**: Press `ESC` or `Enter` from the chat history.
