#!/bin/bash
set -e

NAMESPACE="pressure-demo"

echo "Setting up debug-node-pressure task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace (no limits/quotas)
kubectl create namespace $NAMESPACE

# Create resource-intensive deployment (problematic)
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: resource-hog
spec:
  replicas: 5
  selector:
    matchLabels:
      app: resource-hog
  template:
    metadata:
      labels:
        app: resource-hog
    spec:
      containers:
      - name: hog
        image: busybox
        command: ["sleep", "infinity"]
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: normal-app
spec:
  replicas: 2
  selector:
    matchLabels:
      app: normal-app
  template:
    metadata:
      labels:
        app: normal-app
    spec:
      containers:
      - name: app
        image: nginx:alpine
EOF

echo "Waiting for deployments (some pods may be pending)..."
sleep 10

echo "Setup complete. Diagnose and fix resource pressure issues."
