# Embedded LLM

k13d can run with an embedded LLM for complete offline operation without any external API dependencies.

## Overview

The embedded LLM feature allows k13d to:

- **Run Offline**: No internet connection required
- **Zero Cost**: No API fees
- **Full Privacy**: Data never leaves your machine
- **Air-Gapped**: Suitable for restricted environments

## Quick Start

```bash
# Run with embedded LLM
k13d --embedded-llm

# Web mode with embedded LLM
k13d --embedded-llm -web -port 8080
```

## How It Works

```
┌─────────────────────────────────────────────────────────────────┐
│                         k13d Binary                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │   UI Layer   │───▶│  AI Agent    │───▶│ Embedded LLM │       │
│  │ (TUI/Web)    │    │              │    │ (llama.cpp)  │       │
│  └──────────────┘    └──────────────┘    └──────────────┘       │
│                                                                  │
│                      No External API Calls                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

The embedded LLM uses llama.cpp integrated directly into the k13d binary.

## System Requirements

### Minimum Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| **RAM** | 4 GB | 8+ GB |
| **CPU** | 4 cores | 8+ cores |
| **Disk** | 2 GB free | 5+ GB free |

### Hardware Acceleration

| Platform | Acceleration | Performance |
|----------|--------------|-------------|
| **Apple Silicon** | Metal | Excellent |
| **NVIDIA GPU** | CUDA | Excellent |
| **AMD GPU** | ROCm | Good |
| **CPU Only** | AVX2/AVX512 | Moderate |

## Configuration

### Basic Configuration

```yaml
# ~/.config/k13d/config.yaml
llm:
  provider: embedded
  model: llama-3.2-1b
```

### Advanced Configuration

```yaml
llm:
  provider: embedded
  model: llama-3.2-1b

  embedded:
    # Model parameters
    context_length: 4096
    threads: 8              # CPU threads
    gpu_layers: 32          # Layers to offload to GPU

    # Generation parameters
    temperature: 0.7
    top_p: 0.9
    top_k: 40
    repeat_penalty: 1.1

    # Performance
    batch_size: 512
    use_mmap: true
    use_mlock: false
```

## Available Models

### Built-in Models

| Model | Size | Quality | Speed |
|-------|------|---------|-------|
| `llama-3.2-1b` | 1.2 GB | Good | Fast |
| `llama-3.2-3b` | 2.5 GB | Better | Moderate |
| `qwen2-1.5b` | 1.5 GB | Good | Fast |

### Using Custom Models

Download GGUF models from HuggingFace:

```bash
# Download model
wget https://huggingface.co/TheBloke/Llama-2-7B-GGUF/resolve/main/llama-2-7b.Q4_K_M.gguf

# Configure k13d
cat << EOF >> ~/.config/k13d/config.yaml
llm:
  provider: embedded
  embedded:
    model_path: /path/to/llama-2-7b.Q4_K_M.gguf
EOF
```

## Performance Tuning

### For Speed

```yaml
llm:
  provider: embedded
  model: llama-3.2-1b
  embedded:
    context_length: 2048    # Smaller context
    threads: 8              # Use all CPU cores
    gpu_layers: 32          # GPU offload
    batch_size: 1024        # Larger batches
```

### For Quality

```yaml
llm:
  provider: embedded
  model: llama-3.2-3b      # Larger model
  embedded:
    context_length: 8192    # More context
    temperature: 0.3        # More deterministic
```

### For Low Memory

```yaml
llm:
  provider: embedded
  model: llama-3.2-1b
  embedded:
    context_length: 1024
    use_mmap: true
    gpu_layers: 0           # CPU only
```

## GPU Acceleration

### NVIDIA CUDA

```bash
# Ensure CUDA is installed
nvidia-smi

# Run with GPU
k13d --embedded-llm -web
```

### Apple Metal

Automatically enabled on Apple Silicon Macs.

### Check GPU Usage

```bash
# NVIDIA
nvidia-smi -l 1

# macOS
sudo powermetrics --samplers gpu_power
```

## Comparison with API Providers

| Aspect | Embedded | OpenAI GPT-4 |
|--------|----------|--------------|
| **Cost** | Free | ~$30/1M tokens |
| **Latency** | 2-10s | 2-5s |
| **Quality** | Good | Excellent |
| **Privacy** | Complete | API calls |
| **Offline** | Yes | No |
| **Setup** | Simple | API key needed |

## Use Cases

### Air-Gapped Environments

```bash
# Copy binary to air-gapped machine
scp k13d air-gapped-server:/usr/local/bin/

# Run on air-gapped machine
k13d --embedded-llm -web
```

### Development/Testing

```bash
# Quick testing without API costs
k13d --embedded-llm
```

### Privacy-Sensitive Data

```bash
# Kubernetes cluster with sensitive data
k13d --embedded-llm -web
# All AI analysis happens locally
```

## Limitations

### Compared to Cloud LLMs

| Limitation | Impact | Mitigation |
|------------|--------|------------|
| Smaller context | May miss details | Summarize inputs |
| Lower quality | Less accurate | Use larger model |
| No vision | Can't analyze images | Use text descriptions |
| Slower first response | Initial delay | Preload model |

### Hardware Dependent

Performance varies significantly based on hardware:

- **High-end laptop**: 5-10 tokens/second
- **Desktop with GPU**: 30-50 tokens/second
- **Apple M3 Max**: 50-100 tokens/second

## Troubleshooting

### Out of Memory

```
Error: failed to allocate memory
```

Solutions:
1. Use smaller model
2. Reduce context length
3. Enable mmap: `use_mmap: true`
4. Close other applications

### Slow Responses

```
Response taking >30 seconds
```

Solutions:
1. Use smaller model
2. Enable GPU offload
3. Increase thread count
4. Reduce context length

### Model Not Found

```
Error: model file not found
```

Solutions:
1. Check model_path in config
2. Ensure file exists
3. Check file permissions

### GPU Not Detected

```
Warning: GPU not available, using CPU
```

Solutions:
1. Install CUDA/ROCm drivers
2. Verify GPU with `nvidia-smi`
3. Set `gpu_layers: 32` in config

## Hybrid Mode

Use embedded for simple queries, API for complex ones:

```yaml
models:
  - name: embedded
    provider: embedded
    model: llama-3.2-1b

  - name: gpt-4
    provider: openai
    model: gpt-4

# Start with embedded
active_model: embedded
```

Switch when needed:
```
:model gpt-4  # Complex analysis
:model embedded  # Simple queries
```

## Best Practices

### 1. Choose Right Model Size

- **1-3B**: Fast, basic tasks
- **7-13B**: Balanced quality/speed
- **30B+**: High quality, slow

### 2. Optimize for Hardware

Match configuration to your hardware capabilities.

### 3. Preload Model

Start k13d in advance to avoid first-query delay.

### 4. Monitor Resources

Watch memory and CPU during operation.

## Next Steps

- [LLM Providers](providers.md) - Compare all providers
- [Tool Calling](tool-calling.md) - AI command execution
- [Benchmarks](benchmarks.md) - Model comparisons
