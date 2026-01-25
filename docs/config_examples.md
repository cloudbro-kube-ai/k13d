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
Stored in `~/.config/kubectl-ai/mcp.yaml`. This allows the AI Assistant to use external tools.

### Example `mcp.yaml`
```yaml
mcpServers:
  kubernetes:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-kubernetes"]
  
  google-search:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-google-search"]
    env:
      GOOGLE_API_KEY: "your-google-api-key"
      GOOGLE_SEARCH_ENGINE_ID: "your-cse-id"

  filesystem:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/dir"]
```

## Assistant Navigation
- **Switch to Assistant**: Press `TAB` or `Right Arrow`.
- **Focus Chat History**: Press `Up Arrow` while in the text input field.
- **Scroll Chat**: Use `Up`/`Down` or `PgUp`/`PgDn` once the chat history is focused.
- **Return to Typing**: Press `ESC` or `Enter` from the chat history.
