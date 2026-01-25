# Changelog

All notable changes to k13d will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
