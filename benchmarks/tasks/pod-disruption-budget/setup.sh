#!/bin/bash
set -e

NAMESPACE="pdb-demo"

echo "Setting up pod-disruption-budget task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create workloads
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-frontend
spec:
  replicas: 3
  selector:
    matchLabels:
      app: web-frontend
  template:
    metadata:
      labels:
        app: web-frontend
        tier: frontend
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-backend
spec:
  replicas: 5
  selector:
    matchLabels:
      app: api-backend
  template:
    metadata:
      labels:
        app: api-backend
        tier: backend
    spec:
      containers:
      - name: api
        image: nginx:alpine
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: cache
spec:
  serviceName: cache
  replicas: 3
  selector:
    matchLabels:
      app: cache
  template:
    metadata:
      labels:
        app: cache
        tier: cache
    spec:
      containers:
      - name: redis
        image: redis:alpine
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: batch-worker
spec:
  replicas: 4
  selector:
    matchLabels:
      app: batch-worker
  template:
    metadata:
      labels:
        app: batch-worker
        tier: worker
    spec:
      containers:
      - name: worker
        image: busybox
        command: ["sleep", "infinity"]
EOF

echo "Waiting for deployments..."
kubectl wait --for=condition=Available deployment/web-frontend -n $NAMESPACE --timeout=60s || true
kubectl wait --for=condition=Available deployment/api-backend -n $NAMESPACE --timeout=60s || true
kubectl wait --for=condition=Available deployment/batch-worker -n $NAMESPACE --timeout=60s || true

echo "Setup complete. Create Pod Disruption Budgets for each workload."
