# K13D AI Benchmark

AI benchmark suite for evaluating AI agents on Kubernetes tasks. Inspired by [k8s-ai-bench](https://github.com/gke-labs/k8s-ai-bench).

## Overview

This benchmark evaluates AI agents' ability to perform common Kubernetes operations including:
- **Creation tasks**: Creating pods, services, deployments
- **Troubleshooting tasks**: Fixing CrashLoopBackOff, ImagePullBackOff, etc.
- **Operations tasks**: Scaling, updating, rolling back deployments
- **Networking tasks**: Creating services, ingresses, network policies

---

## Prerequisites

### Required
- **Go 1.25.0+**: For building the benchmark tool
- **kubectl**: For Kubernetes operations
- **Kubernetes cluster access**: Either existing cluster or Kind/vCluster

### Optional
- **Kind**: For automatic cluster provisioning (`kind` command in PATH)
- **vCluster**: For isolated virtual cluster provisioning

### LLM API Keys (at least one required)
```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# Anthropic
export ANTHROPIC_API_KEY="sk-ant-..."

# Ollama (local, no key required)
# Just run: ollama serve
```

---

## Installation

```bash
# Clone the repository
git clone https://github.com/kube-ai-dashboard/kube-ai-dashboard-cli.git
cd kube-ai-dashboard-cli

# Build the benchmark tool
go build -o k13d-bench ./cmd/bench/

# Verify installation
./k13d-bench --help
```

---

## Quick Start

### 1. List Available Tasks

```bash
./k13d-bench list --task-dir benchmarks/tasks
```

Output:
```
Found 5 tasks:

ID                        DIFFICULTY CATEGORY        DESCRIPTION
--------------------------------------------------------------------------------
create-pod                easy       creation        Create a simple nginx pod...
scale-deployment          easy       operations      Scale a deployment to N...
create-service            easy       networking      Create a ClusterIP service...
fix-crashloop             medium     troubleshooting Fix CrashLoopBackOff
fix-image-pull            medium     troubleshooting Fix ImagePullBackOff
```

### 2. Run Benchmarks

```bash
# Run all benchmarks with GPT-4
./k13d-bench run \
  --task-dir benchmarks/tasks \
  --llm-provider openai \
  --llm-model gpt-4 \
  --auto-approve

# Run only easy tasks
./k13d-bench run --difficulty easy --llm-provider openai --llm-model gpt-4

# Run specific task pattern
./k13d-bench run --task-pattern "fix-.*" --llm-provider anthropic --llm-model claude-3-opus-20240229
```

### 3. Analyze Results

```bash
# View summary
./k13d-bench analyze --input-dir .build/bench

# Export as Markdown report
./k13d-bench analyze --input-dir .build/bench --output-format markdown --output report.md

# Export as JSON
./k13d-bench analyze --input-dir .build/bench --output-format json --output results.json
```

---

## Configuration

### CLI Options Reference

#### `run` Command

| Flag | Default | Description |
|------|---------|-------------|
| `--task-dir` | `benchmarks/tasks` | Directory containing benchmark tasks |
| `--task-pattern` | `""` | Regex pattern to filter tasks (e.g., `"fix-.*"`) |
| `--difficulty` | `""` | Filter by difficulty: `easy`, `medium`, `hard` |
| `--categories` | `""` | Filter by categories (comma-separated) |
| `--tags` | `""` | Filter by tags (comma-separated) |
| `--parallelism` | `1` | Number of parallel workers |
| `--timeout` | `10m` | Default task timeout |
| `--retries` | `0` | Number of retries per task |
| `--output-dir` | `.build/bench` | Directory for results |
| `--output-format` | `markdown` | Output format: `json`, `jsonl`, `yaml`, `markdown` |

**Cluster Options:**

| Flag | Default | Description |
|------|---------|-------------|
| `--cluster-provider` | `existing` | Cluster provider: `existing`, `kind`, `vcluster` |
| `--kubeconfig` | `""` | Path to kubeconfig file |
| `--cluster-name` | `""` | Cluster name (for kind/vcluster) |

**LLM Options:**

| Flag | Default | Description |
|------|---------|-------------|
| `--llm-provider` | `openai` | LLM provider: `openai`, `anthropic`, `ollama`, `azure` |
| `--llm-model` | `gpt-4` | LLM model name |
| `--llm-endpoint` | `""` | Custom API endpoint |
| `--llm-api-key` | `""` | API key (overrides env var) |
| `--enable-tools` | `true` | Enable tool/function calling |
| `--auto-approve` | `true` | Auto-approve tool executions |

#### `analyze` Command

| Flag | Default | Description |
|------|---------|-------------|
| `--input-dir` | `.build/bench` | Directory containing results |
| `--output-format` | `markdown` | Output format |
| `--output` | `""` | Output file (stdout if empty) |

#### `list` Command

| Flag | Default | Description |
|------|---------|-------------|
| `--task-dir` | `benchmarks/tasks` | Directory containing tasks |
| `--difficulty` | `""` | Filter by difficulty |
| `--categories` | `""` | Filter by categories |
| `--tags` | `""` | Filter by tags |

---

## LLM Provider Configuration

### OpenAI

```bash
export OPENAI_API_KEY="sk-..."

./k13d-bench run \
  --llm-provider openai \
  --llm-model gpt-4 \
  --enable-tools
```

Supported models: `gpt-4`, `gpt-4-turbo`, `gpt-4o`, `gpt-3.5-turbo`

### Anthropic

```bash
export ANTHROPIC_API_KEY="sk-ant-..."

./k13d-bench run \
  --llm-provider anthropic \
  --llm-model claude-3-opus-20240229 \
  --enable-tools
```

Supported models: `claude-3-opus-20240229`, `claude-3-sonnet-20240229`, `claude-3-haiku-20240307`

### Ollama (Local)

```bash
# Start Ollama server
ollama serve

# Pull model
ollama pull llama3

# Run benchmark
./k13d-bench run \
  --llm-provider ollama \
  --llm-model llama3 \
  --llm-endpoint http://localhost:11434
```

### Azure OpenAI

```bash
export AZURE_OPENAI_API_KEY="..."
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"

./k13d-bench run \
  --llm-provider azure \
  --llm-model gpt-4 \
  --llm-endpoint $AZURE_OPENAI_ENDPOINT \
  --llm-api-key $AZURE_OPENAI_API_KEY
```

---

## Cluster Provider Configuration

### Existing Cluster (Default)

Uses your current kubeconfig:

```bash
# Uses default kubeconfig (~/.kube/config)
./k13d-bench run --cluster-provider existing

# Specify kubeconfig
./k13d-bench run --cluster-provider existing --kubeconfig /path/to/kubeconfig
```

### Kind (Auto-Provisioned)

Automatically creates and manages Kind clusters:

```bash
# Install Kind first
# https://kind.sigs.k8s.io/docs/user/quick-start/#installation

./k13d-bench run \
  --cluster-provider kind \
  --cluster-name k13d-bench-cluster
```

### vCluster (Isolated)

Creates virtual clusters for isolation:

```bash
# Install vCluster first
# https://www.vcluster.com/docs/getting-started/setup

./k13d-bench run \
  --cluster-provider vcluster \
  --cluster-name k13d-bench-vcluster
```

---

## Metrics and Scoring

### Extracted Metrics

The benchmark system extracts the following metrics:

#### 1. Overall Metrics

| Metric | Description | Calculation |
|--------|-------------|-------------|
| **Pass@1** | First-attempt success rate | `(success_count / total_tasks) * 100` |
| **Pass@5** | Success rate within 5 attempts | With retries enabled |
| **Total Tasks** | Number of tasks executed | Count of all task runs |
| **Success Count** | Tasks completed successfully | Verifier exit code = 0 |
| **Fail Count** | Tasks that failed verification | Verifier exit code ≠ 0 |
| **Error Count** | Tasks with execution errors | Setup/agent errors |
| **Timeout Count** | Tasks that exceeded timeout | Duration > timeout |

#### 2. Difficulty Breakdown

| Difficulty | Success | Total | Pass Rate |
|------------|---------|-------|-----------|
| Easy | X | Y | X/Y * 100% |
| Medium | X | Y | X/Y * 100% |
| Hard | X | Y | X/Y * 100% |

#### 3. Per-LLM Metrics

For each LLM model tested:

| Metric | Description |
|--------|-------------|
| **Success Count** | Tasks successfully completed |
| **Fail Count** | Tasks that failed |
| **Error Count** | Tasks with errors |
| **Pass Rate** | Success percentage |
| **Avg Duration** | Average time per task |

#### 4. Per-Task Metrics

For each individual task:

| Metric | Description |
|--------|-------------|
| **Result** | success, fail, error, timeout, skipped |
| **Duration** | Time taken to complete |
| **Failures** | List of failed expectations |
| **Output** | AI agent's response |
| **Setup Log** | Setup script output |
| **Verify Log** | Verifier script output |

### Sample Report Output

```markdown
# K13D AI Benchmark Results

**Run ID:** a1b2c3d4
**Duration:** 5m 32s
**Date:** 2024-01-15T10:30:00Z

## Overall Summary

| Metric | Value |
|--------|-------|
| Total Tasks | 10 |
| Success | 7 |
| Failed | 2 |
| Errors | 1 |
| Pass@1 | 70.0% |

## Results by Difficulty

| Difficulty | Success | Total | Rate |
|------------|---------|-------|------|
| Easy | 4 | 4 | 100.0% |
| Medium | 2 | 4 | 50.0% |
| Hard | 1 | 2 | 50.0% |

## Results by LLM

| Model | Success | Failed | Errors | Pass Rate | Avg Duration |
|-------|---------|--------|--------|-----------|--------------|
| gpt-4 | 7 | 2 | 1 | 70.0% | 45s |

## Detailed Results

### create-pod
**Create Nginx Pod** (easy)

| LLM | Result | Duration | Notes |
|-----|--------|----------|-------|
| gpt-4 | ✅ | 12s | |

### fix-crashloop
**Fix CrashLoopBackOff** (medium)

| LLM | Result | Duration | Notes |
|-----|--------|----------|-------|
| gpt-4 | ❌ | 1m 30s | verifier failed: pod not running |
```

### JSON Output Format

```json
{
  "summary": {
    "runId": "a1b2c3d4",
    "startTime": "2024-01-15T10:30:00Z",
    "endTime": "2024-01-15T10:35:32Z",
    "duration": "5m32s",
    "totalTasks": 10,
    "successCount": 7,
    "failCount": 2,
    "errorCount": 1,
    "passAt1": 70.0,
    "easySuccess": 4,
    "easyTotal": 4,
    "mediumSuccess": 2,
    "mediumTotal": 4,
    "hardSuccess": 1,
    "hardTotal": 2,
    "llmResults": {
      "gpt-4": {
        "totalTasks": 10,
        "successCount": 7,
        "failCount": 2,
        "errorCount": 1,
        "passRate": 70.0,
        "avgDuration": "45s"
      }
    }
  },
  "results": [
    {
      "taskId": "create-pod",
      "taskName": "Create Nginx Pod",
      "difficulty": "easy",
      "llmConfig": {
        "id": "openai-gpt-4",
        "provider": "openai",
        "model": "gpt-4"
      },
      "result": "success",
      "startTime": "2024-01-15T10:30:00Z",
      "endTime": "2024-01-15T10:30:12Z",
      "duration": "12s",
      "output": "I'll create a pod named 'web-server'..."
    }
  ]
}
```

---

## Task Structure

### Directory Layout

```
benchmarks/tasks/
├── create-pod/
│   ├── task.yaml       # Task definition (required)
│   ├── setup.sh        # Setup script (optional)
│   ├── verify.sh       # Verification script (required)
│   ├── cleanup.sh      # Cleanup script (optional)
│   ├── prompt.txt      # External prompt file (optional)
│   └── artifacts/      # K8s manifests (optional)
│       └── deployment.yaml
├── fix-crashloop/
│   └── ...
└── ...
```

### task.yaml Schema

```yaml
# Metadata
id: create-pod                    # Task ID (defaults to directory name)
name: Create Nginx Pod            # Human-readable name
description: Create a pod...      # Description
category: creation                # Category: creation, troubleshooting, operations, networking
difficulty: easy                  # Difficulty: easy, medium, hard
disabled: false                   # Skip this task if true
tags:                             # Tags for filtering
  - pods
  - basics

# Execution
timeout: 10m                      # Task timeout (default: 10m)
isolation: namespace              # Isolation: namespace, cluster, or empty

# Prompts (at least one required)
script:
  - prompt: |                     # Inline prompt
      Create a pod named 'web-server'...
  - promptFile: advanced.txt      # Load from file
    timeout: 5m                   # Per-prompt timeout

# Scripts
setup: setup.sh                   # Runs before agent (optional)
verifier: verify.sh               # Validates success (required)
cleanup: cleanup.sh               # Runs after task (optional)

# Expectations (optional)
expect:
  - contains: "pod.*created"      # Regex that output must match
  - notContains: "error"          # Regex that output must NOT match
```

### Script Environment Variables

Scripts receive these environment variables:

| Variable | Description |
|----------|-------------|
| `KUBECONFIG` | Path to kubeconfig file |
| `NAMESPACE` | Isolated namespace for the task |

### Verification Rules

A task is considered **successful** if:
1. All `expect.contains` patterns match the agent output
2. No `expect.notContains` patterns match the agent output
3. The `verify.sh` script exits with code 0

---

## Available Tasks

### Easy Tasks

| Task | Category | Description |
|------|----------|-------------|
| `create-pod` | creation | Create an nginx pod with specific labels and resource limits |
| `scale-deployment` | operations | Scale a deployment from 1 to 3 replicas |
| `create-service` | networking | Create a ClusterIP service for a deployment |

### Medium Tasks

| Task | Category | Description |
|------|----------|-------------|
| `fix-crashloop` | troubleshooting | Diagnose and fix CrashLoopBackOff caused by invalid command |
| `fix-image-pull` | troubleshooting | Fix ImagePullBackOff by updating to valid image tag |

---

## Adding New Tasks

### Step 1: Create Task Directory

```bash
mkdir -p benchmarks/tasks/my-new-task/artifacts
```

### Step 2: Create task.yaml

```yaml
name: My New Task
description: Description of what this task tests
category: creation
difficulty: medium
tags:
  - pods
  - configmaps

timeout: 10m

script:
  - prompt: |
      You have a deployment named 'my-app' in the current namespace.

      Please:
      1. First step instruction
      2. Second step instruction
      3. Verify the result

setup: setup.sh
verifier: verify.sh
cleanup: cleanup.sh

expect:
  - contains: "success|created|completed"
```

### Step 3: Create setup.sh

```bash
#!/bin/bash
set -e

echo "Setting up my-new-task..."

# Clean up any existing resources
kubectl delete deployment my-app --namespace="${NAMESPACE}" --ignore-not-found=true

# Create initial state
cat <<EOF | kubectl apply --namespace="${NAMESPACE}" -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: app
        image: nginx:1.25
EOF

# Wait for deployment
kubectl rollout status deployment/my-app --namespace="${NAMESPACE}" --timeout=60s

echo "Setup complete."
```

### Step 4: Create verify.sh

```bash
#!/bin/bash
set -e

echo "Verifying my-new-task..."

# Check if resource exists
if ! kubectl get deployment my-app --namespace="${NAMESPACE}" &>/dev/null; then
    echo "ERROR: Deployment 'my-app' not found"
    exit 1
fi

# Check specific conditions
REPLICAS=$(kubectl get deployment my-app --namespace="${NAMESPACE}" -o jsonpath='{.status.readyReplicas}')
if [ "$REPLICAS" != "3" ]; then
    echo "ERROR: Expected 3 replicas, got ${REPLICAS:-0}"
    exit 1
fi

echo "Verification PASSED"
exit 0
```

### Step 5: Create cleanup.sh

```bash
#!/bin/bash
echo "Cleaning up my-new-task..."
kubectl delete deployment my-app --namespace="${NAMESPACE}" --ignore-not-found=true
echo "Cleanup complete."
```

### Step 6: Make Scripts Executable

```bash
chmod +x benchmarks/tasks/my-new-task/*.sh
```

### Step 7: Test Your Task

```bash
./k13d-bench list --task-dir benchmarks/tasks
./k13d-bench run --task-pattern "my-new-task" --llm-provider openai --llm-model gpt-4
```

---

## Troubleshooting

### Common Issues

**1. "no tasks found matching criteria"**
```bash
# Check task directory exists
ls -la benchmarks/tasks/

# Verify task.yaml files exist
find benchmarks/tasks -name "task.yaml"
```

**2. "failed to get kubeconfig"**
```bash
# Check kubectl access
kubectl cluster-info

# Specify kubeconfig explicitly
./k13d-bench run --kubeconfig ~/.kube/config
```

**3. "AI request failed"**
```bash
# Check API key
echo $OPENAI_API_KEY

# Try with different provider
./k13d-bench run --llm-provider ollama --llm-model llama3
```

**4. Task timeout**
```bash
# Increase timeout
./k13d-bench run --timeout 15m
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      k13d-bench CLI                          │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐              │
│  │   run    │    │ analyze  │    │   list   │              │
│  └────┬─────┘    └────┬─────┘    └────┬─────┘              │
└───────┼───────────────┼───────────────┼─────────────────────┘
        │               │               │
        ▼               ▼               ▼
┌─────────────────────────────────────────────────────────────┐
│                    Benchmark Runner                          │
│  ┌─────────────┐  ┌───────────────┐  ┌─────────────────┐   │
│  │ Task Loader │  │ Cluster Mgr   │  │   AI Client     │   │
│  │             │  │               │  │                 │   │
│  │ - Load YAML │  │ - Kind        │  │ - OpenAI       │   │
│  │ - Filter    │  │ - vCluster    │  │ - Anthropic    │   │
│  │ - Validate  │  │ - Existing    │  │ - Ollama       │   │
│  └─────────────┘  └───────────────┘  └─────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   Evaluation Flow                            │
│                                                              │
│   ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐ │
│   │  Setup  │───▶│  Agent  │───▶│ Verify  │───▶│ Cleanup │ │
│   │         │    │         │    │         │    │         │ │
│   │ setup.sh│    │ AI LLM  │    │verify.sh│    │cleanup.sh│ │
│   └─────────┘    └─────────┘    └─────────┘    └─────────┘ │
│                                                              │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Output / Analysis                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │    JSON     │  │   Markdown  │  │   Summary Stats     │ │
│  │   JSONL     │  │   Report    │  │   Pass@1, etc.      │ │
│  │   YAML      │  │             │  │                     │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

---

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/new-task`)
3. Add your benchmark task under `benchmarks/tasks/`
4. Test your task locally
5. Commit your changes (`git commit -am 'Add new benchmark task'`)
6. Push to the branch (`git push origin feature/new-task`)
7. Create a Pull Request

---

## License

Same as parent project (Apache 2.0)
