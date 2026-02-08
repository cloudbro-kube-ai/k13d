# Quick Start

This guide will get you up and running with k13d in under 5 minutes.

## Prerequisites

Before you begin, ensure you have:

- [x] A running Kubernetes cluster
- [x] `kubectl` configured and working (`kubectl get nodes` should work)
- [x] k13d installed ([Installation Guide](installation.md))

---

## Step 1: Start k13d

=== "TUI Mode"

    ```bash
    # Default mode - opens terminal dashboard
    ./k13d
    ```

    You'll see a k9s-style interface with your cluster resources.

=== "Web Mode"

    ```bash
    # Start web server
    ./k13d -web -port 8080

    # Open in browser
    open http://localhost:8080
    ```

    Default credentials: `admin` / `admin123`

---

## Step 2: Navigate Resources

### TUI Navigation

| Key | Action |
|-----|--------|
| `j` / `k` | Move down / up |
| `Enter` | Select / drill down |
| `Esc` | Go back |
| `:pods` | Switch to pods view |
| `:svc` | Switch to services view |
| `/` | Filter current view |
| `Tab` | Toggle AI panel |
| `?` | Show help |

### Web Navigation

1. Use the sidebar to select resource types
2. Click on a resource to view details
3. Use the AI chat panel on the right for assistance

---

## Step 3: Try the AI Assistant

Press `Tab` (TUI) or use the chat panel (Web) to open the AI assistant.

Try these example prompts:

```
Show me pods that are not running
```

```
What deployments are in the default namespace?
```

```
Explain why this pod is in CrashLoopBackOff
```

The AI will:

1. Understand your request
2. Execute the appropriate kubectl commands
3. Provide a clear explanation

---

## Step 4: Configure LLM (Optional)

For the best AI experience, configure an LLM provider:

=== "Upstage Solar (Recommended)"

    ```yaml title="~/.config/k13d/config.yaml"
    llm:
      provider: solar
      model: solar-pro2
      endpoint: https://api.upstage.ai/v1
      api_key: your-upstage-api-key
    ```

=== "OpenAI"

    ```yaml title="~/.config/k13d/config.yaml"
    llm:
      provider: openai
      model: gpt-4
      api_key: your-openai-api-key
    ```

=== "Ollama (Local)"

    ```bash
    # Start Ollama first
    ollama pull qwen2.5:3b
    ```

    ```yaml title="~/.config/k13d/config.yaml"
    llm:
      provider: ollama
      model: qwen2.5:3b
      endpoint: http://localhost:11434/v1
    ```

---

## Common Operations

### View Pod Logs

=== "TUI"
    1. Navigate to the pod
    2. Press `l`

=== "Web"
    1. Click on a pod
    2. Click "Logs" tab

=== "AI"
    ```
    Show me the logs for pod nginx-xxx
    ```

### Scale a Deployment

=== "TUI"
    1. Navigate to the deployment
    2. Press `s`
    3. Enter new replica count

=== "AI"
    ```
    Scale deployment nginx to 3 replicas
    ```

### Describe a Resource

=== "TUI"
    1. Select the resource
    2. Press `d`

=== "AI"
    ```
    Describe deployment nginx
    ```

---

## Next Steps

<div class="grid cards" markdown>

-   :material-cog:{ .lg .middle } __Configuration__

    ---

    Full configuration options

    [:octicons-arrow-right-24: Configuration](configuration.md)

-   :material-console:{ .lg .middle } __TUI Guide__

    ---

    Master all TUI features

    [:octicons-arrow-right-24: TUI Dashboard](../user-guide/tui.md)

-   :material-robot:{ .lg .middle } __AI Features__

    ---

    Advanced AI capabilities

    [:octicons-arrow-right-24: AI Assistant](../concepts/ai-assistant.md)

</div>
