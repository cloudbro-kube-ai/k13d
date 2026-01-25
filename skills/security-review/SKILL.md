---
name: security-review
description: Security-focused code review and vulnerability detection. Use this skill when auditing code for security issues, reviewing authentication/authorization logic, or assessing attack surfaces. Based on Trail of Bits security practices.
version: 1.0.0
---

# Security Review Skill

Comprehensive security audit framework for identifying vulnerabilities and ensuring secure coding practices.

## When to Use

- Security audits of new features
- Reviewing authentication/authorization code
- Assessing API endpoint security
- Analyzing Kubernetes RBAC configurations
- Pre-deployment security checks

## Security Audit Methodology

### Phase 1: Attack Surface Identification

Map all entry points:
- HTTP/WebSocket endpoints
- CLI arguments and environment variables
- File inputs (config files, uploads)
- Database queries
- External service integrations
- Kubernetes API interactions

### Phase 2: Security Checklist

#### A. Injection Vulnerabilities

```go
// BAD: SQL Injection
query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", userInput)

// GOOD: Parameterized query
query := "SELECT * FROM users WHERE id = $1"
db.Query(query, userInput)
```

- [ ] SQL/NoSQL injection
- [ ] Command injection
- [ ] LDAP injection
- [ ] XPath injection
- [ ] Template injection

#### B. Authentication & Authorization

```go
// GOOD: Check authorization before action
func (h *Handler) DeleteResource(w http.ResponseWriter, r *http.Request) {
    user := auth.GetUser(r.Context())
    resource := getResource(r)

    if !auth.CanDelete(user, resource) {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    // proceed with deletion
}
```

- [ ] Authentication bypass
- [ ] Session management issues
- [ ] Privilege escalation
- [ ] Missing authorization checks
- [ ] Insecure token handling

#### C. Kubernetes Security

```go
// BAD: Using cluster-admin for everything
clusterRoleBinding.RoleRef.Name = "cluster-admin"

// GOOD: Principle of least privilege
clusterRoleBinding.RoleRef.Name = "namespace-reader"
```

- [ ] Overly permissive RBAC
- [ ] Secrets exposed in logs
- [ ] Container security context
- [ ] Network policy gaps
- [ ] Service account token misuse

#### D. Data Protection

```go
// BAD: Logging sensitive data
log.Printf("User login: %s password: %s", user, password)

// GOOD: Redact sensitive information
log.Printf("User login: %s", user)
```

- [ ] Sensitive data in logs
- [ ] Unencrypted data at rest
- [ ] Insecure data transmission
- [ ] PII exposure
- [ ] Secrets in code/config

#### E. Input Validation

```go
// GOOD: Validate and sanitize input
func validateNamespace(ns string) error {
    if !namespaceRegex.MatchString(ns) {
        return fmt.Errorf("invalid namespace: %s", ns)
    }
    if len(ns) > 63 {
        return fmt.Errorf("namespace too long")
    }
    return nil
}
```

- [ ] Missing input validation
- [ ] Improper output encoding
- [ ] Path traversal
- [ ] XML external entities (XXE)
- [ ] Unsafe deserialization

#### F. Cryptography

```go
// BAD: Weak hashing
hash := md5.Sum(password)

// GOOD: Use bcrypt for passwords
hash, _ := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
```

- [ ] Weak algorithms (MD5, SHA1)
- [ ] Hardcoded keys/salts
- [ ] Insufficient key length
- [ ] Missing encryption
- [ ] Improper random generation

#### G. Error Handling

```go
// BAD: Exposing internal errors
http.Error(w, err.Error(), http.StatusInternalServerError)

// GOOD: Generic error to client, detailed logging
log.Printf("Database error: %v", err)
http.Error(w, "Internal server error", http.StatusInternalServerError)
```

- [ ] Stack traces exposed
- [ ] Database errors leaked
- [ ] Verbose error messages
- [ ] Missing error handling

### Phase 3: Web UI Security

#### XSS Prevention
```javascript
// BAD: Direct HTML insertion
element.innerHTML = userInput;

// GOOD: Text content or sanitization
element.textContent = userInput;
// or use DOMPurify for HTML
element.innerHTML = DOMPurify.sanitize(userInput);
```

#### CSRF Protection
```go
// GOOD: Validate CSRF token
func csrfMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != "GET" {
            token := r.Header.Get("X-CSRF-Token")
            if !validateCSRFToken(token, r) {
                http.Error(w, "Invalid CSRF token", http.StatusForbidden)
                return
            }
        }
        next.ServeHTTP(w, r)
    })
}
```

- [ ] XSS vulnerabilities
- [ ] CSRF protection
- [ ] Clickjacking prevention
- [ ] Content Security Policy
- [ ] Secure cookie flags

### Phase 4: Dependency Security

```bash
# Check for known vulnerabilities
govulncheck ./...

# Review go.mod for outdated dependencies
go list -u -m all
```

- [ ] Known CVEs in dependencies
- [ ] Outdated packages
- [ ] Unmaintained libraries
- [ ] Excessive permissions

## OWASP Top 10 Reference

| Risk | Check |
|------|-------|
| A01 Broken Access Control | Authorization on every action |
| A02 Cryptographic Failures | Proper encryption, no weak algorithms |
| A03 Injection | Parameterized queries, input validation |
| A04 Insecure Design | Threat modeling, security requirements |
| A05 Security Misconfiguration | Secure defaults, minimal exposure |
| A06 Vulnerable Components | Dependency scanning, updates |
| A07 Auth Failures | Strong auth, session management |
| A08 Data Integrity Failures | Input validation, secure updates |
| A09 Logging Failures | Comprehensive audit logging |
| A10 SSRF | URL validation, network restrictions |

## Output Format

```markdown
## Security Audit Report

### Critical Vulnerabilities
| ID | File:Line | Category | Description | CVSS | Remediation |
|----|-----------|----------|-------------|------|-------------|
| SEC-001 | ... | Injection | ... | 9.8 | ... |

### High Risk Issues
...

### Medium Risk Issues
...

### Low Risk / Informational
...

### Security Recommendations
1. ...
2. ...
```

## Severity Levels

- **CRITICAL**: Immediate exploitation possible, severe impact
- **HIGH**: Exploitable with moderate effort, significant impact
- **MEDIUM**: Requires specific conditions, moderate impact
- **LOW**: Minor impact, difficult to exploit
- **INFO**: Best practice recommendations
