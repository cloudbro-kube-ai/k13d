#!/bin/bash
set -e

NAMESPACE="eviction-demo"

echo "Setting up debug-evicted-pods task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true
kubectl delete priorityclass critical-priority low-priority --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create deployments without priority classes
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: critical-app
spec:
  replicas: 2
  selector:
    matchLabels:
      app: critical-app
  template:
    metadata:
      labels:
        app: critical-app
        tier: critical
    spec:
      containers:
      - name: app
        image: nginx:alpine
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "256Mi"
            cpu: "200m"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: background-job
spec:
  replicas: 3
  selector:
    matchLabels:
      app: background-job
  template:
    metadata:
      labels:
        app: background-job
        tier: background
    spec:
      containers:
      - name: worker
        image: busybox
        command: ["sleep", "infinity"]
        resources:
          requests:
            memory: "32Mi"
            cpu: "25m"
EOF

echo "Waiting for deployments..."
kubectl wait --for=condition=Available deployment/critical-app -n $NAMESPACE --timeout=60s || true
kubectl wait --for=condition=Available deployment/background-job -n $NAMESPACE --timeout=60s || true

echo "Setup complete. Configure priority classes to prevent critical pod evictions."
