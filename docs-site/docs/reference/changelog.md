# Changelog

All notable changes to k13d are documented here.

## [Unreleased]

### Added
- Documentation site with MkDocs Material
- GitHub Pages deployment

### Changed
- Improved error handling consistency

### Fixed
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
