#!/bin/bash
# Setup script for pause-rollout task

set -e

echo "Setting up pause-rollout task..."

# Create namespace if not exists
kubectl create namespace deploy-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing deployment
kubectl delete deployment rolling-app --namespace=deploy-test --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create initial deployment
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rolling-app
  namespace: deploy-test
spec:
  replicas: 4
  selector:
    matchLabels:
      app: rolling-app
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: rolling-app
    spec:
      containers:
      - name: nginx
        image: nginx:1.24
        ports:
        - containerPort: 80
EOF

# Wait for initial deployment
kubectl rollout status deployment/rolling-app --namespace=deploy-test --timeout=60s || true

# Trigger a rolling update
kubectl set image deployment/rolling-app nginx=nginx:1.25 --namespace=deploy-test

echo "Setup complete. Deployment 'rolling-app' is rolling out an update."
