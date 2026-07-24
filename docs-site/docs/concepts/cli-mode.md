# CLI Mode

## Overview

CLI Mode is an interactive REPL (Read-Eval-Print Loop) interface for k13d. It provides a lightweight, terminal-based command-line experience that sits between the full TUI dashboard and one-off `kubectl` commands. When launched, it displays a centered ASCII art splash screen and an input prompt for executing Kubernetes operations.

This mode is designed for users who want a quick, keyboard-driven interface without the overhead of the full TUI dashboard, or for scripting/automation contexts where the Web UI is unnecessary.

## User Experience

### Startup

When `k13d` is launched with `--cli` flag (or when `K13D_CLI=true` is set), the terminal clears and displays:

```
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║                                                              ║
║                                                              ║
║                    _    ___  _     _                         ║
║                   | | / / || |   | |                        ║
║                   | |/ /| || | __| |                        ║
║                   |    \| || |/ _` |                        ║
║                   | |\  \__/ | (_| |                        ║
║                   \_| \_/___/ \__,_|                        ║
║                                                              ║
║                      Kubernetes CLI                          ║
║                                                              ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝

k13d> _
```

- The ASCII art is rendered in the center of the terminal (both horizontally and vertically).
- Below the art, a single prompt line `k13d> ` awaits user input.
- Terminal dimensions are detected at startup; the splash adapts to fit.

### Interaction Model

The CLI mode follows a REPL pattern:

| Input | Behavior |
|-------|----------|
| `k13d> get pods` | Executes `kubectl get pods`, prints output inline |
| `k13d> get pods -n kube-system` | Supports kubectl-style flags and arguments |
| `k13d> :help` | Shows available CLI commands |
| `k13d> :quit` or `k13d> :exit` | Exits the CLI mode |
| `k13d> :clear` | Clears screen, re-displays splash |
| `k13d> :version` | Shows k13d version |
| `k13d> :namespace default` | Sets default namespace for subsequent commands |
| `k13d> :context my-cluster` | Switches Kubernetes context |
| `k13d> :model` | Shows AI model profile selector |
| `k13d> :model gpt-4o` | Switches to a named AI model profile directly |
| `k13d> :history` | Shows command history |
| `↑` / `↓` | Navigate command history |
| `←` / `→` | Move cursor within the current input line for editing |
| `Tab` | Auto-complete commands and resource names |
| `Ctrl+C` or `Esc` | Cancel current command / exit |
| `Ctrl+L` | Clear screen |
| `Ctrl+D` | Exit (on empty prompt) |
All `kubectl` commands are forwarded to the underlying `kubectl` binary or Kubernetes client API. Output is printed directly to stdout.

### Output

Command output is displayed inline between the splash and the prompt:

```
                    _    ___  _     _
                   | | / / || |   | |
                   | |/ /| || | __| |
                   |    \| || |/ _` |
                   | |\  \__/ | (_| |
                   \_| \_/___/ \__,_|

                      Kubernetes CLI

────────────────────────────────────────────────────────────────
NAMESPACE     NAME                    READY   STATUS    RESTARTS
default       nginx-7854ff8877-6kzjz  1/1     Running   0
default       redis-6b7f6f5d9c-x8m2p  1/1     Running   0
kube-system   coredns-1234abcd56-xyz9  1/1     Running   0
────────────────────────────────────────────────────────────────

k13d> get pods
```

- Long output is paginated with `--more--` prompt at the bottom (press Space to continue, Q to quit).
- Output is scrollable if it exceeds terminal height.
- Errors are shown in red inline, without leaving the REPL.

## Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────────┐
│                      k13d Binary                              │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐   │
│  │   TUI Mode   │    │   Web Mode   │    │  CLI Mode    │   │
│  │   (tview)    │    │   (HTTP)     │    │  (REPL)      │   │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘   │
│         │                   │                    │           │
│         └───────────────────┼────────────────────┘           │
│                             │                                 │
│                    ┌────────▼────────┐                       │
│                    │   Shared Core    │                       │
│                    ├──────────────────┤                       │
│                    │ • AI Agent       │                       │
│                    │ • K8s Client     │                       │
│                    │ • Tool Registry  │                       │
│                    │ • Safety Analyzer│                       │
│                    │ • Session Store  │                       │
│                    │ • Audit Logger   │                       │
│                    │ • Issue Automation│                      │
│                    └──────────────────┘                       │
└─────────────────────────────────────────────────────────────┘
```

### Package Structure

New package to add:

```
pkg/cli/
├── repl.go          # REPL loop: read → eval → print
├── splash.go        # ASCII art rendering and centering
├── commands.go      # :command handlers (:help, :quit, etc.)
├── history.go       # Command history (in-memory + file)
├── completion.go    # Tab completion logic
└── output.go        # Output formatting, pagination
```

No changes to existing shared packages. The CLI mode consumes the same `pkg/k8s`, `pkg/config`, `pkg/ai`, etc. as TUI and Web modes.

### Key Components

#### 1. Entry Point (`pkg/cli/repl.go`)

A `Start(cfg *config.Config)` function that:

1. Initializes the Kubernetes client from shared `pkg/k8s`
2. Clears the terminal
3. Renders the splash screen via `splash.Render()`
4. Enters the read-eval-print loop

The REPL loop:
- Reads a line of input from the prompt
- Parses it into a command (either `:builtin` or raw kubectl)
- Executes via the appropriate handler
- Prints output
- Loops until `:quit`, `:exit`, or `Ctrl+D`/`Ctrl+C`/`Esc`

#### 2. Splash Screen (`pkg/cli/splash.go`)

- Embeds the ASCII art as a raw string constant
- Detects terminal width/height via `tcell` or `golang.org/x/term`
- Calculates vertical and horizontal padding to center the art
- Renders to a string buffer, optionally with a border box

Implementation approach for centering:

```go
func Render(width, height int) string {
    art := strings.Split(k13dAsciiArt, "\n")
    artHeight := len(art)
    artWidth := maxLineWidth(art)

    vertPad := (height - artHeight) / 2
    horizPad := (width - artWidth) / 2

    var buf strings.Builder
    // Top padding (blank lines)
    for i := 0; i < vertPad; i++ {
        buf.WriteString(strings.Repeat(" ", width) + "\n")
    }
    // Art lines with horizontal padding
    for _, line := range art {
        buf.WriteString(strings.Repeat(" ", horizPad))
        buf.WriteString(line)
        buf.WriteString("\n")
    }
    // Bottom padding
    for i := 0; i < vertPad; i++ {
        buf.WriteString(strings.Repeat(" ", width) + "\n")
    }
    return buf.String()
}
```

**Terminal detection**: Use `golang.org/x/term` (already in Go stdlib as `golang.org/x/term`) to get terminal size. Fallback to 80x24 if not available.

#### 3. Command Routing (`pkg/cli/commands.go`)

Commands are categorized:

**Built-in commands** (prefixed with `:`):
| Command | Handler | Description |
|---------|---------|-------------|
| `:help` | `cmdHelp()` | List all commands |
| `:quit` / `:exit` | `cmdQuit()` | Exit REPL |
| `:clear` | `cmdClear()` | Clear screen + re-splash |
| `:version` | `cmdVersion()` | Print k13d version |
| `:namespace <ns>` | `cmdNamespace()` | Set default namespace |
| `:context <ctx>` | `cmdContext()` | Switch k8s context |
| `:history` | `cmdHistory()` | Show command history |
| `:ai <prompt>` | `cmdAI()` | Send prompt to AI agent |

**Raw commands**: Any input not starting with `:` is treated as a kubectl command. It is sent through `pkg/k8s` client or executed via the configured kubectl binary.

#### 4. History (`pkg/cli/history.go`)

- In-memory ring buffer (last 500 commands)
- Optional persistence to `~/.k13d/cli_history` file
- Navigation via `↑`/`↓` in the REPL

#### 5. Tab Completion (`pkg/cli/completion.go`)

- Completes `:commands` when input starts with `:`
- Completes Kubernetes resource names (pods, deployments, services, etc.)
- Completes resource names by querying the cluster

#### 6. Output (`pkg/cli/output.go`)

- Wraps command output for terminal display
- Implements pagination for multi-page output
- Colorizes errors and warning output

### Main Entry Point Changes (`cmd/kube-ai-dashboard-cli/main.go`)

Add a new flag:

```go
cliMode := flag.Bool("cli", cli.EnvBoolDefault("K13D_CLI", false), "Start CLI REPL mode")
```

Mode selection logic (inserted after MCP mode check, before Web mode check):

```go
// CLI REPL mode
if *cliMode {
    runCLI(cfg)
    return
}
```

New `runCLI` function:

```go
func runCLI(cfg *config.Config) {
    defer cli.InitDB(cfg)()

    cliRepl := cli.New(cfg)
    if err := cliRepl.Start(); err != nil {
        fmt.Fprintf(os.Stderr, "CLI error: %v\n", err)
        os.Exit(1)
    }
}
```

## Implementation Steps

### Step 1: Create Package Structure

Create `pkg/cli/` directory with stub files. Only `repl.go` and `splash.go` needed for MVP.

### Step 2: Implement Splash Screen

- Define the ASCII art constant
- Implement centering logic using `golang.org/x/term`
- Support terminal resize detection (SIGWINCH)

### Step 3: Implement Basic REPL Loop

- Use `bufio.Scanner` or `github.com/chzyer/readline` for line input
  - Recommendation: `github.com/chzyer/readline` provides history, auto-complete, and key bindings out of the box
  - Alternative: `github.com/peterh/liner` (lighter, used by `sqlite3` CLI)
- Parse input into command + args
- Route to handlers

### Step 4: Wire Up Kubectl Execution

- Forward raw commands to `k8s.Client` or shell out to `kubectl`
- Capture stdout/stderr and display inline
- Implement output pagination

### Step 5: Built-in Commands

- Implement `:help`, `:quit`, `:clear`, `:version`, `:namespace`, `:context`

### Step 6: History & Completion

- Add in-memory history with file persistence
- Add tab completion for `:` commands and k8s resources

### Step 7: AI Integration (Optional)

- Wire `:ai` command to `pkg/ai` client for quick AI queries

### Step 8: Integration & Flag

- Add `--cli` flag to `main.go`
- Add `K13D_CLI` env var support
- Update docs: architecture diagram, CLI reference, concepts/cli-mode.md

## Dependencies

### Required (all already in go.mod or stdlib)
| Dependency | Purpose |
|------------|---------|
| `golang.org/x/term` | Terminal size detection (stdlib) |
| `bufio` (stdlib) | Line input |
| `os/exec` (stdlib) | Kubectl execution fallback |

### Optional (for better UX)
| Dependency | Purpose |
|------------|---------|
| `github.com/chzyer/readline` | History, auto-complete, key bindings |
| or `github.com/peterh/liner` | Lighter alternative |

## Edge Cases & Error Handling

| Scenario | Behavior |
|----------|----------|
| Terminal too small for splash | Print minimal `k13d CLI` text, show prompt |
| Non-TTY output (piped) | Disable splash, raw output mode |
| Kubectl not installed | Error message with install hint, stay in REPL |
| Network error on command | Print error, stay in REPL |
| Empty input (Enter on empty) | Ignore, re-show prompt |
| Very long output | Paginate with `--more--` |
| Background process | Handle `SIGTSTP` properly |

## Future Enhancements

- **Pipeline support**: `get pods | grep nginx`
- **Output format flags**: Support `-o wide`, `-o yaml`, `-o json`
- **Multi-line input**: For complex commands or YAML editing
- **AI chat mode**: Continuous chat with AI agent within CLI
- **Script mode**: `k13d --cli script.k13d` for batch commands
- **Custom prompt**: Configurable prompt via config.yaml

## Testing Strategy

| Test Type | Scope |
|-----------|-------|
| Unit | Splash centering, command parsing, history ring buffer |
| Integration | REPL with mock stdin/stdout, command routing |
| E2E | Full startup → command → output → exit flow |
| TTY simulation | Pty-based tests for terminal size detection |

## Related Documents

- [Architecture](architecture.md) — System architecture showing CLI Mode integration
- [CLI Reference](../reference/cli.md) — CLI flag reference
- [AI Assistant](ai-assistant.md) — AI agent integration
- [TUI Dashboard](../user-guide/tui.md) — Full TUI dashboard guide
