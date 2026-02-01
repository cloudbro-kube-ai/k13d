#!/bin/bash
set -e

NAMESPACE="graceful-demo"

echo "Setting up graceful-shutdown task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create deployments without graceful shutdown configuration
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-server
spec:
  replicas: 2
  selector:
    matchLabels:
      app: web-server
  template:
    metadata:
      labels:
        app: web-server
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        ports:
        - containerPort: 80
        resources:
          requests:
            memory: "32Mi"
            cpu: "25m"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-server
spec:
  replicas: 2
  selector:
    matchLabels:
      app: api-server
  template:
    metadata:
      labels:
        app: api-server
    spec:
      containers:
      - name: api
        image: nginx:alpine
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: "32Mi"
            cpu: "25m"
EOF

echo "Waiting for deployments..."
kubectl wait --for=condition=Available deployment/web-server -n $NAMESPACE --timeout=60s || true
kubectl wait --for=condition=Available deployment/api-server -n $NAMESPACE --timeout=60s || true

echo "Setup complete. Configure graceful shutdown for both services."
