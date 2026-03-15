# Kubernetes Deployment

!!! warning "Beta / not supported yet"
    Kubernetes deployment is still **in preparation** and is **not officially supported** for end users.

## Current Status

- The manifests under `deploy/kubernetes/` are **work in progress**.
- The in-cluster deployment story is not ready for general use yet.
- There is **no official public Docker image repository** to support a stable Kubernetes deployment path at this time.

## Use k13d Today

The supported path today is to run k13d locally as a single binary against your cluster:

```bash
# TUI
./k13d

# Web UI
./k13d --web --auth-mode local
```

## About The Existing Manifests

Treat the current Kubernetes assets as:

- draft manifests for contributors
- reference material for packaging work in progress
- not a supported deployment promise

This page will be expanded once Kubernetes deployment is ready with:

- supported manifests
- image publishing details
- auth and RBAC guidance
- upgrade and operational recommendations
