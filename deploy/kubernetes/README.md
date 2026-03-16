# k13d Kubernetes Deployment

> [!WARNING]
> Kubernetes deployment is currently **Beta / in preparation** and is **not officially supported** for end users.
> The manifests in this directory should be treated as **work-in-progress reference material** only.
> There is also **no official public Docker image repository** available yet, so this path is not ready for general use.

## Current Recommendation

If you want to use k13d today, focus on the local single-binary experience:

```bash
# TUI
./k13d

# Web UI
./k13d --web --auth-mode local
```

## About This Directory

Files in `deploy/kubernetes/` currently exist for:

- packaging work in progress
- contributor experimentation
- future deployment documentation

They do **not** represent a supported deployment promise in the current release line.
