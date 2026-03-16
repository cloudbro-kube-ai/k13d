# Quick Start

This guide will get you up and running with k13d in under 5 minutes.

!!! info "Scope of this quick start"
    This quick start covers the **currently supported path**: running the k13d **binary locally** for the **TUI** and **Web UI**.
    Docker, Docker Compose, Kubernetes, and Helm deployment flows are still Beta / in preparation.

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
    # Start web server with local auth
    ./k13d --web --auth-mode local --port 8080

    # Open in browser
    open http://localhost:8080
    ```

    Default username: `admin` — a random password is generated and printed in the terminal on startup.
    In `--auth-mode local`, the login screen shows the username/password form only. The token form is used with `--auth-mode token`.

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

Default config file path:

- Linux: `${XDG_CONFIG_HOME:-~/.config}/k13d/config.yaml`
- macOS: `~/.config/k13d/config.yaml`
- Windows: `%AppData%\\k13d\\config.yaml`

You can override that with `--config /path/to/config.yaml` or `K13D_CONFIG=/path/to/config.yaml`.

=== "Upstage Solar (Recommended)"

    ```yaml title="~/.config/k13d/config.yaml"
    llm:
      provider: upstage
      model: solar-pro2
      endpoint: https://api.upstage.ai/v1
      api_key: ${UPSTAGE_API_KEY}
    ```

=== "OpenAI"

    ```yaml title="~/.config/k13d/config.yaml"
    llm:
      provider: openai
      model: gpt-4o
      endpoint: https://api.openai.com/v1
      api_key: ${OPENAI_API_KEY}
    ```

=== "Ollama (Local)"

    ```bash
    # Start Ollama first
    ollama pull gpt-oss:20b
    ```

    ```yaml title="~/.config/k13d/config.yaml"
    llm:
      provider: ollama
      model: gpt-oss:20b
      endpoint: http://localhost:11434
    ```

    Use an Ollama model that explicitly supports **tools/function calling**. Text-only Ollama models may connect, but the k13d AI Assistant will not work correctly.

The file is only created when you first save settings if it does not already exist. To verify the active path at startup, check the startup log lines `Config File`, `Config Path Source`, `Env Overrides`, and `LLM Settings`.

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
