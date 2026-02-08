# Kubernetes Deployment

Deploy k13d directly to your Kubernetes cluster for production use.

## Quick Start

```bash
# Apply manifests
kubectl apply -f https://raw.githubusercontent.com/cloudbro-kube-ai/k13d/main/deploy/kubernetes/deployment.yaml

# Wait for pod
kubectl wait --for=condition=ready pod -l app=k13d

# Access via port-forward
kubectl port-forward svc/k13d 8080:8080
```

Access at: http://localhost:8080

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Kubernetes Cluster                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │   Ingress    │───▶│   Service    │───▶│    Pod       │       │
│  │   (optional) │    │   (k13d)     │    │   (k13d)     │       │
│  └──────────────┘    └──────────────┘    └──────────────┘       │
│                                                 │                │
│                                          ┌──────┴──────┐        │
│                                          │  ServiceAcc │        │
│                                          │  (k13d-sa)  │        │
│                                          └─────────────┘        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Manifests

### Basic Deployment

```yaml
# k13d-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k13d
  labels:
    app: k13d
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
          image: cloudbro/k13d:latest
          args: ["-web", "-port", "8080"]
          ports:
            - containerPort: 8080
          env:
            - name: OPENAI_API_KEY
              valueFrom:
                secretKeyRef:
                  name: k13d-secrets
                  key: openai-api-key
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
spec:
  selector:
    app: k13d
  ports:
    - port: 8080
      targetPort: 8080
  type: ClusterIP
```

### RBAC Configuration

```yaml
# k13d-rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: k13d
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k13d
rules:
  # Read access to all resources
  - apiGroups: [""]
    resources: ["*"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["*"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["batch"]
    resources: ["*"]
    verbs: ["get", "list", "watch"]
  # Write access (for AI operations)
  - apiGroups: [""]
    resources: ["pods", "services", "configmaps", "secrets"]
    verbs: ["create", "update", "patch", "delete"]
  - apiGroups: ["apps"]
    resources: ["deployments", "statefulsets", "daemonsets", "replicasets"]
    verbs: ["create", "update", "patch", "delete"]
  # Scale permissions
  - apiGroups: ["apps"]
    resources: ["deployments/scale", "statefulsets/scale", "replicasets/scale"]
    verbs: ["get", "update", "patch"]
  # Logs and exec
  - apiGroups: [""]
    resources: ["pods/log", "pods/exec"]
    verbs: ["get", "create"]
  # Metrics
  - apiGroups: ["metrics.k8s.io"]
    resources: ["pods", "nodes"]
    verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: k13d
subjects:
  - kind: ServiceAccount
    name: k13d
    namespace: default
roleRef:
  kind: ClusterRole
  name: k13d
  apiGroup: rbac.authorization.k8s.io
```

### Secrets

```yaml
# k13d-secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: k13d-secrets
type: Opaque
stringData:
  openai-api-key: "sk-your-key-here"
  password: "your-secure-password"
```

### ConfigMap

```yaml
# k13d-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: k13d-config
data:
  config.yaml: |
    llm:
      provider: openai
      model: gpt-4

    auth:
      password: ${K13D_PASSWORD}

    enable_audit: true
    language: en
```

### Ingress

```yaml
# k13d-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: k13d
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
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
                  number: 8080
```

## Installation

### Apply All Manifests

```bash
# Create namespace
kubectl create namespace k13d

# Apply secrets first
kubectl apply -f k13d-secrets.yaml -n k13d

# Apply RBAC
kubectl apply -f k13d-rbac.yaml -n k13d

# Apply config
kubectl apply -f k13d-configmap.yaml -n k13d

# Apply deployment
kubectl apply -f k13d-deployment.yaml -n k13d

# Apply ingress (optional)
kubectl apply -f k13d-ingress.yaml -n k13d
```

### Verify Installation

```bash
# Check pod status
kubectl get pods -n k13d

# Check logs
kubectl logs -f deployment/k13d -n k13d

# Test health
kubectl exec -it deployment/k13d -n k13d -- curl localhost:8080/api/health
```

## Access Methods

### Port Forward

```bash
kubectl port-forward svc/k13d 8080:8080 -n k13d
```

### NodePort

```yaml
apiVersion: v1
kind: Service
metadata:
  name: k13d
spec:
  type: NodePort
  selector:
    app: k13d
  ports:
    - port: 8080
      targetPort: 8080
      nodePort: 30080
```

Access: `http://<node-ip>:30080`

### LoadBalancer

```yaml
apiVersion: v1
kind: Service
metadata:
  name: k13d
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
spec:
  type: LoadBalancer
  selector:
    app: k13d
  ports:
    - port: 80
      targetPort: 8080
```

## Advanced Configuration

### With Ollama Sidecar

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k13d
spec:
  template:
    spec:
      containers:
        - name: k13d
          image: cloudbro/k13d:latest
          args: ["-web", "-port", "8080"]
          env:
            - name: LLM_PROVIDER
              value: "ollama"
            - name: LLM_ENDPOINT
              value: "http://localhost:11434"

        - name: ollama
          image: ollama/ollama:latest
          ports:
            - containerPort: 11434
          volumeMounts:
            - name: ollama-models
              mountPath: /root/.ollama
          # Init container to pull model
          lifecycle:
            postStart:
              exec:
                command: ["/bin/sh", "-c", "sleep 10 && ollama pull llama3.2"]

      volumes:
        - name: ollama-models
          persistentVolumeClaim:
            claimName: ollama-models-pvc
```

### High Availability

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k13d
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
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
```

### Resource Limits

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "1Gi"
    cpu: "1000m"
```

## Monitoring

### Prometheus Metrics

```yaml
apiVersion: v1
kind: Service
metadata:
  name: k13d
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: "/metrics"
```

### Grafana Dashboard

Import dashboard ID: `XXXXX` (k13d dashboard)

## Security Hardening

### Pod Security

```yaml
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
        - name: k13d
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop:
                - ALL
```

### Network Policy

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: k13d
spec:
  podSelector:
    matchLabels:
      app: k13d
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
      ports:
        - protocol: TCP
          port: 8080
  egress:
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0
      ports:
        - protocol: TCP
          port: 443  # HTTPS to API
        - protocol: TCP
          port: 6443 # Kubernetes API
```

### Read-Only RBAC

For read-only access:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k13d-readonly
rules:
  - apiGroups: [""]
    resources: ["*"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps", "batch", "extensions"]
    resources: ["*"]
    verbs: ["get", "list", "watch"]
```

## Troubleshooting

### Pod CrashLoopBackOff

```bash
# Check logs
kubectl logs deployment/k13d -n k13d --previous

# Common issues:
# - Missing secrets
# - Invalid RBAC
# - Network issues
```

### Permission Denied

```bash
# Verify service account
kubectl auth can-i get pods --as=system:serviceaccount:k13d:k13d

# Check RBAC
kubectl describe clusterrolebinding k13d
```

### Cannot Access API

```bash
# Check service
kubectl get svc k13d -n k13d

# Check endpoints
kubectl get endpoints k13d -n k13d

# Test from cluster
kubectl run test --rm -it --image=curlimages/curl -- curl k13d.k13d:8080/api/health
```

## Next Steps

- [Helm](helm.md) - Deploy with Helm
- [Air-Gapped](air-gapped.md) - Offline deployment
- [Security](../concepts/security.md) - Security configuration
