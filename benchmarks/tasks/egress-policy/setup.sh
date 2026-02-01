#!/bin/bash
set -e

NAMESPACE="secure-apps"

echo "Setting up egress-policy task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create pods
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: v1
kind: Pod
metadata:
  name: secure-worker
  labels:
    app: worker
    security: high
spec:
  containers:
  - name: worker
    image: busybox
    command: ["sleep", "infinity"]
---
apiVersion: v1
kind: Pod
metadata:
  name: api-server
  labels:
    app: api
    security: medium
spec:
  containers:
  - name: api
    image: nginx:alpine
    ports:
    - containerPort: 80
EOF

echo "Waiting for pods to be ready..."
kubectl wait --for=condition=Ready pod -l app=worker -n $NAMESPACE --timeout=60s || true
kubectl wait --for=condition=Ready pod -l app=api -n $NAMESPACE --timeout=60s || true

echo "Setup complete. Create the egress network policies."
