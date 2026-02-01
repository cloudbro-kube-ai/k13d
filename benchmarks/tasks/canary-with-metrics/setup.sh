#!/bin/bash
set -e

NAMESPACE="canary-metrics"

echo "Setting up canary-with-metrics task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create stable deployment
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-stable
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
      version: v1
  template:
    metadata:
      labels:
        app: myapp
        version: v1
    spec:
      containers:
      - name: app
        image: nginx:1.24-alpine
        ports:
        - containerPort: 80
        resources:
          requests:
            memory: "32Mi"
            cpu: "25m"
---
apiVersion: v1
kind: Service
metadata:
  name: app-service
spec:
  selector:
    app: myapp
    version: v1
  ports:
  - port: 80
    targetPort: 80
EOF

echo "Waiting for stable deployment..."
kubectl wait --for=condition=Available deployment/app-stable -n $NAMESPACE --timeout=60s || true

echo "Setup complete. Implement canary deployment with traffic splitting."
