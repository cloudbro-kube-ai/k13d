# LLM Providers

k13d supports multiple LLM providers for AI-powered features.

## Supported Providers

| Provider | Models | Local | API Key |
|----------|--------|-------|---------|
| **OpenAI** | GPT-4, GPT-3.5 | No | Required |
| **Anthropic** | Claude 3 | No | Required |
| **Google** | Gemini Pro | No | Required |
| **Ollama** | Llama, Mistral, etc. | Yes | Not needed |
| **Azure OpenAI** | GPT-4, GPT-3.5 | No | Required |
| **AWS Bedrock** | Claude, Titan | No | Required |
| **Embedded** | llama.cpp | Yes | Not needed |

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
k13d -web
```

### Anthropic (Claude)

```yaml
llm:
  provider: anthropic
  model: claude-3-opus-20240229
  api_key: ${ANTHROPIC_API_KEY}
```

### Google Gemini

```yaml
llm:
  provider: gemini
  model: gemini-pro
  api_key: ${GOOGLE_API_KEY}
```

### Ollama (Local)

Start Ollama:

```bash
ollama serve
ollama pull llama3.2
```

Configure k13d:

```yaml
llm:
  provider: ollama
  model: llama3.2
  endpoint: http://localhost:11434
```

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

### Embedded LLM

Run without any external API:

```bash
k13d --embedded-llm -web
```

Configuration:

```yaml
llm:
  provider: embedded
  model: llama-3.2-1b  # Built-in model
```

## Multi-Model Configuration

Configure multiple models and switch between them:

```yaml
models:
  - name: gpt-4
    provider: openai
    model: gpt-4
    api_key: ${OPENAI_API_KEY}

  - name: local-llama
    provider: ollama
    model: llama3.2
    endpoint: http://localhost:11434

  - name: claude
    provider: anthropic
    model: claude-3-opus-20240229
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

## Provider Features

### Feature Comparison

| Feature | OpenAI | Anthropic | Gemini | Ollama |
|---------|--------|-----------|--------|--------|
| Streaming | ✅ | ✅ | ✅ | ✅ |
| Tool Calling | ✅ | ✅ | ✅ | ✅ |
| Vision | ✅ | ✅ | ✅ | ⚠️ |
| Context Length | 128K | 200K | 32K | Varies |

### Tool Calling Support

All providers support tool calling for kubectl and bash integration:

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
  model: llama3.2
  endpoint: http://localhost:11434
```

### For Air-Gapped

```yaml
llm:
  provider: embedded
  model: llama-3.2-1b
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
k13d -web

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

- [Embedded LLM](embedded.md) - Run without API
- [Tool Calling](tool-calling.md) - How AI executes commands
- [Benchmarks](benchmarks.md) - Model performance comparison
