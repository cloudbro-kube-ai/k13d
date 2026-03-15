# Helm Deployment

!!! warning "Beta / not supported yet"
    Helm deployment is still **in preparation** and is **not officially supported** for end users.

## Current Status

- There is no supported public Helm chart release today.
- Chart packaging, image publishing, and upgrade guidance are still being prepared.
- The current Helm assets in this repository should be treated as **reference material only**.

## Use k13d Today

The supported path today is the local binary:

```bash
# TUI
./k13d

# Web UI
./k13d --web --auth-mode local
```

## What Will Be Documented Later

When Helm support is ready, this page will cover:

- chart repository details
- supported values and defaults
- upgrade and rollback guidance
- image and version compatibility
