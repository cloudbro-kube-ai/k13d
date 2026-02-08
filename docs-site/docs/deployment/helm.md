# Helm Deployment

Deploy k13d using Helm for easy configuration and upgrades.

## Quick Start

```bash
# Add Helm repository
helm repo add k13d https://cloudbro-kube-ai.github.io/k13d/charts
helm repo update

# Install k13d
helm install k13d k13d/k13d \
  --set secrets.openaiApiKey=$OPENAI_API_KEY \
  --namespace k13d \
  --create-namespace

# Access
kubectl port-forward svc/k13d 8080:8080 -n k13d
```

## Chart Repository

### Add Repository

```bash
helm repo add k13d https://cloudbro-kube-ai.github.io/k13d/charts
helm repo update
```

### Search Versions

```bash
helm search repo k13d --versions
```

## Installation

### Basic Installation

```bash
helm install k13d k13d/k13d -n k13d --create-namespace
```

### With Values File

```bash
helm install k13d k13d/k13d -n k13d -f values.yaml
```

### From Source

```bash
git clone https://github.com/cloudbro-kube-ai/k13d.git
cd k13d/deploy/helm

helm install k13d . -n k13d --create-namespace
```

## Configuration

### values.yaml

```yaml
# k13d Helm Chart Values

# Image configuration
image:
  repository: cloudbro/k13d
  tag: latest
  pullPolicy: IfNotPresent

# Replicas
replicaCount: 1

# Service configuration
service:
  type: ClusterIP
  port: 8080

# Ingress configuration
ingress:
  enabled: false
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: k13d.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: k13d-tls
      hosts:
        - k13d.example.com

# LLM configuration
llm:
  provider: openai
  model: gpt-4
  # For Ollama
  # provider: ollama
  # endpoint: http://ollama:11434
  # model: llama3.2

# Authentication
auth:
  enabled: true
  # password: "" # Set via secrets

# Secrets (create externally or set values)
secrets:
  openaiApiKey: ""
  password: ""

# Resource limits
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"

# RBAC
rbac:
  create: true
  # Restrict to specific namespaces
  namespaces: []  # Empty = cluster-wide

# Service Account
serviceAccount:
  create: true
  name: k13d

# Pod Security Context
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000

# Container Security Context
securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop:
      - ALL

# Probes
livenessProbe:
  httpGet:
    path: /api/health
    port: http
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /api/health
    port: http
  initialDelaySeconds: 5
  periodSeconds: 10

# Node selector
nodeSelector: {}

# Tolerations
tolerations: []

# Affinity
affinity: {}

# Extra environment variables
extraEnv: []

# Extra volumes
extraVolumes: []

# Extra volume mounts
extraVolumeMounts: []

# Ollama sidecar
ollama:
  enabled: false
  image: ollama/ollama:latest
  model: llama3.2
  persistence:
    enabled: true
    size: 10Gi
    storageClass: ""

# Metrics
metrics:
  enabled: false
  serviceMonitor:
    enabled: false
```

## Common Configurations

### With Ingress

```yaml
# values-ingress.yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
  hosts:
    - host: k13d.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: k13d-tls
      hosts:
        - k13d.example.com
```

```bash
helm install k13d k13d/k13d -f values-ingress.yaml -n k13d
```

### With Ollama

```yaml
# values-ollama.yaml
llm:
  provider: ollama
  endpoint: http://localhost:11434
  model: llama3.2

ollama:
  enabled: true
  model: llama3.2
  persistence:
    enabled: true
    size: 20Gi
```

```bash
helm install k13d k13d/k13d -f values-ollama.yaml -n k13d
```

### High Availability

```yaml
# values-ha.yaml
replicaCount: 3

affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app.kubernetes.io/name: k13d
          topologyKey: kubernetes.io/hostname

resources:
  requests:
    memory: "512Mi"
    cpu: "250m"
  limits:
    memory: "1Gi"
    cpu: "1000m"
```

### Namespace-Restricted

```yaml
# values-namespaced.yaml
rbac:
  create: true
  namespaces:
    - production
    - staging
```

## Secrets Management

### Using External Secrets

```yaml
# External Secrets Operator
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: k13d-secrets
spec:
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: k13d-secrets
  data:
    - secretKey: openai-api-key
      remoteRef:
        key: k13d/openai
        property: api-key
```

```yaml
# values.yaml
secrets:
  existingSecret: k13d-secrets
```

### Using Sealed Secrets

```bash
# Create sealed secret
kubectl create secret generic k13d-secrets \
  --from-literal=openai-api-key=$OPENAI_API_KEY \
  --dry-run=client -o yaml | kubeseal > sealed-secrets.yaml

kubectl apply -f sealed-secrets.yaml
```

```yaml
# values.yaml
secrets:
  existingSecret: k13d-secrets
```

## Upgrading

### Check Current Version

```bash
helm list -n k13d
```

### Upgrade to Latest

```bash
helm repo update
helm upgrade k13d k13d/k13d -n k13d
```

### Upgrade with Values

```bash
helm upgrade k13d k13d/k13d -n k13d -f values.yaml
```

### Rollback

```bash
# List revisions
helm history k13d -n k13d

# Rollback to revision
helm rollback k13d 1 -n k13d
```

## Uninstallation

```bash
# Uninstall release
helm uninstall k13d -n k13d

# Delete namespace
kubectl delete namespace k13d
```

## Customization

### Adding MCP Servers

```yaml
extraEnv:
  - name: MCP_SERVERS
    value: |
      [
        {
          "name": "thinking",
          "command": "npx",
          "args": ["-y", "@modelcontextprotocol/server-sequential-thinking"]
        }
      ]

extraVolumes:
  - name: npm-cache
    emptyDir: {}

extraVolumeMounts:
  - name: npm-cache
    mountPath: /root/.npm
```

### Custom Init Containers

```yaml
extraInitContainers:
  - name: download-plugins
    image: alpine:3
    command:
      - sh
      - -c
      - |
        wget -O /plugins/custom-plugin.so https://example.com/plugin.so
    volumeMounts:
      - name: plugins
        mountPath: /plugins

extraVolumes:
  - name: plugins
    emptyDir: {}

extraVolumeMounts:
  - name: plugins
    mountPath: /opt/k13d/plugins
```

## Troubleshooting

### Installation Fails

```bash
# Debug installation
helm install k13d k13d/k13d -n k13d --debug --dry-run

# Check template output
helm template k13d k13d/k13d -n k13d
```

### Values Not Applied

```bash
# Get deployed values
helm get values k13d -n k13d

# Get all values (including defaults)
helm get values k13d -n k13d -a
```

### RBAC Issues

```bash
# Verify RBAC
kubectl auth can-i get pods --as=system:serviceaccount:k13d:k13d

# Check cluster role
kubectl get clusterrole k13d -o yaml
```

## Next Steps

- [Kubernetes Deployment](kubernetes.md) - Manual deployment
- [Air-Gapped](air-gapped.md) - Offline deployment
- [Configuration](../getting-started/configuration.md) - Configuration guide
