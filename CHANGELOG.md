# Changelog

All notable changes to k13d will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.8.5] - 2026-02-16

### Added
- **TUI: Sort Picker (`:sort`)**: Interactive column picker modal for sorting resources — dynamically lists current table columns with active sort indicator
- **TUI: Help Modal Sorting Section**: Help screen (`?`) now documents all sorting shortcuts (Shift+N/A/T/P/C/D) and `:sort` command
- **Web UI: Report Section Selection**: Choose which sections (Nodes, Namespaces, Workloads, Events, Security, FinOps, Metrics) to include when generating reports
- **Web UI: Rich Custom Resource Detail**: CRDs now display Overview/YAML/Events tabs with auto-generated status badge, metadata, printer columns, conditions table, and spec/status summary
- **Web UI: Historical Metrics Charts**: CPU, Memory, Pod Count, and Node Count charts with configurable time ranges (5m–24h) backed by SQLite storage
- **Web UI: Collect Now Button**: Trigger immediate metrics collection from the Metrics modal
- **AI Provider: Upstage/Solar**: Added Upstage (Solar) as a supported LLM provider with default endpoint

### Fixed
- **Web UI: Metrics Array Sync**: Fixed metricsHistory arrays going out of sync during live updates
- **Web UI: Chart Cleanup**: Charts are now properly destroyed and history reset when closing Metrics modal
- **Web UI: Context Switch Stale Data**: Switching cluster context now reloads namespaces and resource data

### Removed
- **Web UI: Healing View**: Removed auto-remediation rules UI

### Documentation
- Added 8 new screenshots to mkdocs (TUI autocomplete/help/LLM settings, Web UI applications/report preview/event timeline/network policy map/resource templates)
- Added Event Timeline, Network Policy Map, Resource Templates sections to web-ui docs
- Updated changelog, reports docs, and help references

## [0.8.4] - 2026-02-16

### Added
- **Helm Chart Release**: Helm chart (.tgz) now included as extra file in goreleaser releases
- **Validation Resource Counts**: Validation view now shows scanned resource counts (Pods, Services, Deployments, etc.)

### Fixed
- **Web UI: Report AI Check**: Report generation now shows error if AI is not connected instead of silently failing
- **Web UI: Ollama CSP Violation**: Ollama status check now uses backend proxy instead of direct browser fetch blocked by Content Security Policy
- **Web UI: Validation View**: Shows placeholder message when no namespace is selected instead of blank page
- **Web UI: Template Deployment**: Fixed `showNotification` undefined error and silent failure — now uses `showToast` with proper error/success feedback
- **Web UI: Template Category Filter**: Fixed case mismatch between backend categories and frontend dropdown values
- **Web UI: Audit Logs**: Fixed "Failed to load audit logs" caused by double DB initialization and non-JSON error responses
- **Web UI: Context Switch**: Context switch now properly reloads namespaces and data
- **Web UI: Log Download**: Log download filename now includes pod name and timestamp
- **Web UI: Namespace Indicator**: Shows available namespaces instead of empty list
- **Release: goreleaser**: Moved Helm chart output from dist/ to .helm-out/ to avoid "dist is not empty" error
- **Database**: Fixed double initialization — NewServer() no longer overwrites DB connection already set up by main

### Removed
- **Helm View**: Removed from sidebar (requires Helm releases installed)
- **GitOps View**: Removed from sidebar (requires ArgoCD or Flux installed)

### Documentation
- **MkDocs Expansion**: Detailed plugin, hotkey, alias, views, and TUI feature documentation

## [0.8.3] - 2026-02-14

### Added
- **Notification System**: Dispatch K8s cluster events (pod crash, OOM, node not ready, deploy fail, image pull fail) to Slack, Discord, Teams, Email (SMTP), and custom webhooks
- **NotificationManager**: Background event watcher with SHA256 dedup (5-min cooldown), provider-specific payloads, in-memory history (100 entries)
- **Email (SMTP) Provider**: Full SMTP/TLS support with config fields in UI (host, port, username, password, from, to)
- **Notification History**: View recent dispatch results in Settings → Notifications
- **Metrics Collector**: Auto-start on server boot — collects cluster/node/pod metrics every 1 min, stores in SQLite (7-day retention) with ring buffer cache
- **Historical Charts**: Now populated with real data from the metrics collector (previously empty due to uninitialized collector)

### Fixed
- **API Rate Limit**: Increased from 100 to 600 req/min — fixes 429 Too Many Requests on Validate and other dashboard views
- **Reports**: Metrics history now included in generated reports (was empty when collector was nil)

### Removed
- **Backups (Velero)**: Removed from sidebar, HTML, JS, and CSS
- **Light Mode**: Removed theme toggle — dark mode only
- **Diagnose Button**: Removed from assistant panel header

### Improved
- **Settings Tabs**: Horizontal scroll for overflow on smaller screens

## [0.8.2] - 2026-02-14

### Fixed
- **WebSocket Terminal**: Fixed connection failure caused by `responseWriter` not implementing `http.Hijacker`, preventing gorilla/websocket from upgrading connections
- **Server Timeouts**: Replaced `ReadTimeout`/`WriteTimeout` with `ReadHeaderTimeout` to prevent server-level timeouts from killing WebSocket and SSE streams
- **Overview Events**: Fixed `loadRecentEvents()` using non-existent `/api/resources/events` — corrected to `/api/k8s/events`
- **Overview Data**: Fixed `apiFetch` → `fetchWithAuth` (undefined function) in overview panel

## [0.8.1] - 2026-02-14

### Added
- **Gemini Model Validation**: Validate Gemini model names against known versioned prefixes (gemini-2.5-*, gemini-2.0-*, etc.)
- **Fetch Available Models**: New `/api/llm/available-models` endpoint with "Fetch Models" button for Gemini and Ollama providers
- **Model Autocomplete**: `<datalist>` suggestions populated from provider's model list
- **SSRF Protection**: Webhook URL validation - HTTPS-only, DNS resolution check, blocks private/loopback/link-local IPs
- **AuthzMiddleware**: Added authorization checks to context switch and notification endpoints
- **Backend Filters**: `warnings_only` for events timeline, `subject_kind` for RBAC visualization
- **Release Review Report**: Comprehensive `docs/RELEASE_REVIEW_REPORT.md` with findings from 6 review teams

### Fixed
- **Web UI Test Connection**: Fixed `response_time` vs `response_time_ms` field mismatch causing "undefinedms" display
- **Settings Save Error Handling**: `saveSettings()` now checks response status and shows error toasts instead of silently failing
- **XSS Prevention**: Replaced inline `onclick` handlers with `data-` attributes and event delegation for cluster context switching
- **Variable Shadowing**: Fixed terminal modal crash caused by Go variable shadowing in web handlers
- **Velero Type Assertion**: Fixed unsafe type assertion `first, _ := ns[0].(string)` with proper `ok` check
- **CSS Theme Variables**: Added missing `--bg-hover` and `--text-muted` to all 6 theme variants

### Changed
- Default Gemini model updated from `gemini-1.5-flash` to `gemini-2.5-flash`
- `showToast()` now supports `error` type (red background, 5s duration)

## [0.7.7] - 2026-02-14

### Added
- **Audit Logs Modal**: Audit logs now open as a modal overlay, preserving the center resource table
- **Reports Modal**: Cluster reports open as a modal overlay instead of replacing the resource table
- **Overview Page**: Dedicated Overview page with cluster health cards, quick actions, and recent events (AI panel hidden on Overview for clean look)
- **Trivy-Bundled Release**: goreleaser now produces `k13d-with-trivy` archives for linux/darwin with pre-bundled Trivy binary

### Changed
- AI Assistant panel auto-hides on Overview and restores on other views
- Audit Logs filter controls integrated into modal header

## [0.7.6] - 2026-02-14

### Added
- **Web UI: Metrics Dashboard**: Real-time cluster health cards with CPU/Memory bars, pod/deployment/node status, and recent events (backed by `/api/pulse`)
- **Web UI: Topology Tree View**: Hierarchical resource ownership visualization with collapsible tree nodes (backed by `/api/xray`)
- **Web UI: Applications View**: App-centric grouped view by `app.kubernetes.io/name` labels with health status badges
- **Web UI: Validate View**: Cross-resource validation with severity levels (critical/warning/info) and actionable suggestions
- **Web UI: Healing View**: Auto-remediation rules CRUD interface with event history tracking
- **Web UI: Helm Manager**: Full Helm release management with details, values, history, rollback, and uninstall
- **Web UI: Theme/Skin Selector**: 5 color themes in Settings - Tokyo Night (default), Production (red), Staging (yellow), Development (green), Light
- **Cross-view Navigation**: Related views linked together (Metrics↔Charts, Topology Graph↔Tree, Validate↔Reports)
- **Overview Quick Actions**: New buttons for Metrics, Topology Tree, Applications, Validate, and Helm

### Changed
- Sidebar reorganized: Visualization (Topology, Applications), Operations (Validate, Healing, Helm), Monitoring (Metrics, Audit Logs, Reports)
- XRay renamed to "Topology Tree" for consistency with existing Topology graph view
- Pulse renamed to "Metrics" for consistency with existing metrics charts

## [Unreleased]

### Added
- **Screen Ghosting Fix**: Eliminated TUI visual artifacts during modal transitions and AI streaming
  - Added 50ms draw throttle for AI streaming callbacks to prevent goroutine contention
  - Added periodic 500ms safety sync in `SetBeforeDrawFunc` as repaint safety net
  - Created `showModal()`/`closeModal()` helpers that trigger `screen.Sync()` on every modal transition

- **AI Chat History Preservation**: Previous Q&A sessions are now preserved when asking new questions
  - Chat history separated by visual dividers (`────────────────────────────`)
  - Scroll up to review previous conversations within the same session

- **Autocomplete Dropdown**: k9s-style command autocomplete with dropdown overlay
  - Shows dropdown when 2+ completions match typed text
  - Navigate with Up/Down arrows, select with Tab/Enter, dismiss with Esc
  - Single-match dimmed hint text preserved for quick completion

- **Configurable Resource Aliases** (`aliases.yaml`): Custom command shortcuts
  - Define short aliases for resource commands (e.g., `pp` → `pods`)
  - `:alias` command to view all configured aliases
  - Aliases merged with built-in commands in autocomplete

- **Per-Resource Sort Defaults** (`views.yaml`): Remember sort preferences per resource
  - Configure default sort column and direction per resource type
  - Applied automatically when navigating to a resource

- **LLM Model Switching** (`:model` command): Switch AI models from TUI
  - `:model` shows modal with all configured model profiles
  - `:model <name>` switches directly to a named profile
  - Active model marked with `*` in selector

- **Plugin System TUI Integration**: External plugins now accessible from TUI
  - Plugins loaded from `plugins.yaml` on startup
  - Plugin keyboard shortcuts active on matching resource scopes
  - `:plugins` command shows all available plugins
  - Supports foreground (with TUI suspend) and background execution

### Changed
- All 37+ `AddPage()`/`RemovePage()` calls migrated to `showModal()`/`closeModal()` helpers
- Command handling refactored to support prefix matching (`:model <name>`)

## [0.6.3] - 2026-02-08

### Added
- **Unified Command Classification**: New safety classification system for consistent tool approval
  - `pkg/ai/safety/classifier.go` - Unified command classifier using AST parsing
  - `pkg/ai/safety/enforcer.go` - Policy-based approval enforcement
  - Detects piped commands, chained commands, and file redirects
  - 27 comprehensive tests for classifier and enforcer

- **Tool Approval Policy Configuration**: Configurable approval behavior
  - `ToolApprovalPolicy` in config with AutoApproveReadOnly, BlockDangerous options
  - BlockedPatterns for regex-based command blocking
  - Configurable timeout for approval requests

- **Frontend Modular Structure**: Foundation for index.html refactoring
  - `scripts/build-frontend.go` - CSS/JS bundler for Go embed compatibility
  - `css/variables.css` - CSS custom properties (Tokyo Night theme)
  - `css/base.css` - Reset, typography, utility classes
  - `js/core/utils.js` - Common helper functions with IIFE pattern
  - `make frontend-build` target in Makefile

### Fixed
- **Security**: Piped/chained commands now properly require approval
  - Previously `kubectl get | xargs rm` was auto-approved as "read-only"
  - Now correctly identified as requiring approval
- **Classification**: Unknown commands now require approval (were wrongly auto-approved)
- **Interactive Commands**: `kubectl exec` now classified as "interactive" (was "write")

### Changed
- **classifyCommand()**: Now uses unified `safety.Classify()` internally
  - Provides consistent classification across TUI and Web UI
  - Backward compatible API

## [0.6.2] - 2026-02-08

### Added
- **MCP Server Tests**: Comprehensive test coverage for MCP server (19 new tests)
  - Server lifecycle tests (New, NewWithIO, RegisterTool)
  - Handler tests (Initialize, ListTools, CallTool, Ping)
  - Error handling tests (UnknownMethod, InvalidJSON, CallUnknownTool)
  - Tool definition tests (DefaultTools, schema validation)

### Fixed
- **Benchmark Task**: Fixed create-canary-deployment task.yaml missing required fields
  - Added name, description, category, tags, timeout, expect fields
  - Improved prompt formatting for readability
- **Benchmark Runner**: Improved error handling
  - Added config validation (nil check, task dir, LLM configs)
  - Better namespace management with detailed error output
  - Added --wait=false to namespace cleanup for faster execution
- **Code Style**: Fixed gofmt formatting in app_test.go
- **Flaky Test**: Fixed namespace switching test that failed when namespaces weren't loaded

## [0.6.0] - 2026-02-08

### Added
- **Branch Strategy**: Established dev as default branch, feature/* branches merge to dev, dev merges to main for releases
- **Enhanced MCP Integration**: k13d as MCP client spawns external MCP servers via JSON-RPC 2.0 stdio
- **Improved Documentation**: Comprehensive updates to all documentation files

### Changed
- **Repository Organization**: Updated repository URL to https://github.com/cloudbro-kube-ai/k13d

### Fixed
- **TUI Screen Ghosting**: Resolved screen ghosting issues during namespace/resource switching
- **Deprecated API Usage**: Improved error handling and fixed deprecated API usage

## [0.5.0] - 2026-02-05

### Added
- **TUI Testing Framework**: Added comprehensive TUI testing with golden files and screen capture
- **Feature Tests**: Added unit, E2E, and deadlock tests for TUI components

### Fixed
- **UI Stability**: Fixed TUI screen ghosting and improved concurrent access patterns

## [0.4.0] - 2026-02-01

### Added
- **Model Profiles DB Storage**: Store LLM model configurations in SQLite with CRUD operations, usage statistics, and active profile tracking
- **Prometheus Integration**: Full metrics dashboard with service discovery for Kubernetes API Server, Nodes, and cAdvisor
- **Trivy CVE Scanner**: Security vulnerability scanning with air-gapped environment support and auto-download capability
- **Storage Configuration**: Comprehensive storage configuration with `--storage-info` command for debugging
- **AI Benchmark Framework**: 125+ benchmark tasks with dry-run mode for cluster-free evaluation
- **TUI Improvements**: k13d ASCII logo, comprehensive tests, and highlight artifact fixes

### Changed
- **Project Structure**: Reorganized Dockerfiles to `deploy/docker/` directory
- **AI Settings**: Merged Model and LLM settings tabs into unified AI configuration tab
- **Chat History UI**: Improved icon alignment and visibility for edit/delete buttons

### Fixed
- **CI/CD**: Fixed Dockerfile path in GitHub Actions workflow
- **TUI Deadlocks**: Prevented highlight artifacts and deadlocks on startup
- **Session Memory**: Fixed benchmark session memory issues

## [0.3.0] - 2026-01-22

### Added
- **Workload Log Viewing**: Deployment, StatefulSet, DaemonSet, ReplicaSet now support multi-pod log viewing using actual label selectors
- **Enhanced i18n Support**: Added comprehensive translations for Chinese (中文) and Japanese (日本語) in Web UI
- **Label Selector Expressions**: Full support for matchExpressions (In, NotIn, Exists, DoesNotExist) in workload selectors
- **Prometheus Integration**: Kubernetes service discovery for metrics collection (API Server, Nodes, cAdvisor)

### Changed
- **Premium Glass Design**: Redesigned main dashboard with modern glass morphism UI
- **LLM Connection Test**: Fixed settings modal to test with form values before saving
- **Report Quality**: Improved AI-generated reports with CIS benchmark style formatting

### Fixed
- **K8s Token Auth**: Fixed session storage for cookie-based token authentication
- **Approval Messages**: Auto-remove approval status messages after 5 seconds

## [0.2.0] - 2026-01-20

### Added
- **OIDC/OAuth SSO**: Implement enterprise SSO authentication
- **Solar LLM Provider**: Added Upstage Solar API support
- **Korean Default Language**: Set Korean as default for web UI
- **Redesigned Login Page**: Modern login interface with glass design

### Fixed
- **CSRF Skip**: Skip CSRF check for login/logout endpoints
- **Server Errors**: Print errors to stderr for better visibility
- **Default Password**: Remove hardcoded password for secure random generation

### Security
- **Comprehensive Hardening**: Added security headers, rate limiting, and E2E tests

## [0.1.0] - 2026-01-15

### Added
- Initial release
- TUI Dashboard with k9s-style navigation
- Web UI with real-time streaming
- AI Assistant with tool execution
- Multi-provider LLM support (OpenAI, Ollama, Azure, Anthropic)
- Authentication (Local, LDAP, Token)
- Audit logging with SQLite
- i18n support (English, Korean)
- Kubernetes resource management (Pods, Deployments, Services, etc.)
