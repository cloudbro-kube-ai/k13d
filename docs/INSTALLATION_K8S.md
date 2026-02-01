# k13d Kubernetes Installation Guide

This guide covers deploying k13d in a Kubernetes cluster with various configurations.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Deployment Options](#deployment-options)
- [Kubernetes Version Compatibility](#kubernetes-version-compatibility)
- [RBAC Configuration](#rbac-configuration)
- [With Ollama Sidecar](#with-ollama-sidecar)
- [High Availability](#high-availability)
- [Ingress Configuration](#ingress-configuration)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

- Kubernetes 1.25+ cluster
- kubectl configured with cluster access
- (Optional) Helm 3.x for Helm installation

---

## Quick Start

### Using kubectl

```bash
# Create namespace
kubectl create namespace k13d

# Apply all-in-one manifest
kubectl apply -f https://raw.githubusercontent.com/youngjukim/k13d/main/deploy/kubernetes/all-in-one.yaml

# Wait for pod to be ready
kubectl wait --for=condition=ready pod -l app=k13d -n k13d --timeout=120s

# Port forward to access
kubectl port-forward -n k13d svc/k13d 8080:80

# Open http://localhost:8080
```

### Using Helm

```bash
# Add Helm repository
helm repo add k13d https://youngjukim.github.io/k13d
helm repo update

# Install with default settings
helm install k13d k13d/k13d -n k13d --create-namespace

# Install with custom values
helm install k13d k13d/k13d -n k13d --create-namespace \
  --set llm.provider=solar \
  --set llm.model=solar-pro2 \
  --set llm.apiKey=your-api-key
```

---

## Deployment Options

### Basic Deployment

```yaml
# k13d-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k13d
  namespace: k13d
spec:
  replicas: 1
  selector:
    matchLabels:
      app: k13d
  template:
    metadata:
      labels:
        app: k13d
    spec:
      serviceAccountName: k13d
      containers:
      - name: k13d
        image: youngjukim/k13d:latest
        ports:
        - containerPort: 8080
        env:
        - name: K13D_AUTH_MODE
          value: "token"
        - name: K13D_LLM_PROVIDER
          value: "solar"
        - name: K13D_LLM_MODEL
          value: "solar-pro2"
        - name: K13D_LLM_API_KEY
          valueFrom:
            secretKeyRef:
              name: k13d-secrets
              key: llm-api-key
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: k13d
  namespace: k13d
spec:
  type: ClusterIP
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: k13d
```

### With ConfigMap and Secrets

```yaml
# k13d-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: k13d-config
  namespace: k13d
data:
  config.yaml: |
    llm:
      provider: solar
      model: solar-pro2
      endpoint: https://api.upstage.ai/v1
    language: en
    beginner_mode: true
    enable_audit: true
---
apiVersion: v1
kind: Secret
metadata:
  name: k13d-secrets
  namespace: k13d
type: Opaque
stringData:
  llm-api-key: "your-api-key-here"
  admin-password: "your-admin-password"
```

---

## Kubernetes Version Compatibility

k13d is tested and compatible with the following Kubernetes versions:

| k13d Version | Kubernetes Versions | Notes |
|--------------|---------------------|-------|
| v1.0.x | 1.25 - 1.32 | Full support |
| v1.0.x | 1.23 - 1.24 | Limited support (deprecated APIs) |

### Version-Specific Notes

#### Kubernetes 1.29+
```yaml
# Uses stable APIs, no special configuration needed
apiVersion: apps/v1
kind: Deployment
```

#### Kubernetes 1.25-1.28
```yaml
# PodSecurityPolicy replaced by Pod Security Admission
# Add namespace labels for pod security
apiVersion: v1
kind: Namespace
metadata:
  name: k13d
  labels:
    pod-security.kubernetes.io/enforce: baseline
    pod-security.kubernetes.io/warn: restricted
```

#### Kubernetes 1.23-1.24
```yaml
# May require additional RBAC for deprecated APIs
# Ensure client-go compatibility
```

---

## RBAC Configuration

k13d requires specific RBAC permissions to access cluster resources.

### ServiceAccount and ClusterRole

```yaml
# k13d-rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: k13d
  namespace: k13d
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k13d
rules:
# Core resources - read access
- apiGroups: [""]
  resources: ["pods", "pods/log", "services", "endpoints", "configmaps",
              "secrets", "namespaces", "nodes", "events",
              "persistentvolumes", "persistentvolumeclaims",
              "serviceaccounts", "replicationcontrollers"]
  verbs: ["get", "list", "watch"]

# Core resources - write access (for AI operations)
- apiGroups: [""]
  resources: ["pods", "services", "configmaps", "secrets"]
  verbs: ["create", "update", "patch", "delete"]

# Pod exec/portforward
- apiGroups: [""]
  resources: ["pods/exec", "pods/portforward"]
  verbs: ["create"]

# Apps resources
- apiGroups: ["apps"]
  resources: ["deployments", "daemonsets", "statefulsets", "replicasets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Batch resources
- apiGroups: ["batch"]
  resources: ["jobs", "cronjobs"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Networking
- apiGroups: ["networking.k8s.io"]
  resources: ["ingresses", "networkpolicies"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# RBAC (read-only)
- apiGroups: ["rbac.authorization.k8s.io"]
  resources: ["roles", "rolebindings", "clusterroles", "clusterrolebindings"]
  verbs: ["get", "list", "watch"]

# Metrics
- apiGroups: ["metrics.k8s.io"]
  resources: ["pods", "nodes"]
  verbs: ["get", "list"]

# Autoscaling
- apiGroups: ["autoscaling"]
  resources: ["horizontalpodautoscalers"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Storage
- apiGroups: ["storage.k8s.io"]
  resources: ["storageclasses"]
  verbs: ["get", "list", "watch"]

# CRDs (read-only)
- apiGroups: ["apiextensions.k8s.io"]
  resources: ["customresourcedefinitions"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: k13d
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: k13d
subjects:
- kind: ServiceAccount
  name: k13d
  namespace: k13d
```

### Read-Only Mode

For restricted environments:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k13d-readonly
rules:
- apiGroups: ["", "apps", "batch", "networking.k8s.io", "autoscaling"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["metrics.k8s.io"]
  resources: ["pods", "nodes"]
  verbs: ["get", "list"]
```

---

## With Ollama Sidecar

Deploy k13d with Ollama for fully local AI:

```yaml
# k13d-with-ollama.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k13d
  namespace: k13d
spec:
  replicas: 1
  selector:
    matchLabels:
      app: k13d
  template:
    metadata:
      labels:
        app: k13d
    spec:
      serviceAccountName: k13d
      containers:
      # k13d container
      - name: k13d
        image: youngjukim/k13d:latest
        ports:
        - containerPort: 8080
        env:
        - name: K13D_LLM_PROVIDER
          value: "ollama"
        - name: K13D_LLM_MODEL
          value: "qwen2.5:7b"
        - name: K13D_LLM_ENDPOINT
          value: "http://localhost:11434/v1"
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"

      # Ollama sidecar
      - name: ollama
        image: ollama/ollama:latest
        ports:
        - containerPort: 11434
        volumeMounts:
        - name: ollama-models
          mountPath: /root/.ollama
        resources:
          requests:
            memory: "4Gi"
            cpu: "1"
          limits:
            memory: "8Gi"
            cpu: "4"
        # GPU support (optional)
        # resources:
        #   limits:
        #     nvidia.com/gpu: 1

      volumes:
      - name: ollama-models
        persistentVolumeClaim:
          claimName: ollama-models-pvc
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ollama-models-pvc
  namespace: k13d
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
```

After deployment, pull a model:

```bash
kubectl exec -n k13d deploy/k13d -c ollama -- ollama pull qwen2.5:7b
```

---

## High Availability

For production deployments:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k13d
  namespace: k13d
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 1
  selector:
    matchLabels:
      app: k13d
  template:
    metadata:
      labels:
        app: k13d
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: k13d
              topologyKey: kubernetes.io/hostname
      # ... rest of spec
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: k13d-pdb
  namespace: k13d
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: k13d
```

---

## Ingress Configuration

### NGINX Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: k13d
  namespace: k13d
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-body-size: "50m"
    # For SSE streaming
    nginx.ingress.kubernetes.io/proxy-buffering: "off"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - k13d.example.com
    secretName: k13d-tls
  rules:
  - host: k13d.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: k13d
            port:
              number: 80
```

### Traefik Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: k13d
  namespace: k13d
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: websecure
    traefik.ingress.kubernetes.io/router.tls: "true"
spec:
  ingressClassName: traefik
  # ... same as above
```

---

## Troubleshooting

### Pod not starting

```bash
# Check pod status
kubectl get pods -n k13d
kubectl describe pod -l app=k13d -n k13d

# Check logs
kubectl logs -l app=k13d -n k13d
```

### RBAC permission errors

```bash
# Check if ServiceAccount can access resources
kubectl auth can-i get pods --as=system:serviceaccount:k13d:k13d

# List RBAC bindings
kubectl get clusterrolebindings | grep k13d
```

### Cannot connect to cluster API

```bash
# Verify ServiceAccount token
kubectl exec -n k13d deploy/k13d -- kubectl cluster-info

# Check API server access
kubectl exec -n k13d deploy/k13d -- curl -sk https://kubernetes.default/api
```

### Ollama model not loading

```bash
# Check Ollama logs
kubectl logs -n k13d deploy/k13d -c ollama

# Check disk space
kubectl exec -n k13d deploy/k13d -c ollama -- df -h /root/.ollama

# Manually pull model
kubectl exec -n k13d deploy/k13d -c ollama -- ollama pull qwen2.5:7b
```

---

## Next Steps

- [Configuration Guide](./CONFIGURATION_GUIDE.md)
- [User Guide](./USER_GUIDE.md)
- [Docker Guide](./INSTALLATION_DOCKER.md)
