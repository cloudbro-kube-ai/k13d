# k13d vs Teleport Comparison

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

## Architecture & Philosophy

### k13d: Kubernetes-First AI Dashboard

k13d is designed as a **direct Kubernetes management tool** that combines the operational efficiency of k9s with AI-powered intelligence.

```
User → k13d (TUI/Web) → Kubernetes API Server
            │
            └→ AI Assistant (LLM) → Tool Execution
```

!!! info "Key Design Principles"
    - Direct cluster access with minimal setup
    - AI-first approach to Kubernetes troubleshooting
    - Dual interface (TUI + Web) with feature parity
    - Single binary, zero external dependencies
    - Offline-capable with embedded LLM

### Teleport: Zero-Trust Infrastructure Gateway

Teleport acts as an **identity-aware access proxy** between users and infrastructure.

```
User → tsh/Web UI → Teleport Proxy → Teleport Auth → Infrastructure
                                                        ├── SSH Servers
                                                        ├── Kubernetes
                                                        ├── Databases
                                                        ├── Web Apps
                                                        ├── Windows RDP
                                                        └── MCP Servers
```

!!! info "Key Design Principles"
    - Zero-trust: verify everything, trust nothing
    - Short-lived certificates replace long-lived secrets
    - Unified access layer across all infrastructure types
    - Complete session recording and audit
    - Compliance-first design (FedRAMP, SOC 2, HIPAA)

---

## Kubernetes Management

| Feature | k13d | Teleport |
|---------|:----:|:--------:|
| **30+ Resource Types** | :material-check: | :material-close: |
| **Interactive Resource Tables** | :material-check: | :material-close: |
| **Resource Actions (YAML, Describe, Scale)** | :material-check: | Via kubectl |
| **Built-in Log Streaming** | :material-check: | Via kubectl |
| **Built-in Shell Access** | :material-check: | Via kubectl |
| **Port Forwarding UI** | :material-check: | Via kubectl |
| **Namespace Quick-Switch** | :material-check: | :material-close: |
| **Resource Drill-Down** | :material-check: | :material-close: |
| **Multi-Cluster Access** | Single (switchable) | :material-check: |
| **Cluster Auto-Discovery** | :material-close: | :material-check: |
| **K8s RBAC Overlay** | Inherits kubeconfig | :material-check: |
| **Helm Integration** | :material-check: | :material-close: |
| **Metrics Display** | :material-check: | :material-close: |
| **Security Scanning** | :material-check: | :material-close: |

k13d provides **deep Kubernetes resource management** with interactive navigation, AI-powered analysis, and operational tools. Teleport provides **secure access to Kubernetes** through proxy-based authentication but delegates resource management to kubectl.

---

## AI & Intelligence

| Feature | k13d | Teleport |
|---------|:----:|:--------:|
| **AI Assistant** | :material-check: Core feature | :material-close: |
| **Natural Language Queries** | :material-check: | :material-close: |
| **AI Tool Execution** | :material-check: kubectl, bash, MCP | :material-close: |
| **Command Safety Analysis** | :material-check: | :material-close: |
| **8+ LLM Providers** | :material-check: | :material-close: |
| **Live Model Switching** | :material-check: | :material-close: |
| **Embedded LLM (Offline)** | :material-check: | :material-close: |
| **Streaming Responses** | :material-check: | :material-close: |
| **AI Benchmarking** | :material-check: 125+ tasks | :material-close: |
| **Session Summaries (AI)** | :material-close: | :material-check: Enterprise |
| **MCP Agent Governance** | Client mode | :material-check: Governance |
| **Agentic Identity Framework** | :material-close: | :material-check: Emerging |

k13d treats AI as a **first-class feature** for Kubernetes operations. Teleport focuses on **securing AI agents** through MCP governance rather than providing an AI assistant.

---

## User Interface

### Terminal Interface

| Feature | k13d | Teleport |
|---------|:----:|:--------:|
| **TUI Dashboard** | :material-check: k9s-style | :material-close: |
| **Vim Navigation** | :material-check: j/k, g/G | :material-close: |
| **Command Bar + Autocomplete** | :material-check: | :material-close: |
| **Filter/Regex Search** | :material-check: | :material-close: |
| **Column Sorting** | :material-check: | :material-close: |
| **Themes/Skins** | :material-check: | :material-close: |
| **Plugin System** | :material-check: | :material-close: |
| **SSH Client** | :material-close: | :material-check: `tsh ssh` |
| **Database Client** | :material-close: | :material-check: `tsh db` |
| **App Access** | :material-close: | :material-check: `tsh apps` |

### Web Interface

| Feature | k13d | Teleport |
|---------|:----:|:--------:|
| **Resource Dashboard** | :material-check: | :material-check: |
| **AI Chat Panel** | :material-check: SSE streaming | :material-close: |
| **Log Viewer** | :material-check: | :material-close: |
| **Web Terminal** | :material-check: xterm.js | :material-check: xterm.js |
| **Session Recording Playback** | :material-close: | :material-check: |
| **Live Session Sharing** | :material-close: | :material-check: |
| **Settings Panel** | :material-check: | :material-check: |
| **Access Request Workflow** | :material-close: | :material-check: |
| **Desktop App** | :material-close: | :material-check: Teleport Connect |
| **VNet (VPN Alternative)** | :material-close: | :material-check: |

---

## Authentication & Security

| Feature | k13d | Teleport |
|---------|:----:|:--------:|
| **Local Auth** | :material-check: | :material-close: |
| **Token Auth** | :material-check: | :material-close: |
| **LDAP/AD** | :material-check: | :material-close: |
| **Certificate-Based Auth** | :material-close: | :material-check: Core |
| **SSO (OIDC/SAML)** | :material-close: | :material-check: |
| **MFA / Hardware Keys** | :material-close: | :material-check: |
| **Passwordless** | :material-close: | :material-check: |
| **Device Trust (TPM)** | :material-close: | :material-check: |
| **RBAC** | 3 roles | Granular with deny rules |
| **ABAC** | :material-close: | :material-check: |
| **JIT Access Requests** | :material-close: | :material-check: |
| **Dual Authorization** | :material-close: | :material-check: |
| **Identity Locks** | :material-close: | :material-check: |
| **SCIM Provisioning** | :material-close: | :material-check: |

Teleport's authentication is **enterprise-grade** with zero-trust principles. k13d provides **practical authentication** suitable for team deployments.

---

## Audit & Compliance

| Feature | k13d | Teleport |
|---------|:----:|:--------:|
| **Action Audit Log** | :material-check: SQLite | :material-check: Structured events |
| **AI Tool Invocation Logging** | :material-check: | :material-close: |
| **Session Recording** | :material-close: | :material-check: All protocols |
| **Session Playback** | :material-close: | :material-check: |
| **Audit Export** | CSV/JSON | SIEM (Splunk, Elastic, Datadog) |
| **FedRAMP** | :material-close: | :material-check: |
| **SOC 2** | :material-close: | :material-check: |
| **HIPAA** | :material-close: | :material-check: |
| **PCI DSS** | :material-close: | :material-check: |
| **ISO 27001** | :material-close: | :material-check: |
| **FIPS Binaries** | :material-close: | :material-check: |

---

## Infrastructure Scope

| Resource Type | k13d | Teleport |
|---------------|:----:|:--------:|
| **Kubernetes Clusters** | :material-check: Direct management | :material-check: Access proxy |
| **SSH Servers** | :material-close: | :material-check: |
| **Databases** | :material-close: | :material-check: |
| **Web Applications** | :material-close: | :material-check: |
| **Windows Desktops** | :material-close: | :material-check: |
| **Cloud APIs** | :material-close: | :material-check: |
| **MCP Servers** | :material-check: Client + Server | :material-check: Governance |

k13d is **Kubernetes-specialized** with the deepest management experience. Teleport is **infrastructure-wide** with unified access control across all resource types.

---

## Deployment

| Feature | k13d | Teleport |
|---------|:----:|:--------:|
| **Single Binary** | :material-check: ~30MB | :material-check: ~100MB+ |
| **Docker** | :material-check: | :material-check: |
| **Kubernetes Manifests** | :material-check: | :material-check: Helm |
| **Air-Gapped** | :material-check: Embedded LLM | :material-check: Self-hosted |
| **Cloud SaaS** | :material-close: | :material-check: Enterprise Cloud |
| **External DB Required** | :material-close: SQLite embedded | :material-check: etcd/DynamoDB |
| **HA Setup** | :material-close: | :material-check: Multi-region |
| **Auto-Discovery** | :material-close: | :material-check: EC2, RDS, EKS |
| **Resource Requirements** | Minimal (laptop) | Moderate (production) |

---

## MCP (Model Context Protocol)

| Feature | k13d | Teleport |
|---------|:----:|:--------:|
| **MCP Client** | :material-check: Consumes tools | :material-close: |
| **MCP Server** | :material-check: Exposes K8s tools | :material-close: |
| **MCP Governance** | Per-command approval | :material-check: RBAC/ABAC |
| **Rate Limiting** | :material-close: | :material-check: |
| **Budget Controls** | :material-close: | :material-check: |
| **Agent Identity** | :material-close: | :material-check: Digital twins |
| **MCP Catalog** | :material-close: | :material-check: |

k13d uses MCP to **extend AI capabilities** with external tools. Teleport uses MCP to **govern AI agent access** — they address complementary concerns.

---

## Complementary Use Cases

!!! tip "k13d and Teleport are not competitors"
    They serve different roles and can work together effectively.

### k13d Strengths (Teleport Cannot Replace)

- :material-check: Interactive K8s Dashboard with TUI/Web
- :material-check: AI-Powered Troubleshooting
- :material-check: k9s-style Keybindings
- :material-check: Helm Management
- :material-check: Metrics Visualization
- :material-check: Security Scanning
- :material-check: Embedded LLM (Offline)
- :material-check: Report Generation

### Teleport Strengths (k13d Cannot Replace)

- :material-check: Zero-Trust Certificate Auth
- :material-check: Multi-Protocol (SSH, K8s, DB, App, Desktop)
- :material-check: Session Recording & Playback
- :material-check: Enterprise SSO (Okta, Entra ID)
- :material-check: Compliance Certifications (FedRAMP, SOC 2)
- :material-check: JIT Access Requests
- :material-check: Multi-Cluster Unified Access
- :material-check: Device Trust
- :material-check: Auto-Discovery

### Potential Integration

```
Developer → Teleport (authenticate) → k13d (manage K8s) → Cluster
                                         │
                                         └→ AI Assistant → kubectl/MCP
```

In enterprise environments, **Teleport** handles authentication, access control, and compliance while **k13d** enhances the operational Kubernetes workflow with AI assistance.

---

## Quick Decision Guide

| If you need... | Use |
|----------------|-----|
| Interactive Kubernetes dashboard | **k13d** |
| AI-powered cluster troubleshooting | **k13d** |
| k9s-style terminal navigation | **k13d** |
| Zero-trust infrastructure access | **Teleport** |
| Session recording & compliance | **Teleport** |
| Multi-infrastructure access | **Teleport** |
| Enterprise SSO & MFA | **Teleport** |
| Offline K8s management | **k13d** |
| Helm release management | **k13d** |
| Quick single-cluster setup | **k13d** |
| Multi-cluster enterprise deployment | **Teleport** |
| AI agent governance (MCP) | **Teleport** |
| AI-assisted operations | **k13d** |
| Secure access AND AI operations | **Teleport + k13d** |

---

## Version Information

- **k13d**: v0.7.0 (MIT License)
- **Teleport**: v17+ (AGPL / Commercial)
- **Comparison Date**: February 2026
