# MCP Profiles - DevOps Quick Start Guide

## Overview

MCP Profiles are pre-configured server bundles for common DevOps scenarios. Instead of manually setting up individual servers, install a profile and get all the tools you need instantly.

## Quick Start (Under 1 minute)

### 1. Start k13d Web UI

```bash
./k13d -web -port 8080
```

### 2. Open Settings

- Navigate to: **Settings** → **MCP Servers**

### 3. Choose a Profile

Click **Install** on any profile:

- **Kubernetes** - For K8s cluster management
- **Docker** - For container management
- **Full DevOps Stack** - Complete toolset
- Or any other profile

### 4. Done!

- Status shows "installed"
- Tools become available in AI chat
- Use immediately

## Available Profiles

### Essential Profiles

#### 🐳 Kubernetes (`k8s`)

**For**: K8s cluster administrators, DevOps engineers
**Includes**:

- List/describe pods, deployments, services
- View and stream logs
- Apply and delete manifests
- Check cluster events and status
- Port forwarding

**Use in chat**:

```
"What pods are running in default namespace?"
"Show me logs from the nginx pod"
"Deploy this manifest: ..."
```

#### 🐳 Docker (`docker`)

**For**: Container and image management
**Includes**:

- Build, run, manage containers
- List and inspect images
- Manage registries
- Container orchestration

**Use in chat**:

```
"Build a Docker image from Dockerfile"
"List all running containers"
"Push image to registry"
```

#### 🔧 Shell & Bash (`shell`)

**For**: System administration, automation
**Includes**:

- Execute shell commands
- Run scripts
- File operations
- System automation

**Use in chat**:

```
"Create backup of /data directory"
"Run deployment script"
"Check disk usage"
```

#### 🚀 GitHub (`github`)

**For**: Source control, repository management
**Requires**: `GITHUB_TOKEN` environment variable
**Includes**:

- Search repositories
- Read code files
- Create issues
- Manage pull requests

**Use in chat**:

```
"Find all repos tagged with 'kubernetes'"
"Create a new issue about performance"
"Check status of my pull requests"
```

#### ☁️ AWS (`aws`)

**For**: AWS infrastructure management
**Requires**: AWS credentials configured
**Includes**:

- EC2 instance management
- S3 bucket operations
- Lambda function management
- RDS database management
- CloudFormation stacks

**Use in chat**:

```
"List all EC2 instances"
"Create S3 bucket named..."
"Deploy Lambda function"
```

#### 🔄 ArgoCD (`argocd`)

**For**: GitOps and continuous deployment
**Requires**: ArgoCD server and auth token
**Includes**:

- Application deployment
- Sync status
- Repo management
- Application lifecycle

**Use in chat**:

```
"Deploy application through ArgoCD"
"Check sync status of my-app"
"Rollback to previous version"
```

#### 🧠 Sequential Thinking (`thinking`)

**For**: Complex problem solving
**Includes**:

- Step-by-step reasoning
- Task decomposition
- Logical analysis

**Use in chat**:

```
"Help me plan a K8s migration (use thinking)"
"Design disaster recovery strategy"
```

### Complete Profiles

#### 🎯 Full DevOps Stack (`fullstack`)

**Includes all of the above**: K8s + Docker + Shell + GitHub + Thinking
**Perfect for**: Teams managing complex infrastructure
**Installation**: One click gets everything

## Advanced Usage

### Installing via API

```bash
# Install Kubernetes profile
curl -X POST http://localhost:8080/api/mcp/profiles \
  -H "Content-Type: application/json" \
  -H "Cookie: session=YOUR_SESSION" \
  -d '{
    "profile_id": "k8s",
    "action": "install"
  }'

# Install Docker profile
curl -X POST http://localhost:8080/api/mcp/profiles \
  -H "Content-Type: application/json" \
  -H "Cookie: session=YOUR_SESSION" \
  -d '{
    "profile_id": "docker",
    "action": "install"
  }'

# Install Full Stack
curl -X POST http://localhost:8080/api/mcp/profiles \
  -H "Content-Type: application/json" \
  -H "Cookie: session=YOUR_SESSION" \
  -d '{
    "profile_id": "fullstack",
    "action": "install"
  }'
```

### Uninstalling via API

```bash
curl -X POST http://localhost:8080/api/mcp/profiles \
  -H "Content-Type: application/json" \
  -H "Cookie: session=YOUR_SESSION" \
  -d '{
    "profile_id": "k8s",
    "action": "uninstall"
  }'
```

### Viewing Profile Status

```bash
curl http://localhost:8080/api/mcp/profiles \
  -H "Cookie: session=YOUR_SESSION" | jq
```

## Configuration Requirements

### Kubernetes Profile

- `KUBECONFIG` - Path to kubeconfig (optional, uses default)
- kubectl must be installed
- Valid kubeconfig file

### GitHub Profile

- `GITHUB_TOKEN` - Personal access token
  ```bash
  export GITHUB_TOKEN="ghp_xxxxxxxxxxxxx"
  ```

### AWS Profile

- AWS credentials configured
  ```bash
  aws configure
  ```
- Or use environment variables:
  ```bash
  export AWS_ACCESS_KEY_ID="..."
  export AWS_SECRET_ACCESS_KEY="..."
  ```

### ArgoCD Profile

- `ARGOCD_SERVER` - ArgoCD server address
  ```bash
  export ARGOCD_SERVER="argocd.example.com"
  ```
- `ARGOCD_AUTH_TOKEN` - Auth token
  ```bash
  export ARGOCD_AUTH_TOKEN="..."
  ```

## Workflow Examples

### Setup 1: Kubernetes Only

```
Install: k8s
Chat: "What's the status of my deployment?"
AI: Uses K8s tools to check and report
```

### Setup 2: Full DevOps Stack

```
Install: fullstack
Chat: "Deploy my app from repo to prod K8s cluster"
AI: Uses Git + K8s + Docker tools to:
  1. Clone repo (GitHub)
  2. Build image (Docker)
  3. Apply manifest (Kubernetes)
```

### Setup 3: Cloud Operations

```
Install: aws, shell
Chat: "List EC2 instances and check their disk space"
AI: Uses AWS + Shell tools to:
  1. List instances (AWS)
  2. SSH and check disk (Shell)
```

### Setup 4: GitOps CI/CD

```
Install: github, argocd
Chat: "Create PR for config update and deploy via ArgoCD"
AI: Uses GitHub + ArgoCD tools to:
  1. Create pull request (GitHub)
  2. Merge and deploy (ArgoCD)
```

## Troubleshooting

### Profile Installation Fails

1. Check if npx is available: `npx --version`
2. Try installing server manually: `npm install -g @anthropic/mcp-server-kubernetes`
3. Check k13d logs: `DEBUG=true ./k13d -web -port 8080`

### Tools Not Appearing

1. Verify profile shows "installed" in settings
2. Check that servers show "connected" status
3. Restart k13d: `./k13d -web -port 8080`
4. Check browser cache (Ctrl+Shift+R or Cmd+Shift+R)

### Connection Issues

1. Verify prerequisites installed (kubectl, docker, etc.)
2. Check environment variables are set
3. Verify credentials/tokens are valid
4. Check firewall/network access to services

## Next Steps

- **[Full MCP Guide](../docs/MCP_GUIDE.md)** - Complete documentation
- **[Create Custom Profiles](../docs/MCP_GUIDE.md#creating-custom-mcp-servers)** - Build your own
- **[Configuration Guide](../docs/CONFIGURATION_GUIDE.md)** - Advanced setup

## Support

- Issues: https://github.com/kube-ai-dashboard/k13d/issues
- Documentation: https://k13d.readthedocs.io
- MCP Spec: https://modelcontextprotocol.io
