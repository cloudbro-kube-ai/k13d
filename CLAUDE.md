# CLAUDE.md
version: 1
default_agent: "@dev-agent"

> **Consulted files (agent must populate before committing):**
>
> **README.md files:**
> - `/README.md` - Main project README with features and getting started
> - `/docs/USER_GUIDE.md` - User guide with navigation and shortcuts
> - `/docs/CONFIGURATION_GUIDE.md` - Configuration guide
> - `/docs/config_examples.md` - Configuration examples
>
> **Documentation files:**
> - `/CONTRIBUTING.md` - Contribution guidelines
> - `/SUPPORT.md` - Support policy
> - `/SECURITY.md` - Security policy
> - `/CODE_OF_CONDUCT.md` - Code of conduct
> - `/docs/TUI_TESTING_STRATEGY.md` - TUI testing strategy for AI agents
>
> **Build/Config files:**
> - `/go.mod` - Go version (1.25.0) and dependencies
> - `/.goreleaser.yaml` - Release configuration
> - `/.github/workflows/release.yml` - CI/CD workflow

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

### Reference Projects
- **k9s** (`/k9s/`) - TUI patterns, keybindings, resource views, skin system
- **kubectl-ai** (`/kubectl-ai/`) - AI agent loop, tool definitions, LLM providers
- **headlamp** (`/headlamp/`) - Plugin architecture, AGENTS.md patterns
- **dashboard** (`/dashboard/`) - Kubernetes Dashboard patterns

### Skills (Pattern Reference Documents)
각 참조 프로젝트의 특장점을 정리한 Skill 문서들:

| Skill | 파일 | 주요 패턴 |
|-------|------|-----------|
| k9s Patterns | `skills/k9s-patterns.md` | MVC 아키텍처, Action 시스템, Plugin/HotKey, Skin, XDG 설정 |
| kubectl-ai Patterns | `skills/kubectl-ai-patterns.md` | Agent Loop, Tool System, LLM 추상화, MCP 통합, Safety Layers |
| Headlamp Patterns | `skills/headlamp-patterns.md` | Plugin Registry, Multi-Cluster, Response Cache, OIDC, i18n |
| K8s Dashboard Patterns | `skills/kubernetes-dashboard-patterns.md` | DataSelector, Multi-Module, Request-Scoped Client, Metrics |

**사용 가이드**: `skills/README.md` 참조

---

## Repository Map

```
kube-ai-dashboard-cli/
├── cmd/
│   ├── kube-ai-dashboard-cli/main.go   # Main entry point (TUI + Web modes)
│   └── eval/main.go                    # Evaluation tool
├── pkg/
│   ├── ui/                             # TUI components
│   │   ├── app.go                      # Main application
│   │   ├── app_*.go                    # App lifecycle & callbacks
│   │   ├── dashboard.go                # Resource dashboard view
│   │   ├── assistant.go                # AI assistant panel
│   │   ├── resource_viewer.go          # Resource detail viewer
│   │   ├── log_viewer.go               # Log streaming view
│   │   ├── audit_viewer.go             # Audit log viewer
│   │   ├── command_bar.go              # Command input bar
│   │   ├── header.go                   # Header component
│   │   ├── help.go                     # Help modal
│   │   ├── settings.go                 # Settings modal
│   │   ├── pulse.go                    # Pulse/health view
│   │   └── resources/                  # Resource-specific views
│   │       ├── pods.go
│   │       ├── deployments.go
│   │       ├── services.go
│   │       ├── nodes.go
│   │       ├── namespaces.go
│   │       ├── contexts.go
│   │       ├── configmaps.go
│   │       ├── secrets.go
│   │       ├── ingresses.go
│   │       ├── statefulsets.go
│   │       ├── storage.go
│   │       ├── events.go
│   │       ├── rbac.go
│   │       ├── serviceaccounts.go
│   │       └── types.go
│   ├── web/                            # Web UI server
│   │   ├── server.go                   # HTTP server & API handlers
│   │   ├── auth.go                     # Authentication system
│   │   ├── auth_test.go                # Auth tests
│   │   ├── reports.go                  # Report generation
│   │   ├── reports_test.go             # Report tests
│   │   └── static/                     # Embedded static files
│   │       └── index.html              # Web UI frontend
│   ├── ai/                             # AI client
│   │   ├── client.go                   # OpenAI-compatible HTTP client
│   │   ├── client_test.go              # AI client tests
│   │   └── reporter.go                 # AI reporter
│   ├── k8s/                            # Kubernetes client wrapper
│   │   ├── client.go
│   │   └── client_test.go
│   ├── config/                         # Configuration management
│   │   ├── config.go
│   │   └── config_test.go
│   ├── db/                             # SQLite database layer
│   │   ├── db.go
│   │   ├── audit.go
│   │   └── audit_test.go
│   ├── i18n/                           # Internationalization
│   │   ├── i18n.go
│   │   └── i18n_test.go
│   ├── log/                            # Logging utilities
│   │   └── log.go
│   ├── mcp/                            # MCP configuration
│   │   └── default_config.yaml
│   └── eval/                           # Evaluation framework
│       ├── eval.go
│       └── tasks.yaml
├── docs/
│   ├── USER_GUIDE.md
│   ├── CONFIGURATION_GUIDE.md
│   └── config_examples.md
├── .github/workflows/
│   └── release.yml                     # Release workflow
├── .goreleaser.yaml                    # GoReleaser config
├── go.mod
├── go.sum
├── README.md
├── CONTRIBUTING.md
├── SUPPORT.md
├── SECURITY.md
├── CODE_OF_CONDUCT.md
└── LICENSE
```

---

## Primary Entry Points (Exact Commands)

### Build Commands
```bash
# Build main binary
go build -o k13d ./cmd/kube-ai-dashboard-cli/main.go

# Build with version info
go build -ldflags "-X main.version=$(git describe --tags)" -o k13d ./cmd/kube-ai-dashboard-cli/main.go

# Build evaluation tool
go build -o k13d-eval ./cmd/eval/main.go
```

### Run Commands
```bash
# Run TUI mode (default)
./k13d

# Run Web UI mode
./k13d -web -port 8080

# Run with specific kubeconfig
./k13d --kubeconfig ~/.kube/config

# Run in debug mode
./k13d --debug
```

### Test Commands
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test -v ./pkg/ui/...
go test -v ./pkg/web/...
go test -v ./pkg/ai/...
go test -v ./pkg/k8s/...
go test -v ./pkg/config/...
go test -v ./pkg/db/...
go test -v ./pkg/i18n/...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Lint Commands
```bash
# Run golangci-lint
golangci-lint run

# Run with auto-fix
golangci-lint run --fix

# Run gofmt
gofmt -s -w .

# Run go vet
go vet ./...
```

### Format Commands
```bash
# Format all Go files
gofmt -s -w .

# Check formatting
gofmt -d .
```

---

## Allowed Commands and CI Interactions

### Permitted to Suggest/Run Locally
- All Go commands: `go build`, `go test`, `go fmt`, `go vet`, `go mod tidy`
- Linting: `golangci-lint run`
- Local execution: `./k13d`

### Require Human Approval
- Pushing container images
- Publishing releases
- Modifying `.github/workflows/*`
- Changing `Dockerfile` (if exists)
- Modifying release configurations

### Reporting CI Results
GitHub Actions workflows in `.github/workflows/` - summarize failing steps, include logs, recommend fixes with local reproduction commands.

---

## Change Rules and Safety Constraints

### Manual-Review-Only Files
- `.github/workflows/*` - CI workflows
- `.goreleaser.yaml` - Release configuration
- `SECURITY.md`, `SECURITY_CONTACTS` - Security policy files
- `LICENSE` - License file
- `CODE_OF_CONDUCT.md` - Code of conduct

### Pre-Change Checks
1. Run `gofmt -s -w .` to format code
2. Run `go vet ./...` to check for issues
3. Run `golangci-lint run` for linting
4. Run `go test ./...` to ensure tests pass
5. Run `go build ./...` to verify compilation

### Dependency Updates
- Run full test suite: `go test ./...`
- Do not bump major versions without approval
- Verify compatibility with kubectl-ai and gollm dependencies

---

## Best Practices and Coding Guidelines

### Reduce Solution Size
- Make minimal, surgical changes - modify as few lines as possible
- Prefer focused, single-purpose changes over large refactors
- Break down complex changes into smaller, reviewable increments

### TUI Development Guidelines (tview/tcell)
- Follow k9s patterns for keybindings and navigation
- Use `tview.Application.QueueUpdateDraw()` for thread-safe UI updates
- Implement `Draw()` method efficiently to avoid flickering
- Handle resize events gracefully
- Use `tcell.EventKey` for keyboard handling consistently

### AI Integration Guidelines
- Follow kubectl-ai patterns for tool definitions
- Ensure AI-proposed modifications require explicit user approval
- Log all AI tool invocations to audit database
- Handle LLM provider errors gracefully with user feedback

### Kubernetes Client Guidelines
- Use informers for efficient resource watching
- Implement proper error handling for API failures
- Support multiple contexts and namespaces
- Cache resources appropriately to reduce API calls

### Testing Best Practices
- Avoid using mocks if possible - prefer testing with real implementations
- Use integration tests for complex features
- Write tests that validate actual behavior, not implementation details
- Mock external dependencies (K8s API, LLM providers) when necessary

### Internationalization (i18n)
- Use `/pkg/i18n/` for all user-facing strings
- Support: English, 한국어, 简体中文, 日本語
- Test UI with different languages for layout issues

---

## Key Features Reference

### Dashboard Navigation (k9s Parity)
| Key | Action |
|-----|--------|
| `j/k` | Move selection up/down |
| `Left/Right/Tab` | Switch focus between panels |
| `Ctrl+H/Ctrl+L` | Resize panels |
| `:` | Command mode (e.g., `:pods`, `:svc`, `:deploy`) |
| `/` | Filter current table |
| `ESC` | Close modal/return to main view |

### Resource Actions
| Key | Action |
|-----|--------|
| `y` | View YAML manifest |
| `l` | Stream logs (Pods) |
| `d` | Native describe |
| `L` | AI Analyze (send to assistant) |
| `h` | Explain This (pedagogical) |
| `s` | Scale replicas |
| `r` | Rollout restart |
| `Shift+F` | Port forwarding |
| `Ctrl+D` | Delete (with confirmation) |

### AI Assistant Features
- **Context Awareness**: Receives YAML, events, and logs for analysis
- **Tool Use**: kubectl, bash, MCP integration
- **Safety**: All modifications require explicit user approval
- **Beginner Mode**: Simple explanations for complex resources

---

## Examples and Templates

### Example 1: Adding a New Resource View

```go
// pkg/ui/resources/newresource.go
package resources

import (
    "context"
    "github.com/rivo/tview"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NewResourceView struct {
    *BaseResourceView
}

func NewNewResourceView(app *tview.Application, k8sClient K8sClient) *NewResourceView {
    v := &NewResourceView{
        BaseResourceView: NewBaseResourceView(app, k8sClient, "NewResource"),
    }
    v.SetColumns([]string{"NAME", "NAMESPACE", "AGE"})
    return v
}

func (v *NewResourceView) Refresh(ctx context.Context) error {
    // Fetch resources from K8s API
    // Update table data
    return nil
}
```

**Commands to validate:**
1. `gofmt -s -w .`
2. `go vet ./...`
3. `go test -v ./pkg/ui/resources/...`
4. `go build ./...`

### Example 2: Adding AI Tool Integration

```go
// pkg/ai/tools.go - Following kubectl-ai patterns
type NewTool struct {
    Name        string
    Description string
    Parameters  map[string]interface{}
}

func (t *NewTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
    // Implement tool logic
    // Log to audit database
    return result, nil
}
```

**Commands to validate:**
1. `go test -v ./pkg/ai/...`
2. Verify audit logging works

### Example 3: Bug Fix in TUI Component

**Files to change:** `pkg/ui/dashboard.go`
**Rationale:** Fix navigation issue in resource table
**Commands to validate:**
1. `gofmt -s -w .`
2. `go vet ./...`
3. `go test -v ./pkg/ui/...`
4. Manual testing: `./k13d` and verify navigation

---

## PR Description & Commit Message Format

### Commit Message Format (Conventional Commits)
```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat:` new features
- `fix:` bug fixes
- `docs:` documentation changes
- `refactor:` code restructuring
- `test:` adding tests
- `chore:` maintenance tasks

**Examples:**
- `feat(ui): add StatefulSet resource view`
- `fix(k8s): handle context switch error gracefully`
- `docs(readme): update installation instructions`
- `refactor(ai): simplify tool registration`

### PR Description Template
```markdown
## Summary
Brief description of what the change does.

## Related Issue
Fixes #ISSUE_NUMBER (if applicable)

## Changes
- Added/Updated/Fixed component X
- Modified behavior Y

## Testing
- [ ] Unit tests added/updated
- [ ] Manual testing completed
- [ ] All existing tests pass

## Screenshots (for UI changes)
Include screenshots showing the visual changes.
```

---

## Agent Output Checklist

Before creating a patch/PR, ensure:

- [ ] **Summary:** one-line intent and short rationale
- [ ] **Sources:** list consulted README/docs file paths
- [ ] **Files changed:** explicit file list with rationale for each
- [ ] **Diff/patch:** minimal unified diff showing only necessary changes
- [ ] **Tests:**
  - List tests added/updated
  - Exact commands to run them
  - Test results showing pass status
- [ ] **Local validation:**
  - Exact commands to reproduce build/test results
  - Output showing successful execution
  - Manual verification of TUI changes
- [ ] **CI expectations:**
  - Which workflows should pass
  - Expected test coverage

---

## Appendix

### Key Documentation Files
1. `/README.md` - Main project overview
2. `/docs/USER_GUIDE.md` - User navigation and shortcuts
3. `/docs/CONFIGURATION_GUIDE.md` - Configuration options
4. `/CONTRIBUTING.md` - Contribution guidelines

### Reference Projects for Patterns
1. **k9s** - TUI architecture, resource views, keybindings, skins
2. **kubectl-ai** - AI agent loop, tool definitions, LLM provider integration
3. **headlamp** - Plugin system, AGENTS.md structure

### Version Information
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
