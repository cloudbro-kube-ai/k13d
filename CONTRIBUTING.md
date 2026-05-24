# Contributing to k13d

First off, thank you for considering contributing to **k13d**! It's people like you who make k13d such a great tool.

## 🌟 Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

## 🚀 Getting Started

1.  **Fork the repository** on GitHub.
2.  **Clone your fork** locally:
    ```bash
    git clone https://github.com/cloudbro-kube-ai/k13d.git
    cd k13d
    ```
3.  **Set up the development environment**:
    - Ensure you have Go 1.25+ installed.
    - Run `go mod download`.
4.  **Create a branch from `dev`** for your changes:
    ```bash
    git checkout dev
    git pull origin dev
    git checkout -b feature/my-new-feature
    ```

## 🌿 Branch Strategy

We use a simple branch strategy based on `dev`:

```
main        ← Production releases only
  │
  └── dev   ← Development branch (default for PRs)
       │
       ├── feature/xxx   ← New features
       ├── fix/xxx       ← Bug fixes
       ├── docs/xxx      ← Documentation
       └── refactor/xxx  ← Code refactoring
```

### Branch Rules

| Branch | Purpose | Merge Target |
|--------|---------|--------------|
| `main` | Production releases | - |
| `dev` | Development integration | `main` (release) |
| `feature/*` | New features | `dev` |
| `fix/*` | Bug fixes | `dev` |
| `docs/*` | Documentation | `dev` |
| `refactor/*` | Code refactoring | `dev` |

### Workflow

1. **Always branch from `dev`**:
   ```bash
   git checkout dev
   git pull origin dev
   git checkout -b feature/my-feature
   ```

2. **Submit PR to `dev`** (not `main`)

3. **After PR merge**: Maintainers will merge `dev` → `main` for releases

---

## 🛠 Development Workflow

### Coding Standards
- **Go Style**: We strictly follow standard Go formatting. Run `gofmt -s -w .` before committing.
- **Pre-commit**: Install hooks with `pre-commit install`. Commits will run `go fmt ./...` automatically.
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
- **Headless TUI note**: if your local environment launches an interactive editor during TUI tests, run `EDITOR=true go test ./pkg/ui` for non-interactive validation of edit paths.

## 🤖 Issue-Driven Development

Maintainers can also drive a complete development loop from a GitHub Issue. This is the recommended workflow when you want a natural-language request to become one traceable branch, one PR, one Preview URL, and one final merge decision.

### Human Flow

1. Create a `Codex 개발 요청` issue.
2. Keep the request small enough for one PR.
3. Include goal, context, requested behavior, acceptance criteria, validation, and safety notes.
4. Do not include secrets, kubeconfigs, passwords, API keys, or private tokens.
5. Review the issue manually before adding `codex:auto`.
6. Add `codex:auto` only when the request is safe to execute.
7. Watch for `codex:running`, the accepted comment, the linked PR, CI status, and the Preview URL.
8. If the Preview needs changes, comment on the same issue with `k13d 수정해줘: ...`.
9. k13d marks the triggering comment with a 🚀 reaction and continues on the same issue branch and open PR.
10. Ask for another automated review with `k13d 코드리뷰 해줘`.
11. After human Preview verification, use the issue control panel merge checkbox or comment `k13d merge 해줘` when issue merge is enabled.

### Comment Commands

| Comment | Effect |
|---------|--------|
| `k13d 수정해줘: ...` | Continue development on the same issue branch and open PR |
| `k13d 계속 개발해줘: ...` | Same as above, useful after Preview review |
| `k13d 코드리뷰 해줘` | Run the configured review command and post a PR Review |
| `k13d merge 해줘` | Merge the linked PR when issue merge is enabled and GitHub protections pass |

Plain discussion comments are ignored on purpose. Use explicit `k13d` wording when you want the automation runner to act.

## 📥 Pull Request Process

1.  **Update Documentation**: If you add a new feature or change an existing one, update the `README.md` or files in `docs/`.
2.  **Self-Review**: Review your own code for any obvious issues or optimizations.
3.  **Submit PR**: Fill out the PR template completely.
4.  **Wait for Review**: Maintainers will review your PR and may suggest changes.

## 🐞 Reporting Bugs

- Use the [GitHub Issue Tracker](https://github.com/cloudbro-kube-ai/k13d/issues).
- Provide a clear summary and steps to reproduce.
- Include your OS, Go version, and Kubernetes version.

## 💡 Feature Requests

- Enhancement suggestions are tracked as GitHub Issues.
- Describe the "why" behind the feature and how it benefits the community.

---

*Happy Coding!*
