# Changelog

All notable changes to k13d are documented here.

## [0.8.5] - 2026-02-16

### Added
- **TUI: Sort Picker (`:sort`)**: Interactive column picker modal for sorting resources
- **TUI: Help Modal Sorting Section**: Help screen (`?`) now documents all sorting shortcuts (Shift+N/A/T/P/C/D)
- **Web UI: Report Section Selection**: Choose which sections to include when generating reports
- **Web UI: Rich Custom Resource Detail**: CRDs now display Overview/YAML/Events tabs
- **Web UI: Historical Metrics Charts**: Time-series charts with configurable time ranges (5m–24h)
- **Web UI: Collect Now Button**: Trigger immediate metrics collection
- **AI Provider: Upstage/Solar**: Added Upstage (Solar) as a supported LLM provider

### Fixed
- **Web UI: Metrics Array Sync**: Fixed metricsHistory arrays going out of sync
- **Web UI: Chart Cleanup**: Charts properly destroyed on Metrics modal close
- **Web UI: Context Switch Stale Data**: Context switch now reloads namespaces and data

### Removed
- **Web UI: Healing View**: Removed auto-remediation rules UI

---

## [0.8.4] - 2026-02-16

### Added
- **Helm Chart Release**: Helm chart (.tgz) now included as extra file in goreleaser releases
- **Validation Resource Counts**: Validation view now shows scanned resource counts

---

## [0.8.0] - 2026-02-14

### Added
- **Multi-Cluster**: Context switcher in top bar for switching between kubeconfig contexts
- **RBAC Viewer**: Visual subject→role relationship viewer with filtering by kind
- **Network Policy Map**: Ingress/egress rule visualization per policy
- **Event Timeline**: Cluster events grouped by time windows with stats
- **GitOps Integration**: ArgoCD Application and Flux Kustomization sync status
- **Resource Templates**: One-click deploy for common K8s patterns (Nginx, Redis, PostgreSQL, etc.)
- **Backups (Velero)**: Velero backup and schedule management with status tracking
- **Resource Diff**: Side-by-side YAML comparison (current vs last-applied-configuration)
- **Notifications**: Slack/Discord/Teams webhook alerts for cluster events
- **AI Auto-Troubleshoot**: One-click AI-powered cluster diagnosis via Diagnose button
- **kubectl Plugin**: Install as `kubectl k13d` via Krew or direct binary
- **Homebrew Formula**: `brew install k13d` support

### Changed
- Sidebar reorganized with new sections: RBAC Viewer, Net Policy Map, Event Timeline under Visualization; GitOps, Templates, Backups under Operations
- Settings modal gains Notifications tab

---

## [0.7.7] - 2026-02-14

### Added
- **Audit Logs Modal**: Audit logs open as modal overlay, preserving center resource table
- **Reports Modal**: Cluster reports open as modal overlay instead of replacing resource table
- **Overview Page**: Dedicated Overview with cluster health cards, quick actions, and recent events
- **Trivy-Bundled Release**: goreleaser produces `k13d-with-trivy` archives for linux/darwin

### Changed
- AI Assistant panel auto-hides on Overview and restores on other views
- Audit Logs filter controls integrated into modal header

---

## [0.7.6] - 2026-02-14

### Added
- **Web UI: Metrics Dashboard**: Real-time cluster health cards with CPU/Memory bars, pod/deployment/node status
- **Web UI: Topology Tree View**: Hierarchical resource ownership visualization with collapsible tree nodes
- **Web UI: Applications View**: App-centric grouped view by `app.kubernetes.io/name` labels
- **Web UI: Validate View**: Cross-resource validation with severity levels (critical/warning/info)
- **Web UI: Healing View**: Auto-remediation rules CRUD interface with event history tracking
- **Web UI: Helm Manager**: Full Helm release management with details, values, history, rollback, uninstall
- **Web UI: Theme/Skin Selector**: 5 color themes - Tokyo Night, Production, Staging, Development, Light
- **Cross-view Navigation**: Related views linked together (Metrics↔Charts, Topology Graph↔Tree, Validate↔Reports)
- **Dashboard Phase 1-5**: Backend APIs for pulse, xray, applications, validate, healing, helm, cost analysis

### Changed
- Sidebar reorganized: Visualization, Operations, Monitoring sections
- XRay renamed to "Topology Tree", Pulse renamed to "Metrics"

---

## [Unreleased]

### Added
- Documentation site with MkDocs Material
- GitHub Pages deployment
- **Screen Ghosting Fix**: 50ms draw throttle for AI streaming + 500ms periodic safety sync + modal transition sync
- **AI Chat History Preservation**: Previous Q&A sessions preserved with visual dividers within TUI session
- **Autocomplete Dropdown**: k9s-style dropdown overlay when 2+ command completions match
- **Configurable Resource Aliases** (`aliases.yaml`): Custom command shortcuts (e.g., `pp` → `pods`) with `:alias` viewer
- **Per-Resource Sort Defaults** (`views.yaml`): Remember sort column and direction per resource type
- **LLM Model Switching**: `:model` command to switch AI model profiles, `:model <name>` for direct switch
- **Plugin System TUI Integration**: Plugins from `plugins.yaml` now accessible via keyboard shortcuts and `:plugins` command

### Changed
- All modal open/close calls migrated to `showModal()`/`closeModal()` helpers with screen sync
- Command handling refactored for prefix matching (`:model <name>`)
- Improved error handling consistency

### Fixed
- TUI screen ghosting (visual artifacts) during modal transitions and AI streaming
- AI chat history lost when new response arrives
- Minor bug fixes

---

## [0.6.3] - 2024-01-15

### Added
- Tool Approval unification for consistent security
- PolicyEnforcer for centralized approval logic
- TUI audit logging for better compliance

### Changed
- Refactored index.html into modular CSS/JS files
- Improved frontend build process

### Fixed
- Docker Hub login conditional in CI workflow
- formatAge utility function consolidation

---

## [0.6.2] - 2024-01-10

### Added
- Tokyo Night theme for TUI
- Enhanced benchmark tooling
- MCP server mode (`--mcp` flag)

### Changed
- Improved AI response streaming
- Better error messages

### Fixed
- Test timeout issues (300s → 600s)
- TUI resize handling

---

## [0.6.1] - 2024-01-05

### Added
- Embedded LLM support (`--embedded-llm`)
- llama.cpp integration
- Offline operation capability

### Changed
- Reduced binary size
- Improved startup time

### Fixed
- Memory leaks in streaming responses
- Context cancellation handling

---

## [0.6.0] - 2024-01-01

### Added
- MCP (Model Context Protocol) integration
- External tool extensibility
- MCP client for connecting to MCP servers
- Sequential thinking MCP server support

### Changed
- Tool registry architecture
- AI agent state machine improvements

### Fixed
- Tool timeout issues
- JSON-RPC 2.0 compliance

---

## [0.5.0] - 2023-12-15

### Added
- Web UI with full feature parity to TUI
- SSE (Server-Sent Events) for AI streaming
- Authentication system
- Session management
- Audit logging

### Changed
- Unified API between TUI and Web
- Improved Kubernetes client performance

### Fixed
- Multiple concurrent session handling
- Resource leak on connection close

---

## [0.4.0] - 2023-12-01

### Added
- AI-powered analysis features
- Tool calling (kubectl, bash)
- Safety analyzer with AST parsing
- Command approval workflow

### Changed
- Agent architecture improvements
- Better prompt engineering

### Fixed
- Tool execution timeout issues
- Safety analysis false positives

---

## [0.3.0] - 2023-11-15

### Added
- Multi-provider LLM support
- OpenAI, Anthropic, Gemini, Ollama providers
- Provider abstraction layer
- Streaming response support

### Changed
- Configuration file format
- API key handling

### Fixed
- Provider reconnection logic
- Rate limit handling

---

## [0.2.0] - 2023-11-01

### Added
- AI Assistant panel in TUI
- Basic chat functionality
- Context-aware queries
- Resource context injection

### Changed
- TUI layout improvements
- Keyboard shortcut refinements

### Fixed
- Input handling edge cases
- Screen rendering issues

---

## [0.1.0] - 2023-10-15

### Added
- Initial release
- k9s-style TUI dashboard
- Kubernetes resource views
- Basic resource operations
- Vim-style navigation
- Multi-namespace support

---

## Version Format

k13d follows [Semantic Versioning](https://semver.org/):

- **MAJOR**: Incompatible API changes
- **MINOR**: New features (backwards compatible)
- **PATCH**: Bug fixes (backwards compatible)

## Links

- [GitHub Releases](https://github.com/cloudbro-kube-ai/k13d/releases)
- [Migration Guide](../getting-started/installation.md)
- [Contributing](https://github.com/cloudbro-kube-ai/k13d/blob/main/CONTRIBUTING.md)
