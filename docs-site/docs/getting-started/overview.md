# Overview

## What is k13d?

**k13d** (pronounced "k-thirteen-d") is a comprehensive Kubernetes management tool that combines:

- :desktop_computer: **k9s-style TUI** - Fast terminal dashboard with Vim keybindings
- :robot: **kubectl-ai Intelligence** - Agentic AI that *actually executes* kubectl commands
- :globe_with_meridians: **Modern Web UI** - Browser-based dashboard with real-time streaming

The name follows the numeronym pattern like k8s (k + 8 letters + s = kubernetes):

> **k**ube**a**i**d**ashboard = **k** + 13 letters + **d** = **k13d**

---

## Why k13d?

### The Problem

Managing Kubernetes clusters requires:

1. **Expert Knowledge** - Complex YAML, kubectl commands, resource relationships
2. **Context Switching** - Jumping between terminal, dashboards, and documentation
3. **Manual Investigation** - Debugging issues requires multiple commands and mental correlation

### The Solution

k13d solves these problems by:

| Challenge | k13d Solution |
|-----------|---------------|
| Complex kubectl commands | AI understands natural language and executes for you |
| Multiple tools needed | Single tool for TUI, Web, and AI assistance |
| Context switching | Integrated AI with full cluster context (YAML + Events + Logs) |
| Learning curve | Beginner mode with simple explanations |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              k13d                                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌────────────────┐    ┌────────────────┐    ┌────────────────┐        │
│  │   TUI Layer    │    │   Web Layer    │    │  AI/Agent Layer │        │
│  │  (tview/tcell) │    │  (net/http)    │    │  (OpenAI API)   │        │
│  └───────┬────────┘    └───────┬────────┘    └───────┬────────┘        │
│          │                     │                     │                  │
│          └─────────────────────┼─────────────────────┘                  │
│                                │                                         │
│                    ┌───────────▼───────────┐                            │
│                    │   Core Services       │                            │
│                    │  • K8s Client         │                            │
│                    │  • Config Manager     │                            │
│                    │  • Audit Logger       │                            │
│                    └───────────────────────┘                            │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Feature Comparison

| Feature | k13d | k9s | kubectl-ai | K8s Dashboard |
|---------|:----:|:---:|:----------:|:-------------:|
| TUI Interface | :white_check_mark: | :white_check_mark: | :x: | :x: |
| Web Interface | :white_check_mark: | :x: | :x: | :white_check_mark: |
| AI Assistant | :white_check_mark: | :x: | :white_check_mark: | :x: |
| Tool Execution | :white_check_mark: | :x: | :white_check_mark: | :x: |
| MCP Support | :white_check_mark: | :x: | :x: | :x: |
| Embedded LLM | :white_check_mark: | :x: | :x: | :x: |
| RBAC Authorization | :white_check_mark: | :x: | :x: | :warning: |
| Audit Logging | :white_check_mark: | :x: | :x: | :x: |

---

## Use Cases

### DevOps Engineers

- Quickly diagnose pod issues with AI assistance
- Execute complex kubectl operations via natural language
- Generate cluster health reports

### Platform Teams

- Provide self-service Kubernetes access with RBAC controls
- Audit all AI-assisted operations
- Integrate with existing LDAP/SSO

### Developers

- Understand Kubernetes resources without memorizing kubectl
- Get beginner-friendly explanations
- Access cluster from browser without terminal

---

## Getting Started

Ready to try k13d? Start with the [Installation Guide](installation.md).
