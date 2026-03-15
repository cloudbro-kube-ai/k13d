# Docker Deployment

!!! warning "Beta / not supported yet"
    Docker and Docker Compose deployment are still **in preparation** and are **not officially supported** for end users.

## Current Status

- There is **no official public Docker image repository** for k13d yet.
- The Docker-related files in this repository are **work-in-progress reference material** only.
- Please do not treat them as a supported install path in the current release line.

## Use k13d Today

The supported experience today is the local single-binary flow:

```bash
# TUI
./k13d

# Web UI
./k13d --web --auth-mode local
```

## What Will Land Later

This page will be expanded when Docker support is ready, including:

- official image publication
- supported `docker run` examples
- supported Docker Compose examples
- versioning and upgrade guidance
- persistence, auth, and security recommendations
