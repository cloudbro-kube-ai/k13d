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

## LiteLLM Sidecar / Gateway Note

If you run k13d against a separate LiteLLM gateway today, pin the LiteLLM image to a stable tag instead of `latest`.

Current stable reference used in the docs:

```bash
docker run --rm -p 4000:4000 \
  -e LITELLM_MASTER_KEY=your-master-key \
  ghcr.io/berriai/litellm:v1.82.3-stable.patch.2
```

Then point k13d at it:

```bash
export K13D_LLM_PROVIDER=litellm
export K13D_LLM_MODEL=gpt-4o-mini
export LITELLM_ENDPOINT=http://localhost:4000
```
