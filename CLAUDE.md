# CLAUDE.md
version: 1
default_agent: "@dev-agent"

> **Consulted files (agent must populate before committing):**
>
> **README.md files:**
> - `/README.md` - Main project README with features and getting started
> - `/docs/USER_GUIDE.md` - User guide with navigation and shortcuts
> - `/docs/CONFIGURATION_GUIDE.md` - Configuration guide
> - `/docs/MCP_GUIDE.md` - MCP server architecture and integration
>
> **Documentation files:**
> - `/docs/INSTALLATION.md` - Installation overview (binary, docker, k8s)
> - `/docs/INSTALLATION_DOCKER.md` - Docker and Docker Compose setup
> - `/docs/INSTALLATION_K8S.md` - Kubernetes deployment by version
> - `/docs/ARCHITECTURE.md` - System architecture overview
> - `/docs/BENCHMARK_RESULTS.md` - AI model benchmark results
> - `/CONTRIBUTING.md` - Contribution guidelines
> - `/SECURITY.md` - Security policy
>
> **Build/Config files:**
> - `/go.mod` - Go version (1.25.0) and dependencies
> - `/Makefile` - Build, test, and deployment commands
> - `/.goreleaser.yaml` - Release configuration
> - `/.github/workflows/ci.yml` - CI/CD workflow

---

## Project Overview

**k13d (kube-ai-dashboard-cli)** is a comprehensive Kubernetes management tool that provides both a terminal-based UI (TUI) and a web-based dashboard with an integrated agentic AI assistant. It combines the TUI experience of [k9s](https://k9scli.io/) with the AI-powered intelligence of [kubectl-ai](https://github.com/GoogleCloudPlatform/kubectl-ai).

### Core Value Proposition
- **Dual Interface**: Both TUI (terminal) and Web UI with same feature set
- **k9s Parity**: Full-featured TUI dashboard with Vim-style navigation
- **kubectl-ai Intelligence**: Agentic AI loop with tool-use (Kubectl, Bash, MCP)
- **Deep Synergy**: AI analysis with full context (YAML + Events + Logs)
- **Enterprise Ready**: Authentication, audit logging, and report generation

---

## Agent Persona and Scope

- **@dev-agent** - pragmatic, conservative, test-first, risk-averse.
- **Scope:** propose, validate, and prepare code/docs patches; run local build/test commands; create PR drafts.
- **Not allowed:** push images/releases, modify CI or infra, or merge without human approval.

---

## Explicit Non-Goals

The agent should NOT do the following unless explicitly requested:
- Propose refactors without a clear bug, performance, or maintenance justification
- Change public APIs without explicit request
- Reformat unrelated code
- Rename files or symbols for stylistic reasons
- Introduce new dependencies unless required to fix a bug or implement a requested feature

---

## Tech Stack and Environment

### Languages & Frameworks
- **Language:** Go 1.25.0+
- **TUI Framework:** [tview](https://github.com/rivo/tview) with [tcell](https://github.com/gdamore/tcell/v2)
- **Web Framework:** Standard library `net/http` with embedded static files
- **AI Integration:** Custom OpenAI-compatible HTTP client (supports OpenAI, Ollama, Anthropic)
- **MCP Integration:** JSON-RPC 2.0 stdio protocol for tool extensibility
- **Kubernetes Client:** client-go v0.35.0, metrics v0.35.0
- **Database:** CGO-free SQLite (modernc.org/sqlite) for audit logs and settings
- **Authentication:** Session-based with SHA256 password hashing

### Key Dependencies
```
github.com/rivo/tview v0.42.0           # TUI framework
github.com/gdamore/tcell/v2 v2.13.6     # Terminal cell handling
github.com/GoogleCloudPlatform/kubectl-ai # AI agent integration
github.com/adrg/xdg v0.5.3               # XDG directory support
modernc.org/sqlite v1.43.0              # CGO-free SQLite
k8s.io/client-go v0.35.0                # Kubernetes client
k8s.io/metrics v0.35.0                  # Metrics API
```

### Skills (Pattern Reference Documents)

| Skill | File | Key Patterns |
|-------|------|--------------|
| k9s Patterns | `skills/k9s-patterns.md` | MVC architecture, Action system, Plugin/HotKey, Skin, XDG config |
| kubectl-ai Patterns | `skills/kubectl-ai-patterns.md` | Agent Loop, Tool System, LLM abstraction, MCP integration |
| Headlamp Patterns | `skills/headlamp-patterns.md` | Plugin Registry, Multi-Cluster, Response Cache, OIDC, i18n |
| K8s Dashboard Patterns | `skills/kubernetes-dashboard-patterns.md` | DataSelector, Multi-Module, Request-Scoped Client |

**Usage Guide**: `skills/README.md`

---

## Repository Structure

```
k13d/
├── cmd/
│   ├── kube-ai-dashboard-cli/main.go   # Main entry point (TUI + Web modes)
│   ├── bench/main.go                   # Benchmark tool
│   └── eval/main.go                    # Evaluation tool
├── pkg/
│   ├── ui/                             # TUI components
│   │   ├── app.go                      # Main application
│   │   ├── app_*.go                    # App lifecycle & callbacks
│   │   ├── dashboard.go                # Resource dashboard view
│   │   ├── assistant.go                # AI assistant panel
│   │   └── resources/                  # Resource-specific views
│   ├── web/                            # Web UI server
│   │   ├── server.go                   # HTTP server & API handlers
│   │   ├── auth.go                     # Authentication system
│   │   └── static/index.html           # Web UI frontend
│   ├── ai/                             # AI client & agent
│   │   ├── client.go                   # OpenAI-compatible HTTP client
│   │   ├── agent/                      # Agent loop implementation
│   │   ├── providers/                  # LLM provider adapters
│   │   ├── tools/                      # Tool definitions (kubectl, bash)
│   │   └── safety/                     # Command safety analysis
│   ├── mcp/                            # MCP (Model Context Protocol)
│   │   └── client.go                   # MCP server connection manager
│   ├── k8s/                            # Kubernetes client wrapper
│   ├── config/                         # Configuration management
│   ├── db/                             # SQLite database layer
│   └── i18n/                           # Internationalization
├── deploy/                             # Deployment configurations
│   ├── docker/                         # Docker and Compose files
│   │   ├── Dockerfile                  # Main Dockerfile
│   │   ├── Dockerfile.bench            # Benchmark runner
│   │   ├── Dockerfile.prebuilt         # Pre-built binary
│   │   ├── docker-compose.yaml         # Main compose file
│   │   ├── docker-compose.test.yaml    # Test environment
│   │   ├── docker-compose.bench.yaml   # Benchmark environment
│   │   └── docker-compose.airgapped.yaml
│   └── kubernetes/                     # Kubernetes manifests
│       ├── deployment.yaml             # Standard deployment
│       ├── local-deployment.yaml       # Local development
│       ├── single-pod.yaml             # Single pod deployment
│       └── single-pod-with-ollama.yaml # With Ollama sidecar
├── docs/                               # Documentation
│   ├── INSTALLATION.md                 # Installation overview
│   ├── INSTALLATION_DOCKER.md          # Docker guide
│   ├── INSTALLATION_K8S.md             # Kubernetes guide
│   ├── MCP_GUIDE.md                    # MCP architecture & usage
│   ├── USER_GUIDE.md                   # User guide
│   ├── CONFIGURATION_GUIDE.md          # Configuration options
│   ├── ARCHITECTURE.md                 # System architecture
│   ├── BENCHMARK_RESULTS.md            # AI benchmark results
│   └── IMPROVEMENTS.md                 # Feature improvements log
├── benchmarks/                         # AI benchmark tasks
│   └── tasks/                          # Task definitions
├── skills/                             # Pattern reference documents
├── scripts/                            # Build and utility scripts
├── tests/                              # Test utilities
│   └── mocks/                          # Mock implementations
├── .github/workflows/                  # CI/CD workflows
│   └── ci.yml                          # Main CI workflow
├── Makefile                            # Build automation
├── .goreleaser.yaml                    # Release configuration
├── go.mod / go.sum                     # Go modules
├── README.md                           # Project overview
├── CHANGELOG.md                        # Version history
├── CONTRIBUTING.md                     # Contribution guidelines
├── SECURITY.md                         # Security policy
└── LICENSE                             # MIT License
```

---

## Primary Entry Points (Exact Commands)

### Build Commands
```bash
# Build main binary
make build
# or: go build -o k13d ./cmd/kube-ai-dashboard-cli/main.go

# Build for all platforms
make build-all

# Build benchmark tool
make bench-build
```

### Run Commands
```bash
# Run TUI mode (default)
./k13d

# Run Web UI mode
./k13d -web -port 8080

# Run with embedded LLM (no API key needed)
./k13d --embedded-llm -web -port 8080

# Run with debug mode
./k13d --debug
```

### Test Commands
```bash
# Run all tests
make test
# or: go test -v -race ./...

# Run with coverage report
make test-coverage

# Run integration tests (requires docker-test-up first)
make docker-test-up
make test-integration
make docker-test-down
```

### Lint & Format Commands
```bash
# Format all Go files
gofmt -s -w .

# Run linters
make lint
# or: golangci-lint run

# Check for issues
go vet ./...
```

---

## MCP (Model Context Protocol) Integration

k13d uses MCP to extend AI capabilities with external tools. See `/docs/MCP_GUIDE.md` for details.

### How MCP Works in k13d

```
┌─────────────────────────────────────────────────────────────────┐
│                           k13d                                   │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │  AI Agent    │───▶│  Tool Router │───▶│  MCP Client  │      │
│  └──────────────┘    └──────────────┘    └──────────────┘      │
│                              │                   │              │
│                              │                   │ JSON-RPC 2.0 │
│                              ▼                   ▼              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Tool Registry                         │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────────────────────┐  │   │
│  │  │ kubectl │  │  bash   │  │   MCP Tools (dynamic)   │  │   │
│  │  └─────────┘  └─────────┘  └─────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
            ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
            │ MCP Server  │ │ MCP Server  │ │ MCP Server  │
            │ (kubectl)   │ │ (database)  │ │ (custom)    │
            └─────────────┘ └─────────────┘ └─────────────┘
```

### MCP Connection Flow

1. k13d spawns MCP server process (e.g., `npx @anthropic/mcp-server-kubernetes`)
2. Establishes stdio communication using JSON-RPC 2.0
3. Sends `initialize` request with protocol version
4. Calls `tools/list` to discover available tools
5. Registers tools in the AI Tool Registry
6. AI can now invoke these tools via `tools/call`

### Key MCP Files

- `pkg/mcp/client.go` - MCP client implementation
- `docs/MCP_GUIDE.md` - Complete MCP documentation

---

## Change Rules and Safety Constraints

### Manual-Review-Only Files
- `.github/workflows/*` - CI workflows
- `.goreleaser.yaml` - Release configuration
- `SECURITY.md` - Security policy
- `LICENSE` - License file

### Pre-Change Checks
1. Run `gofmt -s -w .` to format code
2. Run `go vet ./...` to check for issues
3. Run `golangci-lint run` for linting
4. Run `go test -race ./...` to ensure tests pass
5. Run `go build ./...` to verify compilation

### Dependency Updates
- Run full test suite: `go test ./...`
- Do not bump major versions without approval

---

## Best Practices and Coding Guidelines

### Reduce Solution Size
- Make minimal, surgical changes
- Prefer focused, single-purpose changes
- Break down complex changes into smaller increments

### TUI Development Guidelines (tview/tcell)
- Follow k9s patterns for keybindings and navigation
- Use `tview.Application.QueueUpdateDraw()` for thread-safe UI updates
- Handle resize events gracefully
- Use `tcell.EventKey` for keyboard handling consistently

### AI Integration Guidelines
- Follow kubectl-ai patterns for tool definitions
- Ensure AI-proposed modifications require explicit user approval
- Log all AI tool invocations to audit database
- Handle LLM provider errors gracefully

### MCP Server Guidelines
- MCP servers run as child processes with stdio communication
- Use JSON-RPC 2.0 for all requests/responses
- Handle server process lifecycle properly (start, stop, restart)
- Tag tools with server name for routing

### Testing Best Practices
- Use `httptest.NewServer` for mocking HTTP endpoints
- Prefer testing with real implementations over mocks
- Write tests that validate actual behavior
- Use race detector: `go test -race ./...`

---

## Key Features Reference

### Dashboard Navigation (k9s Parity)
| Key | Action |
|-----|--------|
| `j/k` | Move selection up/down |
| `Left/Right/Tab` | Switch focus between panels |
| `:` | Command mode (e.g., `:pods`, `:svc`) |
| `/` | Filter current table |
| `ESC` | Close modal/return to main view |

### Resource Actions
| Key | Action |
|-----|--------|
| `y` | View YAML manifest |
| `l` | Stream logs (Pods) |
| `d` | Describe resource |
| `L` | AI Analyze |
| `h` | Explain This |
| `s` | Scale replicas |
| `r` | Rollout restart |
| `Ctrl+D` | Delete (with confirmation) |

### Management Commands
| Command | Action |
|---------|--------|
| `:alias` | View all resource aliases |
| `:model` | Switch AI model profile |
| `:model <name>` | Switch to named profile directly |
| `:plugins` | View available plugins |

### AI Assistant Features
- **Context Awareness**: Receives YAML, events, and logs
- **Tool Use**: kubectl, bash, MCP integration
- **Safety**: Dangerous commands require approval
- **Beginner Mode**: Simple explanations for complex resources
- **Chat History**: Previous Q&A preserved within session
- **Model Switching**: Switch LLM profiles via `:model` command

### Configuration Files
| File | Purpose |
|------|---------|
| `config.yaml` | Main config (LLM, language, model profiles) |
| `hotkeys.yaml` | Custom hotkey bindings |
| `plugins.yaml` | External plugins |
| `aliases.yaml` | Resource command aliases |
| `views.yaml` | Per-resource view settings (sort defaults) |

---

## Commit Message Format (Conventional Commits)

```
<type>(<scope>): <description>

[optional body]

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

**Types:**
- `feat:` new features
- `fix:` bug fixes
- `docs:` documentation changes
- `refactor:` code restructuring
- `test:` adding tests
- `chore:` maintenance tasks

---

## Agent Output Checklist

Before creating a patch/PR, ensure:

- [ ] **Summary:** one-line intent and short rationale
- [ ] **Files changed:** explicit file list with rationale
- [ ] **Tests:** List tests added/updated with pass status
- [ ] **Local validation:**
  - `gofmt -s -w .`
  - `go vet ./...`
  - `go test -race ./...`
  - `go build ./...`
- [ ] **CI expectations:** Which workflows should pass

---

## Version Information

- Go: 1.25.0 (from `/go.mod`)
- tview: v0.42.0
- tcell: v2.13.6
- client-go: v0.35.0

---

## Final Instructions for the Agent

1. **Search the repository** for all relevant files before making changes
2. **Follow k9s patterns** for TUI components and keybindings
3. **Follow kubectl-ai patterns** for AI integration
4. **Run all validation commands** before submitting changes
5. **Keep changes minimal** and focused on the specific task
6. **Test thoroughly** including manual TUI verification
7. **Document changes** in commit messages and PR descriptions
