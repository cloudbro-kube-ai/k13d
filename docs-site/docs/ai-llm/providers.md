# LLM Providers

k13d supports multiple LLM providers for AI-powered features.

Need the exact save/switch/storage behavior across Web UI and TUI? See [Model Settings & Storage](model-settings-storage.md).

## Supported Providers

| Provider | Models | Local | API Key |
|----------|--------|-------|---------|
| **OpenAI** | GPT-4o, o3-mini, GPT-4 | No | Required |
| **Anthropic** | Claude Opus 4, Sonnet 4, Haiku 4.5 | No | Required |
| **Google Gemini** | Gemini 2.5 Flash, 2.0 | No | Required |
| **Upstage Solar** | Solar Pro2, Solar Pro | No | Required |
| **Ollama** | Llama, Qwen, Mistral, etc. | Yes | Not needed |
| **Azure OpenAI** | GPT-4, GPT-3.5 | No | Required |
| **AWS Bedrock** | Claude, Llama, Titan | No | Required |

## Configuration

### OpenAI

```yaml
# ~/.config/k13d/config.yaml
llm:
  provider: openai
  model: gpt-4
  api_key: ${OPENAI_API_KEY}
```

Or via environment variable:

```bash
export OPENAI_API_KEY=sk-your-key-here
k13d --web
```

### Anthropic (Claude)

```yaml
llm:
  provider: anthropic
  model: claude-sonnet-4-20250514
  api_key: ${ANTHROPIC_API_KEY}
  # endpoint: https://api.anthropic.com  # default
```

Available models: `claude-opus-4-20250514`, `claude-sonnet-4-20250514`, `claude-haiku-4-5-20251001`, `claude-3-5-sonnet-20241022`

### Google Gemini

```yaml
llm:
  provider: gemini
  model: gemini-2.5-flash
  api_key: ${GOOGLE_API_KEY}
```

### Ollama (Local)

Start Ollama:

```bash
ollama serve
ollama pull gpt-oss:20b
```

Configure k13d:

```yaml
llm:
  provider: ollama
  model: gpt-oss:20b
  endpoint: http://localhost:11434
```

Important: k13d requires an Ollama model with **tools/function calling** support. Some Ollama models can connect and answer plain text prompts but still fail in k13d because the AI Assistant depends on tools. Use `gpt-oss:20b` or another Ollama model whose card explicitly lists tools support.

### Azure OpenAI

```yaml
llm:
  provider: azopenai
  model: gpt-4
  endpoint: https://your-resource.openai.azure.com/
  api_key: ${AZURE_OPENAI_API_KEY}
  api_version: "2024-02-15-preview"
```

### AWS Bedrock

```yaml
llm:
  provider: bedrock
  model: anthropic.claude-3-sonnet-20240229-v1:0
  # Uses AWS credentials from environment
```

Required environment variables:

```bash
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
export AWS_REGION=us-east-1
```

### Embedded LLM Removal

Embedded LLM support has been removed.

- Use **Ollama** for local/private inference
- Update old configs from `provider: embedded` to `provider: ollama`

## Multi-Model Configuration

Configure multiple models and switch between them:

```yaml
models:
  - name: gpt-4
    provider: openai
    model: gpt-4
    api_key: ${OPENAI_API_KEY}

  - name: local-ollama
    provider: ollama
    model: gpt-oss:20b
    endpoint: http://localhost:11434

  - name: claude
    provider: anthropic
    model: claude-sonnet-4-20250514
    api_key: ${ANTHROPIC_API_KEY}

# Default model
active_model: gpt-4
```

Switch models at runtime:

**TUI:**
```
:model local-llama
```

**Web:**
Settings → AI → Active Model

For what this changes in `llm`, `models[]`, and `active_model`, see [Model Settings & Storage](model-settings-storage.md).

## Provider Features

### Feature Comparison

| Feature | OpenAI | Anthropic | Gemini | Ollama | Solar |
|---------|--------|-----------|--------|--------|-------|
| Streaming | ✅ | ✅ | ✅ | ✅ | ✅ |
| Tool Calling | ✅ | ✅ | ✅ | Model-dependent | ✅ |
| Vision | ✅ | ✅ | ✅ | ⚠️ | ❌ |
| Context Length | 128K | 200K | 1M | Varies | 32K |

### Tool Calling Support

k13d's AI Assistant depends on tool calling for kubectl, bash, and MCP integration. Provider support is not enough by itself; the **selected model** must also support tools. This is especially important for **Ollama**, where support varies by model tag.

```
User: "Scale nginx to 5 replicas"
AI: [Calls kubectl scale deployment nginx --replicas=5]
```

## Performance Considerations

### Latency

| Provider | Typical Latency |
|----------|-----------------|
| OpenAI GPT-4 | 2-5 seconds |
| OpenAI GPT-3.5 | 0.5-2 seconds |
| Anthropic Claude | 2-4 seconds |
| Ollama (local) | 1-10 seconds* |
| Embedded | 2-20 seconds* |

*Depends on hardware

### Cost Comparison

| Provider | Cost (per 1M tokens) |
|----------|---------------------|
| GPT-4 | ~$30 |
| GPT-3.5 | ~$0.50 |
| Claude 3 Opus | ~$15 |
| Gemini Pro | ~$0.50 |
| Ollama | Free (local) |
| Embedded | Free (local) |

## Recommended Configurations

### For Best Quality

```yaml
llm:
  provider: openai
  model: gpt-4-turbo
```

### For Speed

```yaml
llm:
  provider: openai
  model: gpt-3.5-turbo
```

### For Privacy (Local)

```yaml
llm:
  provider: ollama
  model: gpt-oss:20b
  endpoint: http://localhost:11434
```

### For Air-Gapped

```yaml
llm:
  provider: ollama
  model: gpt-oss:20b
  endpoint: http://localhost:11434
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | OpenAI API key |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `GOOGLE_API_KEY` | Google AI API key |
| `AZURE_OPENAI_API_KEY` | Azure OpenAI key |
| `AWS_ACCESS_KEY_ID` | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key |
| `AWS_REGION` | AWS region |

## Troubleshooting

### API Key Not Found

```
Error: OpenAI API key not found
```

Solution:
```bash
export OPENAI_API_KEY=sk-your-key
# or add to config.yaml
```

### Rate Limit Exceeded

```
Error: Rate limit exceeded
```

Solutions:
1. Wait and retry
2. Upgrade API plan
3. Switch to different model

### Ollama Connection Failed

```
Error: Connection refused localhost:11434
```

Solution:
```bash
# Start Ollama
ollama serve

# Verify it's running
curl http://localhost:11434/api/tags
```

### Slow Responses

For faster responses:
1. Use GPT-3.5 instead of GPT-4
2. Use local Ollama with smaller model
3. Reduce context length

## Security Best Practices

### 1. Use Environment Variables

```bash
# Good
export OPENAI_API_KEY=sk-...
k13d --web

# Bad - key in config file
llm:
  api_key: sk-actual-key-here
```

### 2. Rotate Keys Regularly

### 3. Use Least Privilege

Create API keys with minimal permissions.

### 4. Monitor Usage

Track API usage to detect anomalies.

## Next Steps

- [Embedded LLM Removal](embedded.md) - Migration note
- [Tool Calling](tool-calling.md) - How AI executes commands
- [Benchmarks](benchmarks.md) - Model performance comparison
