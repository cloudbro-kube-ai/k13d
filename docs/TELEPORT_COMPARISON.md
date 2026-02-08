# k13d vs Teleport: Feature Comparison

This document provides a detailed feature-by-feature comparison between **k13d** and **Gravitational Teleport**, two platforms that address Kubernetes infrastructure management from different perspectives.

## Executive Summary

| Aspect | k13d | Teleport |
|--------|------|----------|
| **Focus** | Kubernetes AI Explorer & Dashboard | Infrastructure Identity & Access Platform |
| **Primary Goal** | Day-to-day K8s operations with AI assistance | Zero-trust access control across all infrastructure |
| **Target User** | DevOps engineers, SREs, developers | Security teams, platform engineers, compliance |
| **Interface** | TUI (k9s-style) + Web UI | Web UI + CLI (`tsh`/`tctl`) + Desktop App |
| **AI Integration** | Core feature (agentic AI assistant) | Emerging (session summaries, MCP governance) |
| **K8s Resource Mgmt** | Deep (30+ resource types, drill-down) | Proxy-based (kubectl access control) |
| **Authentication** | Local, Token, LDAP | SSO, Certificates, MFA, Device Trust |
| **License** | MIT (Open Source) | AGPL (Community) / Commercial (Enterprise) |

---

## 1. Architecture & Philosophy

### k13d: Kubernetes-First AI Dashboard

k13d is designed as a **direct Kubernetes management tool** that combines the operational efficiency of k9s with AI-powered intelligence. It connects directly to the Kubernetes API server and provides a rich interactive interface for managing cluster resources.

```
User ─→ k13d (TUI/Web) ─→ Kubernetes API Server
              │
              └─→ AI Assistant (LLM) ─→ Tool Execution
```

**Key Design Principles:**
- Direct cluster access with minimal setup
- AI-first approach to Kubernetes troubleshooting
- Dual interface (TUI + Web) with feature parity
- Single binary, zero external dependencies
- Offline-capable with embedded LLM

### Teleport: Zero-Trust Infrastructure Gateway

Teleport acts as an **identity-aware access proxy** between users and infrastructure. It replaces VPNs, bastion hosts, and shared credentials with certificate-based authentication and comprehensive audit logging.

```
User ─→ tsh/Web UI ─→ Teleport Proxy ─→ Teleport Auth ─→ Infrastructure
                                                              ├── SSH Servers
                                                              ├── Kubernetes
                                                              ├── Databases
                                                              ├── Web Apps
                                                              ├── Windows RDP
                                                              └── MCP Servers
```

**Key Design Principles:**
- Zero-trust: verify everything, trust nothing
- Short-lived certificates replace long-lived secrets
- Unified access layer across all infrastructure types
- Complete session recording and audit
- Compliance-first design (FedRAMP, SOC 2, HIPAA)

---

## 2. Kubernetes Management

| Feature | k13d | Teleport |
|---------|------|----------|
| **Resource Types** | 30+ (Pods, Deployments, Services, CRDs, etc.) | N/A (proxies kubectl, no resource UI) |
| **Resource Viewing** | Interactive table with sort/filter/drill-down | Via kubectl through proxy |
| **Resource Actions** | YAML, Describe, Edit, Delete, Scale, Restart | Via kubectl through proxy |
| **Log Streaming** | Built-in with ANSI color | Via kubectl through proxy |
| **Shell Access** | Built-in `s` key (pod exec) | Via kubectl through proxy |
| **Port Forwarding** | Built-in `Shift+F` | Via kubectl through proxy |
| **Namespace Navigation** | Quick-switch (0-9 keys), command bar | N/A |
| **Context Switching** | Built-in `:ctx` command | `tsh kube login <cluster>` |
| **Multi-Cluster** | One cluster at a time (switchable) | Multi-cluster with unified access |
| **Cluster Discovery** | Manual kubeconfig | Auto-discovery (EKS, GKE, AKS) |
| **K8s RBAC** | Inherits kubeconfig permissions | Fine-grained proxy RBAC overlay |
| **Helm Integration** | Built-in (list, install, upgrade, uninstall) | N/A |
| **Metrics Display** | CPU/Memory charts (metrics-server) | N/A |
| **Security Scanning** | Trivy CVE, kube-bench compliance | N/A |

### Summary

k13d provides **deep Kubernetes resource management** with interactive navigation, AI-powered analysis, and operational tools. Teleport provides **secure access to Kubernetes** through proxy-based authentication but does not manage resources directly — it delegates to kubectl.

---

## 3. AI & Intelligence

| Feature | k13d | Teleport |
|---------|------|----------|
| **AI Assistant** | Core feature — agentic AI with tool use | N/A |
| **Natural Language Queries** | "Why is this pod failing?" | N/A |
| **AI Tool Execution** | kubectl, bash, MCP tools with approval | N/A |
| **Command Safety Analysis** | Read-only / Write / Dangerous classification | N/A |
| **AI Decision Dialog** | Numbered command approval (1-9, A=all) | N/A |
| **Beginner Mode** | Simplified AI explanations | N/A |
| **LLM Provider Support** | 8+ (OpenAI, Anthropic, Ollama, Bedrock, etc.) | N/A |
| **Model Switching** | Live switching via `:model` command | N/A |
| **Embedded LLM** | Qwen2.5-0.5B (offline, no API key) | N/A |
| **Streaming Responses** | SSE streaming with real-time display | N/A |
| **Chat History** | Preserved within session | N/A |
| **Session Summaries** | N/A | AI-powered session recording summaries (Enterprise) |
| **MCP Governance** | MCP client (tool extensibility) | MCP access control & governance (emerging) |
| **AI Agent Security** | Tool approval per command | Comprehensive agentic identity framework |
| **AI Benchmarking** | 125+ task benchmark suite | N/A |

### Summary

k13d treats AI as a **first-class feature** for Kubernetes operations — users can ask questions, get analysis, and execute commands through an AI assistant. Teleport is developing AI governance capabilities focused on **securing AI agents** (MCP access control, agentic identity) rather than providing an AI assistant directly.

---

## 4. User Interface

### 4.1 Terminal Interface

| Feature | k13d | Teleport |
|---------|------|----------|
| **TUI Dashboard** | Full k9s-style interactive TUI | N/A |
| **Vim Navigation** | j/k, g/G, Ctrl+U/F | N/A |
| **Command Bar** | `:pods`, `:deploy`, `:ns` | N/A |
| **Autocomplete** | Dropdown with descriptions | N/A |
| **Filter/Search** | Substring and regex (`/pattern/`) | N/A |
| **Multi-Select** | Space bar selection | N/A |
| **Column Sorting** | Click/hotkey with visual indicators | N/A |
| **Themes/Skins** | Customizable color schemes | N/A |
| **Plugin System** | External commands via hotkeys | N/A |
| **CLI Tool** | `k13d` (single binary) | `tsh` (user), `tctl` (admin), `tbot` (machine) |
| **SSH Client** | N/A | `tsh ssh user@host` |
| **Database Client** | N/A | `tsh db connect <database>` |
| **App Access** | N/A | `tsh apps login <app>` |

### 4.2 Web Interface

| Feature | k13d | Teleport |
|---------|------|----------|
| **Web Dashboard** | Resource tables, metrics, charts | Cluster management, session viewer |
| **AI Chat Panel** | SSE streaming with approval flow | N/A |
| **Resource YAML View** | Syntax-highlighted YAML editor | N/A |
| **Log Viewer** | Real-time streaming with ANSI colors | N/A |
| **Terminal (Web)** | xterm.js pod exec | xterm.js SSH sessions |
| **Session Recording** | N/A | Full playback of SSH/K8s/DB sessions |
| **Session Sharing** | N/A | Live session observation & moderation |
| **Settings Panel** | LLM, language, refresh, auth config | Cluster config, roles, SSO, users |
| **Access Requests** | N/A | Create/approve access requests |
| **User Management** | Admin dashboard (CRUD) | Full user/role management |
| **Auto-Refresh** | Configurable interval (10s-5m) | Real-time via WebSocket |

### 4.3 Desktop Application

| Feature | k13d | Teleport |
|---------|------|----------|
| **Desktop App** | N/A | Teleport Connect (Electron) |
| **VNet** | N/A | VPN-like experience (macOS/Windows) |
| **Resource Browser** | N/A | Graphical resource discovery |

---

## 5. Authentication & Security

| Feature | k13d | Teleport |
|---------|------|----------|
| **Auth Modes** | Local, Token, LDAP | SSO (OIDC, SAML, GitHub, Okta, Entra ID) |
| **Certificate-Based Auth** | N/A | Core — short-lived X.509/SSH certificates |
| **MFA** | N/A | Per-session MFA, hardware keys (YubiKey) |
| **Passwordless** | N/A | Biometrics, passkeys |
| **Device Trust** | N/A | TPM/secure enclave-based device identity |
| **SSO Integration** | LDAP/AD | GitHub, Google, Okta, Entra ID, SAML, OIDC |
| **Session Management** | Cookie-based (HttpOnly) | Certificate-based with TTL |
| **RBAC** | 3 roles (admin, user, viewer) | Granular roles with deny rules |
| **ABAC** | N/A | Label-based attribute access control |
| **Access Requests** | N/A | JIT access with dual authorization |
| **Access Lists** | N/A | Recurring access grants with reviews |
| **Identity Locks** | N/A | Immediate access revocation |
| **User Provisioning** | Manual or LDAP | SCIM (Okta, Entra ID, SailPoint) |
| **Password Hashing** | SHA256/bcrypt | Certificate-based (no passwords) |
| **CSRF Protection** | Yes | Yes |
| **Rate Limiting** | Login attempts | Comprehensive |
| **Security Headers** | X-Frame-Options, CSP | Full security header suite |
| **TLS/HTTPS** | Optional | Required (mTLS everywhere) |

### Summary

Teleport's authentication is **enterprise-grade** with zero-trust principles, short-lived certificates, and multi-factor authentication. k13d provides **practical authentication** suitable for team deployments with local, token, and LDAP modes.

---

## 6. Audit & Compliance

| Feature | k13d | Teleport |
|---------|------|----------|
| **Audit Logging** | SQLite-based (all actions, AI tools) | Structured events across all protocols |
| **Session Recording** | N/A | Full recording (SSH, K8s, DB, Desktop) |
| **Session Playback** | N/A | Web-based session replay |
| **Live Session View** | N/A | Watch active sessions in real-time |
| **Audit Viewer** | `:audit` command (TUI), `/api/audit` (Web) | Web UI audit log with search |
| **Audit Export** | CSV/JSON download | SIEM integration (Splunk, Elastic, Datadog) |
| **AI Action Audit** | Every AI tool invocation logged | N/A |
| **Sensitive Data Masking** | API keys, passwords masked | Certificate-based (no secrets to mask) |
| **Retention Policy** | Configurable retention days | External storage (S3, DynamoDB) |
| **Compliance Standards** | N/A | FedRAMP, SOC 2, HIPAA, PCI DSS, ISO 27001 |
| **FIPS Compliance** | N/A | FIPS-compliant binaries available |
| **Security Audits** | N/A | Published third-party audits |

### Summary

Teleport provides **enterprise compliance capabilities** with comprehensive session recording, SIEM integration, and regulatory compliance certifications. k13d focuses on **operational audit** logging AI tool invocations and user actions for transparency.

---

## 7. Infrastructure Scope

| Resource Type | k13d | Teleport |
|---------------|------|----------|
| **Kubernetes Clusters** | Direct management (single cluster) | Multi-cluster access proxy |
| **SSH Servers** | N/A | Full access with recording |
| **Databases** | N/A | PostgreSQL, MySQL, MongoDB, Oracle, etc. |
| **Web Applications** | N/A | JWT-based application proxy |
| **Windows Desktops** | N/A | RDP via smart card auth |
| **Cloud APIs** | N/A | AWS, Azure, GCP console access |
| **MCP Servers** | Client (tool extensibility) | Access control & governance |

### Summary

k13d is **Kubernetes-specialized** — it provides the deepest Kubernetes management experience. Teleport is **infrastructure-wide** — it provides unified access control across servers, databases, apps, desktops, and cloud services.

---

## 8. Deployment & Operations

| Feature | k13d | Teleport |
|---------|------|----------|
| **Single Binary** | Yes (Go, ~30MB) | Yes (Go, ~100MB+) |
| **Docker** | Pre-built images, Docker Compose | Docker images available |
| **Kubernetes** | Deployment manifests, single-pod | Helm charts, operator |
| **Air-Gapped** | Supported (Ollama + embedded LLM) | Supported (self-hosted) |
| **Cloud Hosted** | N/A | Teleport Enterprise Cloud (SaaS) |
| **External DB Required** | No (SQLite embedded) | Yes (etcd, DynamoDB, or PostgreSQL) |
| **HA Setup** | N/A (single instance) | Multi-region HA with failover |
| **Auto-Discovery** | N/A | EC2, RDS, EKS, GKE, AKS auto-enrollment |
| **Agent Architecture** | Standalone | Auth + Proxy + Agents (distributed) |
| **Resource Requirements** | Minimal (runs on laptop) | Moderate to high (production cluster) |

---

## 9. Configuration & Extensibility

| Feature | k13d | Teleport |
|---------|------|----------|
| **Config Format** | YAML files (XDG-compliant) | YAML files + dynamic resources |
| **Resource Aliases** | Custom shortcuts (aliases.yaml) | N/A |
| **View Defaults** | Per-resource sort config (views.yaml) | N/A |
| **Custom Hotkeys** | External commands (hotkeys.yaml) | N/A |
| **Plugin System** | Command-based plugins (plugins.yaml) | N/A |
| **Theme System** | Color customization (skins/) | N/A |
| **MCP Integration** | Client + Server modes | Server governance (emerging) |
| **i18n** | 4 languages (en, ko, zh, ja) | English primarily |
| **API Extensibility** | REST API (50+ endpoints) | gRPC API + REST |
| **Terraform Provider** | N/A | Official Terraform provider |
| **Slack/Teams Bots** | N/A | Access request bots |

---

## 10. MCP (Model Context Protocol)

| Feature | k13d | Teleport |
|---------|------|----------|
| **MCP Role** | Client (consumes tools) + Server (exposes tools) | Governance layer (secures MCP access) |
| **Tool Discovery** | `tools/list` via JSON-RPC 2.0 | MCP catalog service |
| **Tool Execution** | AI invokes tools with user approval | Agents invoke tools with RBAC |
| **Tool Approval** | Per-command interactive approval | Policy-based (RBAC/ABAC) |
| **Rate Limiting** | N/A | Token/cost-based rate limits |
| **Budget Controls** | N/A | Per-agent spending limits |
| **Prompt Tracking** | N/A | Prompt/response audit logging |
| **Agent Identity** | N/A | Digital twin identity for agents |
| **Server Mode** | k13d exposes K8s tools to AI clients | N/A (governance only) |

### Summary

k13d uses MCP to **extend AI capabilities** with external tools. Teleport uses MCP to **govern AI agent access** to infrastructure tools — they address complementary concerns.

---

## 11. Complementary Use Cases

k13d and Teleport are not direct competitors — they serve different roles and can work together:

### k13d Strengths (Teleport Cannot Replace)
1. **Interactive K8s Dashboard** — Real-time resource management with TUI/Web
2. **AI-Powered Troubleshooting** — Natural language Kubernetes assistance
3. **Operational Efficiency** — k9s-style keybindings for rapid navigation
4. **Helm Management** — Integrated Helm release lifecycle
5. **Metrics Visualization** — CPU/Memory charts and metrics
6. **Security Scanning** — CVE scanning and compliance checks
7. **Embedded LLM** — Offline AI capabilities without API keys
8. **Report Generation** — PDF/CSV cluster reports

### Teleport Strengths (k13d Cannot Replace)
1. **Zero-Trust Access** — Certificate-based authentication for all infrastructure
2. **Multi-Protocol** — SSH, K8s, databases, apps, desktops in one platform
3. **Session Recording** — Complete audit trail with playback
4. **Enterprise SSO** — Okta, Entra ID, SAML, OIDC integration
5. **Compliance Certifications** — FedRAMP, SOC 2, HIPAA, PCI DSS
6. **Access Requests** — JIT privileged access with dual authorization
7. **Multi-Cluster** — Unified access across many clusters
8. **Device Trust** — Hardware-backed device identity verification
9. **Auto-Discovery** — Automatic infrastructure enrollment

### Potential Integration

```
Developer → Teleport (authenticate) → k13d (manage K8s) → Kubernetes Cluster
                                         │
                                         └─→ AI Assistant → kubectl/MCP tools
```

In an enterprise environment:
- **Teleport** handles authentication, access control, and compliance
- **k13d** provides the day-to-day Kubernetes management experience with AI assistance
- Both tools can coexist: Teleport secures the access path while k13d enhances the operational workflow

---

## 12. Quick Decision Guide

| If you need... | Use |
|----------------|-----|
| Interactive Kubernetes dashboard | **k13d** |
| AI-powered cluster troubleshooting | **k13d** |
| k9s-style terminal navigation | **k13d** |
| Zero-trust infrastructure access | **Teleport** |
| Session recording & compliance | **Teleport** |
| Multi-infrastructure access (SSH, DB, K8s, Desktop) | **Teleport** |
| Enterprise SSO & MFA | **Teleport** |
| Offline/air-gapped K8s management | **k13d** |
| Helm release management | **k13d** |
| Quick single-cluster setup | **k13d** |
| Multi-cluster enterprise deployment | **Teleport** |
| AI agent governance (MCP) | **Teleport** |
| AI-assisted operations | **k13d** |
| Both secure access AND AI operations | **Teleport + k13d** |

---

## Version Information

- **k13d**: v0.7.0 (MIT License)
- **Teleport**: v17+ (AGPL / Commercial)
- **Comparison Date**: February 2026
