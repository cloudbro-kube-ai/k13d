# Changelog

All notable changes to k13d will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2026-04-13

### Security
- **Go Standard Library**: Upgraded Go version from `1.25.8` to `1.25.9` to resolve standard library vulnerabilities:
  - [GO-2026-4869](https://pkg.go.dev/vuln/GO-2026-4869): Unbounded allocation in `archive/tar`
  - Multiple TLS-related vulnerabilities affecting standard HTTP/LDAP clients

### Changed
- **CI/CD**: Updated all GitHub Action workflows and Docker base image to Go `1.25.9`
- **Linting**: Added `golangci-lint` to pre-commit hooks for consistent local quality checks

## [1.0.0] - 2026-04-03

### Added
- **LiteLLM Gateway Mode**: Added gateway-based LLM access for broader provider compatibility and centralized model routing
- **AI Prompting Improvements**: Added kubectl few-shot guidance to improve tool selection quality and command generation
- **Web UI: Password Reset Flow**: Added a dedicated password reset modal and improved admin user-management flows
- **Web UI: Workload Details**: Surfaced workload security context information and enriched job/cronjob timing details

### Changed
- **Release Milestone**: Promoted k13d to `v1.0.0` as the stable baseline for the unified TUI, Web UI, and AI-assisted Kubernetes workflow
- **Web API Architecture**: Migrated the web surface to structured JSON errors and grouped route registration for maintainability
- **CLI Architecture**: Extracted shared CLI logic into `internal/cli` to reduce duplication between `k13d` and `kubectl-k13d`
- **Codebase Maintainability**: Continued modularization across `pkg/ui` and `pkg/web` to reduce large-file complexity ahead of the stable release line

### Fixed
- **TUI Stability**: Removed namespace-switch race paths and executed AI decisions without going through a shell
- **Web UI Stability**: Stabilized local-auth assistant settings flows, removed the legacy Ollama settings panel, and closed port-forward sessions on shutdown
- **Web API Robustness**: Added nil guards for operation handlers, consolidated raw fetch calls behind authenticated helpers, and hardened reports/FinOps analysis
- **Tool Execution Reliability**: Hardened kubectl tool path discovery and pinned the LiteLLM image reference to a stable tag

### Tests
- Added broader regression coverage for CLI initialization, route registration, web settings, Helm/config packages, and structured error handling

## [0.9.8] - 2026-03-15

### Added
- **Anthropic Claude Provider**: Native Anthropic Messages API support with streaming, tool calling, and model listing (Claude Opus 4, Sonnet 4, Haiku 4.5)
- **AI Panel: Stop Button**: Added stop button and functionality to cancel ongoing AI generation
- **AI Panel: Empty Input Feedback**: Visual error feedback with shake animation and red styling for empty AI input
- **AI Panel: Fullscreen Expansion**: Fullscreen expansion for AI panel with dynamic styling for input, context chips, and actions
- **MCP Debug UI**: Show MCP tool server name and type in debug UI
- **LLM Model Discovery**: Enable model fetch for all LLM providers (not just Gemini/Ollama)
- **Developer Scripts**: Added `run.sh` for local build+run and `log.sh` for tailing application logs

### Fixed
- **OpenAI reasoning_effort 400 Error**: Skip `reasoning_effort` parameter for models that don't support it (gpt-4, gpt-4o, etc.) — only sent for o-series and Solar Pro2
- **Gemini MCP Tools**: Strip unsupported schema keys for Gemini tool declarations (fixes 400 errors)
- **LLM Model Listing**: Use discovery mode for LLM model listing
- **Race Condition**: Resolve timeout middleware race condition (concurrent map write)
- **Race Condition**: Resolve race condition in k8s client and TUI test panic
- **Web UI: Race conditions**: Fix race conditions in LLM settings handlers and info disclosure
- **Web UI: RBAC**: Set admin role headers in no-auth mode for consistent RBAC
- **CI/CD**: Upgrade Go to 1.25.8 to resolve security vulnerabilities

### Changed
- **AI Panel Styling**: Moved expanded AI panel styling from JavaScript to CSS with i18n support for input hint
- **Build System**: Clean up Makefile to remove vendor directory usage
- **Code Modularity**: Split large files for maintainability
- **Pre-release Stabilization**: Refactoring and stabilization for release readiness

### Dependencies
- Bumped `tcell/v2` v2.13.6 → v2.13.8
- Bumped `modernc.org/sqlite` v1.43.0 → v1.46.1
- Bumped `golang.org/x/crypto` v0.45.0 → v0.48.0
- Bumped `helm.sh/helm/v3` v3.19.5 → v3.20.0
- Bumped `go-sql-driver/mysql` v1.8.1 → v1.9.3
- Bumped `lib/pq` v1.10.9 → v1.11.2
- Bumped Docker base images: golang 1.26.1-alpine, alpine 3.23
- Bumped GitHub Actions: checkout v6, setup-go v6, setup-python v6, upload-artifact v7, goreleaser-action v7, codeql-action v4, upload-pages-artifact v4

## [1.0.0-rc.1] - 2026-03-09

### Added
- **System Stability**: Added global panic handler in the TUI to gracefully restore terminal state on crash.
- **Code Modularity**: Massively refactored large monoline files (`pkg/ui/app.go` and `pkg/web/reports.go`) into smaller, modular domain-specific files (`app_layout.go`, `app_events.go`, `reports_security.go`, etc.) to improve long-term maintainability.
- **Health Check**: Validated and improved the `/api/health` system status endpoint in Web UI for better uptime monitoring readiness.

### Changed
- **Testing**: Improved test coverage across `pkg/ui` and `pkg/web` packages. Codebase is completely passing all unit and integration tests under `-short` mode.

## [0.9.7] - 2026-03-08

### Fixed
- **Web UI: Settings revert bug**: Fixed `updateEndpointPlaceholder()` overwriting saved provider/model/endpoint values when reopening Settings modal or selecting Ollama model from Quick Setup
- **Web UI: Model profile switch sync**: `switchModel()` now reloads LLM form fields after switching, keeping Settings form in sync with active profile
- **Web UI: Model deletion sync**: `deleteModel()` now reloads Settings form and uses toast notifications instead of browser alerts
- **Web UI: Consistent error feedback**: Replaced `alert()` with `showToast()` in `addModelProfile()` and `deleteModel()` for consistent UX
- **Web UI: Response validation**: Added `resp.ok` checks in `switchModel()`, `deleteModel()`, `addModelProfile()`, and `testLLMConnection()` to properly handle server errors
- **Web UI: Default values mismatch**: Unified fallback defaults between `loadSettings()` and `updateEndpointPlaceholder()` (ollama model, gemini model)
- **Backend: LLM settings validation**: Added required field validation for provider/model in `handleLLMSettings` to prevent config corruption from empty values
- **Backend: Embedded LLM protection**: `handleLLMSettings` now returns 403 when embedded LLM is active, preventing settings changes that would break the embedded server
- **Backend: Race condition in LLM response**: Response values in `handleLLMSettings` are now captured under mutex before unlock, preventing data races with concurrent model switches
- **Backend: Model deletion DB sync**: `handleModels` DELETE now calls `db.DeleteModelProfile()` to keep SQLite in sync with YAML config
- **Backend: Active model DB sync**: `handleActiveModel` PUT now calls `db.SetActiveModelProfile()` to update the `is_active` flag in SQLite
- **Backend: Last model deletion**: `RemoveModelProfile()` now clears `ActiveModel` when the last profile is deleted, instead of leaving a stale reference
- **Backend: Consistent logging**: Replaced `fmt.Printf("Warning: ...")` with `log.Warnf()` across settings handlers for proper structured logging

## [0.9.6] - 2026-03-01

### Added
- **Web UI: Application Detail Modal**: Clicking an app card now opens a detail modal showing status badge, version, component, pod count, and a resource table grouped by kind (Name/Namespace/Status)
- **Web UI: i18n Support**: Added `data-i18n` attributes to ~40 sidebar nav items, section headers, and view titles — changing language in Settings now updates the entire UI in real-time (English, Korean, Chinese, Japanese)
- **i18n: New Translation Keys**: Added translations for all nav items (Overview, Topology, Applications, RBAC Viewer, Net Policy Map, Event Timeline, Metrics, Audit Logs, Reports, NetworkPolicies, ServiceAccounts, Roles, RoleBindings, ClusterRoles, ClusterRoleBindings), section headers (RBAC, Visualization, Monitoring), and application view messages

### Tests
- Added `TestHandleApplications_MultiResourceTypes`: Verifies StatefulSet, DaemonSet, Ingress, Service grouping under same `app.kubernetes.io/name` label
- Added `TestHandleApplications_HealthStatus`: Verifies healthy/degraded/failing status calculation based on pod readiness
- Added `TestHandleApplications_NamespaceFilter`: Verifies `?namespace=X` query parameter correctly filters applications

## [0.9.5] - 2026-03-01

### Fixed
- **Web UI: Login form visibility**: Fixed login form input fields not visible due to CSS specificity conflict between inline `style.display` and class-based `.active` rules — unified all form toggling to use `classList`
- **Web UI: Auth mode form selection**: Correct login form (password vs token) now displayed based on `-auth-mode` flag via server-side HTML injection of `window.__AUTH_MODE__` and inline `style="display:block"`
- **Web UI: Login page layout**: Fixed K13D ASCII logo being pushed to the right by adding `overflow: hidden` to `.login-ascii-logo`
- **Web UI: JS syntax error**: Removed orphan code fragment (`") {"` with duplicate `fetchWithAuth` body) at line 723 that prevented all JavaScript from executing
- **Web UI: renderTableBody undefined**: Defined missing `generateRowHTML()` and `renderTableBody()` functions that were referenced but never implemented, causing `ReferenceError` on resource table rendering
- **Web UI: Infinite reload loop**: Fixed `loadClusterContexts()` being called at global scope before authentication, triggering 401 → `logout()` → `location.reload()` cycle. Moved into `showApp()` and added login page detection in `fetchWithAuth` to prevent auto-logout during login
- **AI Client: Lint error**: Fixed unchecked `w.Write` error return in `pkg/ai/client.go`
- **Web Server: Lint error**: Fixed unchecked `w.Write` error return in server-side HTML injection handler

## [0.8.6] - 2026-02-22

### Security
- **Auth: Session ID generation**: Proper error handling for `crypto/rand.Read` failures with secure fallback
- **Auth: Path traversal prevention**: Validate username extracted from URL path in user update/delete endpoints
- **Auth: CSRF/session cleanup**: Periodic goroutine cleans up expired CSRF tokens and sessions to prevent memory leaks
- **Auth: Token session cleanup**: Expired K8s token sessions now garbage-collected automatically

### Fixed
- **Web UI: Dark mode contrast**: Improved `--text-muted` color from `#565f89` to `#737aa2` for WCAG AA compliance (~4.5:1 ratio)
- **Web UI: Form accessibility**: Added `<label>` elements and `aria-label` attributes to login form inputs
- **Web UI: Debug logging**: Removed `console.log('[DEBUG]')` from production JavaScript
- **Web UI: Autocomplete**: Added proper `autocomplete` attributes to login form inputs
- **Docker: Reproducible builds**: Pinned Go version in `Dockerfile.bench` to `1.25.5` (was unpinned `1.25`)

### Tests
- Added tests for CSRF token cleanup, expired session cleanup, and path traversal prevention
- Added `defer am.StopCleanup()` to all auth tests to prevent goroutine leaks under race detector
- Added user creation validation tests (username length, characters, password length)

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
