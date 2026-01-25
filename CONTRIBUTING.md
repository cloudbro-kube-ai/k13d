# Contributing to k13d

First off, thank you for considering contributing to **k13d**! It's people like you who make k13d such a great tool.

## 🌟 Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

## 🚀 Getting Started

1.  **Fork the repository** on GitHub.
2.  **Clone your fork** locally:
    ```bash
    git clone https://github.com/your-username/k13d.git
    cd k13d
    ```
3.  **Set up the development environment**:
    - Ensure you have Go 1.24+ installed.
    - Run `go mod download`.
4.  **Create a branch** for your changes:
    ```bash
    git checkout -b feature/my-new-feature
    ```

## 🛠 Development Workflow

### Coding Standards
- **Go Style**: We strictly follow standard Go formatting. Run `gofmt -s -w .` before committing.
- **Linting**: We use `golangci-lint`. Please ensure your code passes all lints.
- **Documentation**: All exported functions, types, and constants must have descriptive comments.
- **Commit Messages**: We follow [Conventional Commits](https://www.conventionalcommits.org/).
  - `feat:` for new features
  - `fix:` for bug fixes
  - `docs:` for documentation changes
  - `refactor:` for code restructuring

### Testing
- **Unit Tests**: Always add unit tests for new logic in `pkg/`.
- **Integration Tests**: For complex features, consider adding integration tests that mock the Kubernetes API.
- **Run Tests**:
  ```bash
  go test ./...
  ```

## 📥 Pull Request Process

1.  **Update Documentation**: If you add a new feature or change an existing one, update the `README.md` or files in `docs/`.
2.  **Self-Review**: Review your own code for any obvious issues or optimizations.
3.  **Submit PR**: Fill out the PR template completely.
4.  **Wait for Review**: Maintainers will review your PR and may suggest changes.

## 🐞 Reporting Bugs

- Use the [GitHub Issue Tracker](https://github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/issues).
- Provide a clear summary and steps to reproduce.
- Include your OS, Go version, and Kubernetes version.

## 💡 Feature Requests

- Enhancement suggestions are tracked as GitHub Issues.
- Describe the "why" behind the feature and how it benefits the community.

---

*Happy Coding!*
