# Model Settings & Storage

This page explains how k13d stores AI model settings, how the **Web UI** and **TUI** change them, and what is written back to disk when you click **Save** or switch profiles.

## Short Version

- The single source of truth for model configuration is `config.yaml`.
- The default path is `<XDG config home>/k13d/config.yaml`.
- You can override that path with `--config /path/to/config.yaml` or `K13D_CONFIG=/path/to/config.yaml`.
- Web UI and TUI both save by rewriting that YAML file.
- The current build does **not** use SQLite as the authoritative source for active model settings.
- Saving from Web UI or TUI takes effect immediately and recreates the in-process AI client. No restart is required.
- The file is created on first save if it does not already exist.

## Source Of Truth

k13d loads model configuration from the config directory:

| Platform | Default path |
|----------|--------------|
| Linux | `${XDG_CONFIG_HOME:-~/.config}/k13d/config.yaml` |
| macOS | `~/.config/k13d/config.yaml` |
| Windows | `%AppData%\\k13d\\config.yaml` |

If you previously used a macOS build that wrote `~/Library/Application Support/k13d/config.yaml`, k13d copies that legacy file into `~/.config/k13d/config.yaml` on first startup.

The default file is:

```text
~/.config/k13d/config.yaml
```

or from the path selected with:

```bash
k13d --config /path/to/config.yaml
```

or:

```bash
export K13D_CONFIG=/path/to/config.yaml
```

The path resolution order is:

1. `--config /path/to/config.yaml`
2. `K13D_CONFIG=/path/to/config.yaml`
3. `XDG_CONFIG_HOME=/custom/config-home` -> `$XDG_CONFIG_HOME/k13d/config.yaml`
4. macOS default `~/.config/k13d/config.yaml`
5. platform XDG/AppData default

When k13d saves configuration, it creates the parent directory if needed and writes the file with mode `0600`.

When you start Web UI mode, the terminal also prints `Config File`, `Config Path Source`, and `Env Overrides`. Check those lines first if the Web UI looks out of sync with the file you edited.

!!! note "SQLite is not the active config source"
    k13d creates SQLite tables such as `web_settings` and `model_profiles`, but current Web UI and TUI model configuration is still read from `config.yaml`. Those tables do not override the runtime LLM settings in the current build.

## Missing File Behavior

If the selected `config.yaml` does not exist yet:

- k13d loads built-in defaults
- it still applies `K13D_LLM_*` environment overrides
- it does not create the file just from reading it
- the file appears only after the first save

This is true for both the default path and an explicit custom `--config` path.

## What Lives In `config.yaml`

Three parts matter for model configuration:

```yaml title="~/.config/k13d/config.yaml"
llm:
  provider: upstage
  model: solar-pro2
  endpoint: https://api.upstage.ai/v1
  api_key: ${UPSTAGE_API_KEY}
  reasoning_effort: minimal
  max_iterations: 10

models:
  - name: solar-pro2
    provider: upstage
    model: solar-pro2
    endpoint: https://api.upstage.ai/v1
    api_key: ${UPSTAGE_API_KEY}
    description: "Upstage Solar Pro2"

  - name: gpt-oss-local
    provider: ollama
    model: gpt-oss:20b
    endpoint: http://localhost:11434
    description: "Local Ollama"

active_model: solar-pro2
```

### What each section means

| Section | Purpose |
|---------|---------|
| `llm` | The currently active runtime LLM connection used to create the AI client |
| `models` | Named saved profiles that can be switched with Web UI or `:model` |
| `active_model` | The name of the selected profile in `models` |

## Global vs Per-Profile Fields

This distinction is important:

### Stored in `llm` only

These fields are **global runtime settings**, not per-profile settings:

- `reasoning_effort`
- `use_json_mode`
- `retry_enabled`
- `max_retries`
- `max_backoff`
- `temperature`
- `max_tokens`
- `max_iterations`

When you switch from one saved profile to another, those fields stay in the `llm` section unless you change them separately.

### Stored in `models[]`

Saved model profiles store:

- `name`
- `provider`
- `model`
- `endpoint`
- `api_key`
- `region`
- `azure_deployment`
- `skip_tls_verify`
- `description`

!!! note "Some advanced fields are YAML-first today"
    The current Web UI and TUI focus on the common fields such as provider, model, endpoint, API key, and profile metadata. If you need advanced provider-specific fields like `region` or `azure_deployment`, edit `config.yaml` directly.

## What Each Action Changes

| Action | Updates `llm` | Updates `models[]` | Updates `active_model` | Restart needed |
|--------|----------------|--------------------|------------------------|----------------|
| Web UI: Save current LLM settings | Yes | Syncs active profile if it exists | No | No |
| Web UI: Add Model Profile | No | Yes | No | No |
| Web UI: Use profile | Yes | No | Yes | No |
| Web UI: Delete profile | Sometimes | Yes | Sometimes | No |
| TUI: `Shift+O` Save | Yes | Syncs active profile if it exists | No | No |
| TUI: `:model` or `:model <name>` | Yes | No | Yes | No |

## Web UI Behavior

### Edit the current LLM connection

Path:

```text
Settings -> AI
```

The main LLM form edits the current runtime connection:

- provider
- model
- endpoint
- API key
- reasoning effort where supported

When you click **Save Settings**, the Web UI sends:

- `PUT /api/settings` for general settings such as language/timezone
- `PUT /api/settings/llm` for the active LLM connection

What happens next:

1. `llm.*` is updated in memory.
2. If `active_model` points to an existing profile, k13d copies the active connection fields back into that profile.
3. The AI client is recreated immediately.
4. `config.yaml` is rewritten.

### Important nuance: blank API key fields

The Web UI does not echo the stored API key back into the form. The field is blank on load.

If you save the form with the API key field still blank:

- k13d keeps the current in-memory API key
- it does **not** clear the key
- the subsequent save can still write that key to `config.yaml`

This matters if the original key came from `${ENV_VAR}` or from a `K13D_LLM_*` override.

### Important nuance: placeholder expansion and rewrites

`config.yaml` supports values such as `${OPENAI_API_KEY}`. k13d expands those placeholders when it loads the file. If you later save from Web UI or TUI, the rewritten YAML may contain the resolved in-memory value instead of the original placeholder string.

If preserving placeholders matters to you:

- use environment variables as the runtime source of truth
- or keep a template copy of `config.yaml` outside k13d
- or avoid saving secret-bearing fields from Web UI/TUI

### Add a saved model profile

Path:

```text
Settings -> AI -> Add Model Profile
```

The form uses the same provider list as the main LLM settings form and is prefilled from the current provider, model, and endpoint to make profile creation less error-prone.

This writes a new entry into `models[]`.

It does **not**:

- change `llm`
- change `active_model`
- switch the AI client automatically

After adding a profile, click **Use** if you want it to become active.

### Switch the active profile

Path:

```text
Settings -> AI -> Use
```

This calls `PUT /api/models/active` and:

1. sets `active_model`
2. copies the selected profile into `llm.provider`, `llm.model`, `llm.endpoint`, `llm.api_key`, and related profile-backed fields
3. recreates the AI client
4. saves `config.yaml`

### Delete a profile

Path:

```text
Settings -> AI -> Delete
```

This removes the entry from `models[]` and saves the file.

If the deleted profile was active:

- if another profile still exists, the first remaining profile becomes active
- if no profile remains, `active_model` becomes empty

If the last profile is deleted, the `llm` section is left as-is, so the current runtime connection can still remain configured even though no named profiles exist anymore.

## TUI Behavior

### Edit the current LLM connection

Open the TUI settings modal with:

```text
Shift+O
```

The TUI settings screen currently edits the common connection fields:

- provider
- model
- endpoint
- API key

When you press **Save**:

1. `llm.provider`, `llm.model`, `llm.endpoint`, and optionally `llm.api_key` are updated
2. the active profile is synced from `llm` if `active_model` matches an existing profile
3. `config.yaml` is rewritten
4. the AI client is recreated immediately

The write path is exactly the same active path used by the Web UI. There is no separate TUI-only config file.

The TUI settings modal does **not** create or delete named model profiles.

### Switch profiles from the TUI

Use:

```text
:model
```

or:

```text
:model <name>
```

The TUI reloads `config.yaml` from disk before showing or switching profiles, which helps it pick up recent Web UI changes.

When you switch:

1. `active_model` is changed
2. the selected profile is copied into `llm`
3. the file is saved
4. the AI client is recreated

## Environment Variables And Placeholder Behavior

Load order matters:

1. k13d loads `config.yaml`
2. shell-style placeholders such as `${OPENAI_API_KEY}` are expanded
3. `K13D_*` environment overrides are applied on top

That means the in-memory config already contains resolved values by the time you open Web UI or TUI settings.

!!! warning "Saving will serialize the current in-memory values"
    If your file originally contains `${OPENAI_API_KEY}` or if you launched k13d with `K13D_LLM_API_KEY`, then saving from Web UI or TUI can write the resolved literal key back into `config.yaml`.

If you want to keep placeholder-based secrets in the file:

- edit `config.yaml` manually
- keep using `${ENV_VAR}` placeholders there
- avoid saving the LLM form from Web UI or TUI after launch

## Practical Examples

### Example 1: switch profiles

Before:

```yaml
llm:
  provider: upstage
  model: solar-pro2
  endpoint: https://api.upstage.ai/v1

models:
  - name: solar-pro2
    provider: upstage
    model: solar-pro2
    endpoint: https://api.upstage.ai/v1

  - name: gpt-oss-local
    provider: ollama
    model: gpt-oss:20b
    endpoint: http://localhost:11434

active_model: solar-pro2
```

After selecting `gpt-oss-local`:

```yaml
llm:
  provider: ollama
  model: gpt-oss:20b
  endpoint: http://localhost:11434

models:
  - name: solar-pro2
    provider: upstage
    model: solar-pro2
    endpoint: https://api.upstage.ai/v1

  - name: gpt-oss-local
    provider: ollama
    model: gpt-oss:20b
    endpoint: http://localhost:11434

active_model: gpt-oss-local
```

### Example 2: save the active connection from Settings

If `active_model: gpt-oss-local` already exists and you change the model from `gpt-oss:20b` to `gpt-oss:120b`, then saving from Web UI or TUI updates:

- `llm.model`
- `models[].model` for `gpt-oss-local`

It does **not** automatically create a new named profile.

## Recommended Workflow

- Keep reusable named models in `models[]`.
- Use the Web UI **Add Model Profile** form to build your profile catalog.
- Use Web UI **Use** or TUI `:model` for daily switching.
- Use the LLM settings form to edit the current active connection.
- If you rely on environment-variable placeholders for secrets, prefer manual YAML edits over UI saves.

## Related Pages

- [Configuration](../getting-started/configuration.md)
- [LLM Providers](providers.md)
- [Web Dashboard](../user-guide/web.md)
- [TUI Dashboard](../user-guide/tui.md)
