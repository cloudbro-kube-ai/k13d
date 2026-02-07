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
