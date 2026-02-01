#!/bin/bash
set -e

NAMESPACE="pause-demo"

echo "Setting up deployment-pause-resume task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create deployment
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: web-app
  template:
    metadata:
      labels:
        app: web-app
        version: "1.0"
    spec:
      containers:
      - name: web
        image: nginx:1.24-alpine
        ports:
        - containerPort: 80
        resources:
          requests:
            memory: "32Mi"
            cpu: "25m"
          limits:
            memory: "64Mi"
            cpu: "100m"
EOF

echo "Waiting for deployment..."
kubectl wait --for=condition=Available deployment/web-app -n $NAMESPACE --timeout=60s

echo "Setup complete. Pause the deployment, make batched changes, then resume."
