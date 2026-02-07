# Changelog

All notable changes to k13d will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
