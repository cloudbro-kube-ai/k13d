#!/bin/bash
set -e

NAMESPACE="leader-election"

echo "Setting up leader-election task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create initial deployment without leader election
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
spec:
  replicas: 1
  selector:
    matchLabels:
      app: controller-manager
  template:
    metadata:
      labels:
        app: controller-manager
    spec:
      containers:
      - name: controller
        image: busybox
        command: ["sleep", "infinity"]
        resources:
          requests:
            memory: "32Mi"
            cpu: "25m"
EOF

echo "Waiting for deployment..."
kubectl wait --for=condition=Available deployment/controller-manager -n $NAMESPACE --timeout=60s || true

echo "Setup complete. Configure leader election for the controller-manager."
