#!/bin/bash
# Setup script for scale-deployment task

set -e

echo "Setting up scale-deployment task..."

# Clean up any existing resources
kubectl delete deployment web-app --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
sleep 2

# Create deployment with 1 replica
cat <<EOF | kubectl apply --namespace="${NAMESPACE}" -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
  labels:
    app: web-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: web-app
  template:
    metadata:
      labels:
        app: web-app
    spec:
      containers:
      - name: nginx
        image: nginx:1.25
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: 10m
            memory: 32Mi
          limits:
            cpu: 50m
            memory: 64Mi
EOF

# Wait for initial deployment
echo "Waiting for initial deployment..."
kubectl rollout status deployment/web-app --namespace="${NAMESPACE}" --timeout=60s

echo "Setup complete. Deployment 'web-app' is ready with 1 replica."
