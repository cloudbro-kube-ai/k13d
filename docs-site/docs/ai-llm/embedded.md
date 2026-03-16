# Embedded LLM Removal

Embedded LLM support has been removed from k13d.

## Why It Was Removed

- The response quality was too limited compared with other supported providers
- It added maintenance cost across the CLI, Web UI, docs, and packaging
- Ollama provides a better local/self-hosted path with clearer operational boundaries

## What To Use Instead

### Local / Private Inference

```bash
ollama serve
ollama pull gpt-oss:20b
```

```yaml
llm:
  provider: ollama
  model: gpt-oss:20b
  endpoint: http://localhost:11434
```

### Remote Providers

Use any supported remote provider such as OpenAI, Anthropic, Gemini, Solar, Azure OpenAI, or Bedrock.

## Migration

If an old config still contains:

```yaml
llm:
  provider: embedded
```

change it to an active provider such as:

```yaml
llm:
  provider: ollama
  model: gpt-oss:20b
  endpoint: http://localhost:11434
```
