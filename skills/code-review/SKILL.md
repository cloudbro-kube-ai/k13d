---
name: code-review
description: Perform comprehensive code reviews focusing on security, performance, testing, and design. Use this skill when reviewing PRs, auditing code changes, or ensuring code quality. Based on Sentry's engineering practices.
version: 1.0.0
---

# Code Review Skill

Comprehensive code review framework emphasizing risk reduction across security, performance, testing, and design dimensions.

## When to Use

- Reviewing pull requests
- Auditing code changes before merge
- Ensuring code quality standards
- Identifying potential bugs and vulnerabilities

## Review Methodology

### Phase 1: Gather Context

```bash
# Get complete diff
git diff main...HEAD

# List all modified files
git diff --name-only main...HEAD
```

### Phase 2: Attack Surface Mapping

For each modified file, catalog:
- User inputs and data sources
- Database queries and ORM operations
- Authentication/authorization checks
- External API calls
- Cryptographic operations
- File system operations

### Phase 3: Problem Identification

#### Security Issues
- [ ] SQL/NoSQL injection vulnerabilities
- [ ] XSS (Cross-Site Scripting) risks
- [ ] Command injection possibilities
- [ ] Path traversal vulnerabilities
- [ ] Insecure deserialization
- [ ] Hardcoded credentials or secrets
- [ ] Missing input validation
- [ ] Improper error handling exposing internals

#### Performance Issues
- [ ] N+1 query patterns
- [ ] Missing database indexes
- [ ] Unbounded loops or recursion
- [ ] Memory leaks
- [ ] Blocking operations in async code
- [ ] Missing caching opportunities
- [ ] Inefficient algorithms

#### Kubernetes-Specific Issues
- [ ] Resource limits not set
- [ ] Missing health checks
- [ ] Improper context handling
- [ ] Client-go best practices violations
- [ ] Informer/cache usage issues
- [ ] RBAC permission escalation

#### Go-Specific Issues
- [ ] Goroutine leaks
- [ ] Race conditions
- [ ] Improper error handling
- [ ] Context cancellation not respected
- [ ] Defer in loops
- [ ] Nil pointer dereferences
- [ ] Slice/map initialization issues

### Phase 4: Test Coverage

Every PR should have appropriate test coverage:

```
✓ Unit tests for new functions
✓ Integration tests for API changes
✓ Edge case handling
✓ Error path testing
✓ Concurrent access testing (if applicable)
```

### Phase 5: Design Evaluation

- Does the change align with existing architecture?
- Are components properly decoupled?
- Is the API consistent with existing patterns?
- Are abstractions appropriate (not over/under-engineered)?

## Escalation Triggers

Flag for senior review when PR involves:
- Database schema changes
- API breaking changes
- New external dependencies
- Security-sensitive code paths
- Performance-critical sections
- Authentication/authorization changes

## Feedback Guidelines

### DO
- Provide actionable suggestions with code examples
- Phrase uncertain feedback as questions
- Acknowledge good patterns and improvements
- Focus on high-impact issues first

### DON'T
- Block PRs over style preferences
- Provide vague criticism without solutions
- Nitpick on non-consequential details
- Ignore the broader context

## Output Format

```markdown
## Code Review Summary

### Critical Issues
| File | Line | Severity | Issue | Suggestion |
|------|------|----------|-------|------------|
| ... | ... | HIGH | ... | ... |

### Warnings
| File | Line | Category | Issue | Suggestion |
|------|------|----------|-------|------------|
| ... | ... | PERF | ... | ... |

### Suggestions
- ...

### Positive Observations
- ...
```

## Priority Order

1. **Security vulnerabilities** - Must fix before merge
2. **Bugs** - Likely to cause runtime issues
3. **Performance** - May cause degradation
4. **Code quality** - Maintainability concerns
5. **Style** - Only mention if egregious
