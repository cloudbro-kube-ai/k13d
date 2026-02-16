# Security Features

k13d provides enterprise-grade security features inspired by Teleport.

---

## Overview

| Feature | Description |
|---------|-------------|
| **Authentication** | Multi-mode auth (local, token, LDAP, SSO) |
| **Authorization** | RBAC with deny-overrides-allow |
| **Audit Logging** | Comprehensive event tracking |
| **JWT Tokens** | Short-lived, refreshable tokens |
| **User Locking** | Emergency account lockout |
| **K8s Impersonation** | Role-based impersonation headers |

---

## Authentication Modes

### Login Page

![Login Page](../images/webui-login-page.png)

| Mode | Use Case | Configuration |
|------|----------|---------------|
| **Local** | Small teams | Username/password in DB |
| **Token** | K8s native | ServiceAccount token |
| **LDAP** | Enterprise | Active Directory |
| **SSO** | Modern enterprise | OAuth2/OIDC |
| **No Auth** | Development only | `--no-auth` flag |

### Local Authentication

```yaml
auth:
  mode: local
  admin_user: admin
  admin_password: ${K13D_ADMIN_PASSWORD}
```

### Token Authentication

```bash
# Use Kubernetes ServiceAccount token
k13d -web --auth-mode token
```

### LDAP Integration

```yaml
ldap:
  enabled: true
  host: ldap.example.com
  port: 389
  use_ssl: true
  bind_dn: cn=admin,dc=example,dc=com
  bind_password: ${LDAP_PASSWORD}
  base_dn: dc=example,dc=com
  user_filter: "(uid=%s)"
  group_filter: "(member=%s)"
```

---

## RBAC Authorization

### User Authentication Control

![Auth Control](../images/webui-settings-admin-user-authentication-controll.png)

Admin panel for managing user authentication:

- Enable/disable user accounts
- Role assignment
- Session management

### Role System

k13d uses a Teleport-inspired deny-overrides-allow RBAC system.

| Role | Permissions |
|------|-------------|
| **Viewer** | Read-only access to resources |
| **User** | View + execute read operations |
| **Admin** | Full access including write/delete |

### Permission Matrix

| Action | Viewer | User | Admin |
|--------|:------:|:----:|:-----:|
| View resources | ✅ | ✅ | ✅ |
| View logs | ✅ | ✅ | ✅ |
| Execute read commands | ❌ | ✅ | ✅ |
| Execute write commands | ❌ | ⚠️* | ✅ |
| Delete resources | ❌ | ❌ | ✅ |
| Manage users | ❌ | ❌ | ✅ |
| View audit logs | ❌ | ❌ | ✅ |

*With approval required

### Resource-Level RBAC

```yaml
authorization:
  roles:
    - name: production-viewer
      resources:
        - pods
        - services
        - deployments
      namespaces:
        - production
      verbs:
        - get
        - list
        - watch
```

---

## User Management

### Add New User

![Add New User](../images/webui-settings-new-user.png)

Create new user accounts with role assignment.

| Field | Description |
|-------|-------------|
| **Username** | Unique identifier |
| **Password** | Secure password |
| **Role** | viewer, user, admin |
| **Email** | Contact email |

---

## Audit Logging

### Audit Log Storage

All actions are logged to SQLite:

| Field | Description |
|-------|-------------|
| **Timestamp** | When action occurred |
| **User** | Who performed action |
| **Action** | Type of action |
| **Resource** | Target resource |
| **Details** | Command, result, error |
| **Authorization** | Allow/deny decision |

### Logged Events

| Event Type | Description |
|------------|-------------|
| `login` | User authentication |
| `logout` | User logout |
| `query` | AI queries |
| `approve` | Tool approvals |
| `reject` | Tool rejections |
| `execute` | Command executions |
| `access_request` | JIT access requests |
| `lock` | User account locks |

### Audit API

```bash
# Get audit logs
curl http://localhost:8080/api/audit

# Filter by action
curl http://localhost:8080/api/audit?action=execute

# Filter by user
curl http://localhost:8080/api/audit?user=admin
```

---

## JWT Token Management

### Token Flow

```
┌──────────┐      ┌──────────┐      ┌──────────┐
│  Login   │─────►│  Issue   │─────►│  Store   │
│          │      │  JWT     │      │  Token   │
└──────────┘      └──────────┘      └──────────┘
     │                                    │
     │            ┌──────────┐            │
     └───────────►│  API     │◄───────────┘
                  │ Requests │
                  └──────────┘
```

### Token Configuration

```yaml
authorization:
  jwt:
    token_duration: 1h      # Token lifetime
    refresh_window: 15m     # Refresh before expiry
    signing_key: ${JWT_SECRET}
```

### Token Refresh

```bash
# Tokens auto-refresh within refresh window
# Manual refresh:
curl -X POST http://localhost:8080/api/auth/refresh \
  -H "Authorization: Bearer $TOKEN"
```

---

## Access Requests (JIT Access)

### Just-in-Time Privilege Escalation

Users can request temporary elevated access:

```
┌──────────┐      ┌──────────┐      ┌──────────┐
│ Request  │─────►│ Pending  │─────►│ Approved │
│ Access   │      │ Review   │      │ (TTL)    │
└──────────┘      └──────────┘      └──────────┘
                       │
                       ▼
                  ┌──────────┐
                  │ Denied   │
                  └──────────┘
```

### Request Configuration

```yaml
authorization:
  access_request_ttl: 30m  # Elevated access duration
```

### Self-Approval Prevention

Reviewers cannot approve their own requests.

---

## User Locking

### Emergency Lock

Admins can instantly lock user accounts:

```bash
# Lock user via API
curl -X POST http://localhost:8080/api/admin/lock \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"username": "compromised-user"}'
```

### Lock Features

| Feature | Description |
|---------|-------------|
| **Instant Effect** | All sessions invalidated |
| **DB Persisted** | Survives restart |
| **Audit Logged** | Lock event recorded |
| **Admin Only** | Requires admin role |

---

## K8s Impersonation

### Role-Based Impersonation

k13d can impersonate K8s users based on role:

```yaml
authorization:
  impersonation:
    enabled: true
    mappings:
      viewer: k13d-viewer-sa
      user: k13d-user-sa
      admin: k13d-admin-sa
```

### How It Works

```
┌──────────┐      ┌──────────┐      ┌──────────┐
│  User    │─────►│  k13d    │─────►│   K8s    │
│ (viewer) │      │ (impersonate)   │   API    │
└──────────┘      └──────────┘      └──────────┘
                       │
                       ▼
                  Impersonate: k13d-viewer-sa
```

---

## Command Safety

### Safety Analysis

![Decision Required](../images/webui-decision-required.png)

All AI commands pass through safety analysis:

| Level | Commands | Action |
|-------|----------|--------|
| **Read-Only** | get, describe, logs | Auto-approve (configurable) |
| **Write** | apply, create, patch | Require approval |
| **Dangerous** | delete, drain, rm -rf | Warning + approval |
| **Interactive** | exec, attach, port-forward | Require approval |
| **Unknown** | Non-kubectl/helm commands | Require approval (configurable) |

### TUI Safety Prompt

When a dangerous command is detected, the TUI shows an approval dialog with `Y`/`N`/`A` options before execution.

### AST Parsing

```
"kubectl get pods | xargs rm -rf /"
         │
         ▼
┌─────────────────┐
│  AST Parser     │
├─────────────────┤
│ - Pipeline      │
│ - xargs         │
│ - rm -rf        │
├─────────────────┤
│ Result: DANGER  │
└─────────────────┘
```

### Blocked Patterns

```yaml
authorization:
  tool_approval:
    blocked_patterns:
      - "rm -rf /"
      - "kubectl delete ns kube-system"
      - ":(){:|:&};"  # Fork bomb
```

---

## Security Assessment Report

![Security Assessment](../images/webui-security-assessment.png)

Generate comprehensive security reports:

- RBAC configuration review
- Network policy audit
- Vulnerability assessment
- Compliance check

---

## Best Practices

### 1. Enable Authentication

```bash
# Never run without auth in production
k13d -web --auth-mode local --admin-password "$SECURE_PASSWORD"
```

### 2. Use RBAC

```yaml
authorization:
  default_role: viewer  # Least privilege
```

### 3. Enable Audit Logging

```yaml
enable_audit: true
```

### 4. Rotate Secrets

```bash
# Rotate JWT signing key periodically
export JWT_SECRET=$(openssl rand -base64 32)
```

### 5. Review Audit Logs

Regularly review audit logs for suspicious activity.
