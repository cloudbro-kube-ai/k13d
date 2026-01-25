#!/bin/bash
# Setup script for create-service task

set -e

echo "Setting up create-service task..."

# Clean up any existing resources
kubectl delete deployment backend-app --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete service backend-service --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
sleep 2

# Create deployment
cat <<EOF | kubectl apply --namespace="${NAMESPACE}" -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend-app
  labels:
    app: backend-app
spec:
  replicas: 2
  selector:
    matchLabels:
      app: backend-app
  template:
    metadata:
      labels:
        app: backend-app
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

# Wait for deployment
echo "Waiting for deployment..."
kubectl rollout status deployment/backend-app --namespace="${NAMESPACE}" --timeout=60s

echo "Setup complete. Deployment 'backend-app' is ready."
