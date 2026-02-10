# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.6.x   | :white_check_mark: |
| 0.5.x   | :white_check_mark: |
| 0.4.x   | :x:                |
| < 0.4   | :x:                |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability within **k13d**, please follow these steps:

1. **Do not** open a public GitHub issue.
2. Report security vulnerabilities through GitHub's private vulnerability reporting at [Security Advisories](https://github.com/cloudbro-kube-ai/k13d/security/advisories/new).
3. Include information on how to reproduce the vulnerability.
4. We will acknowledge your report within 48 hours and work on a fix.

We follow a responsible disclosure policy and request that you give us time to address the issue before making it public.

## Security Features

k13d includes several security features:

- **RBAC Authorization**: Teleport-inspired deny-overrides-allow role system
- **JWT Tokens**: Short-lived HMAC-SHA256 tokens with automatic refresh
- **Command Safety Analysis**: AI-suggested commands are analyzed for safety before execution
- **Audit Logging**: All actions are logged to SQLite for accountability
- **No External Dependencies**: CGO-free SQLite, self-contained binary

## CI/CD Security

Our CI/CD pipeline includes multiple security layers:

| Layer | Tool | Purpose |
|-------|------|---------|
| Secrets Detection | [Gitleaks](https://github.com/gitleaks/gitleaks) | Prevents accidental credential commits |
| Dependency Scanning | [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) | Detects vulnerable Go dependencies |
| Static Analysis | [gosec](https://github.com/securego/gosec) | Finds security issues in Go code |
| Container Scanning | [Trivy](https://github.com/aquasecurity/trivy) | Scans Docker images for vulnerabilities |
| Dependency Updates | [Dependabot](https://docs.github.com/en/code-security/dependabot) | Automated security patches |

## Local Development Security

We recommend installing pre-commit hooks:

```bash
pip install pre-commit
pre-commit install
```

This enables local security checks including:
- Secret detection before commits
- Private key detection
- Go formatting and vetting
