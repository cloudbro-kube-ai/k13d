#!/bin/bash
set -e

NAMESPACE="production"

echo "Setting up deny-all-policy task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create pods for each tier
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: v1
kind: Pod
metadata:
  name: web-server
  labels:
    tier: web
    app: frontend
spec:
  containers:
  - name: nginx
    image: nginx:alpine
---
apiVersion: v1
kind: Pod
metadata:
  name: app-server
  labels:
    tier: app
    app: backend
spec:
  containers:
  - name: app
    image: python:alpine
    command: ["sleep", "infinity"]
    ports:
    - containerPort: 8080
---
apiVersion: v1
kind: Pod
metadata:
  name: cache-server
  labels:
    tier: cache
    app: redis
spec:
  containers:
  - name: redis
    image: redis:alpine
    ports:
    - containerPort: 6379
EOF

echo "Waiting for pods to be ready..."
kubectl wait --for=condition=Ready pod --all -n $NAMESPACE --timeout=60s || true

echo "Setup complete. Implement zero-trust networking with default deny policies."
