# Air-Gapped Deployment

Deploy k13d in environments without internet access.

!!! warning "Current scope"
    The **supported** offline path today is to move the **single k13d binary** into the target environment and run it locally.
    Docker-, Docker Compose-, and Kubernetes-based air-gapped deployment packaging is still **Beta / in preparation** and is not officially supported yet.
    There is also **no official public Docker image repository** for end users at this time.

## Supported Offline Flow Today

### 1. Download The Binary On A Connected Machine

```bash
wget https://github.com/cloudbro-kube-ai/k13d/releases/latest/download/k13d-linux-amd64.tar.gz
```

### 2. Transfer It To The Offline Environment

Use your normal transfer method:

- USB drive
- secure file transfer
- internal artifact repository

### 3. Extract And Run

```bash
tar -xzvf k13d-linux-amd64.tar.gz
chmod +x k13d

# TUI
./k13d

# Web UI
./k13d --web --auth-mode local
```

## Not Supported Yet

The following offline deployment stories are still being prepared:

- Docker image packaging
- Docker Compose packaging
- Kubernetes air-gapped deployment
- Helm-based packaging

This page will be expanded once those paths are officially supported.
