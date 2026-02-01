#!/bin/bash
set -e

NAMESPACE="external-access"

echo "Setting up cidr-policy task..."

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
  name: api-gateway
  labels:
    app: api-gateway
    role: gateway
spec:
  containers:
  - name: gateway
    image: nginx:alpine
    ports:
    - containerPort: 8080
---
apiVersion: v1
kind: Pod
metadata:
  name: worker
  labels:
    app: worker
    role: worker
spec:
  containers:
  - name: worker
    image: busybox
    command: ["sleep", "infinity"]
EOF

echo "Waiting for pods to be ready..."
kubectl wait --for=condition=Ready pod --all -n $NAMESPACE --timeout=60s || true

echo "Setup complete. Create CIDR-based network policies."
