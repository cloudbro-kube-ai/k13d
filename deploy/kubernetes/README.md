# k13d Kubernetes Deployment

This directory contains Kubernetes manifests for deploying k13d.

## Deployment Options

| File | Description | Use Case |
|------|-------------|----------|
| `single-pod.yaml` | Single Pod, no AI | Air-gapped, minimal resources |
| `single-pod-with-ollama.yaml` | Single Pod with Ollama sidecar | Air-gapped with AI features |
| `deployment.yaml` | Full deployment with optional Ollama | Production, scalable |

## Quick Start (Air-gapped, No External Dependencies)

### Option 1: Basic Dashboard (No AI)

```bash
# 1. Build and save image (on internet-connected machine)
docker build -t k13d:latest .
docker save k13d:latest | gzip > k13d.tar.gz

# 2. Transfer to air-gapped environment and load
docker load < k13d.tar.gz
# Tag for your registry if needed
docker tag k13d:latest your-registry/k13d:latest
docker push your-registry/k13d:latest

# 3. Update image in manifest (if using private registry)
sed -i 's|youngjukim/k13d:latest|your-registry/k13d:latest|g' kubernetes/single-pod.yaml

# 4. Deploy
kubectl apply -f kubernetes/single-pod.yaml

# 5. Access
kubectl port-forward -n k13d pod/k13d 8080:8080
# Open http://localhost:8080
```

### Option 2: Dashboard with AI (Ollama)

```bash
# 1. Prepare images
docker build -t k13d:latest .
docker pull ollama/ollama:latest
docker save k13d:latest ollama/ollama:latest | gzip > k13d-bundle.tar.gz

# 2. (Optional) Pre-download Ollama model
docker run -v ollama-models:/root/.ollama ollama/ollama pull llama3.2
# Export model data
docker run --rm -v ollama-models:/data -v $(pwd):/backup alpine tar cvf /backup/models.tar -C /data .

# 3. Transfer and load in air-gapped environment
docker load < k13d-bundle.tar.gz

# 4. Deploy
kubectl apply -f kubernetes/single-pod-with-ollama.yaml

# 5. Wait for Ollama to download model (if internet available) or load from PVC
kubectl logs -n k13d k13d -c ollama -f

# 6. Access
kubectl port-forward -n k13d pod/k13d 8080:8080
```

## Access Methods

### Port Forward (Development)
```bash
kubectl port-forward -n k13d pod/k13d 8080:8080
# Access: http://localhost:8080
```

### NodePort (Direct Node Access)
```bash
# Get NodePort
kubectl get svc -n k13d k13d-nodeport -o jsonpath='{.spec.ports[0].nodePort}'
# Access: http://<any-node-ip>:30080
```

### Ingress (Production)
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: k13d
  namespace: k13d
spec:
  rules:
    - host: k13d.your-domain.com
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

## Authentication

### Token Mode (Default, Recommended)
Uses Kubernetes ServiceAccount token. No additional configuration needed.

### Local Mode (Username/Password)
```bash
# Create secret with credentials
kubectl create secret generic k13d-credentials \
  -n k13d \
  --from-literal=username=admin \
  --from-literal=password=your-secure-password

# Update Pod env
# K13D_AUTH_MODE=local
# K13D_USERNAME from secret
# K13D_PASSWORD from secret
```

## Resource Requirements

| Component | CPU Request | Memory Request | CPU Limit | Memory Limit |
|-----------|-------------|----------------|-----------|--------------|
| k13d | 50m | 64Mi | 500m | 256Mi |
| Ollama | 500m | 2Gi | 4 | 8Gi |

**Note**: Ollama requires significant memory. For GPU acceleration, add:
```yaml
resources:
  limits:
    nvidia.com/gpu: 1
```

## Troubleshooting

### Check Pod Status
```bash
kubectl get pods -n k13d
kubectl describe pod -n k13d k13d
```

### View Logs
```bash
# k13d logs
kubectl logs -n k13d k13d -c k13d

# Ollama logs (if using with-ollama variant)
kubectl logs -n k13d k13d -c ollama
```

### Health Check
```bash
kubectl exec -n k13d k13d -c k13d -- curl -s http://localhost:8080/api/health
```

### RBAC Issues
```bash
# Check if ServiceAccount has proper permissions
kubectl auth can-i list pods --as=system:serviceaccount:k13d:k13d -A
```

## Cleanup

```bash
kubectl delete -f kubernetes/single-pod.yaml
# or
kubectl delete namespace k13d
```
