#!/bin/bash
set -e

NAMESPACE="rollback-demo"

echo "Setting up rollback-deployment task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create v1 deployment
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: webapp
  annotations:
    kubernetes.io/change-cause: "Initial deployment v1"
spec:
  replicas: 2
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: webapp
  template:
    metadata:
      labels:
        app: webapp
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
EOF

echo "Waiting for v1 deployment..."
kubectl wait --for=condition=Available deployment/webapp -n $NAMESPACE --timeout=60s

# Update to v2
kubectl set image deployment/webapp web=nginx:1.25-alpine -n $NAMESPACE
kubectl annotate deployment/webapp kubernetes.io/change-cause="Update to v2" -n $NAMESPACE --overwrite

echo "Waiting for v2 deployment..."
kubectl wait --for=condition=Available deployment/webapp -n $NAMESPACE --timeout=60s

# Update to broken v3
kubectl set image deployment/webapp web=nginx:invalid-tag-does-not-exist -n $NAMESPACE
kubectl annotate deployment/webapp kubernetes.io/change-cause="Update to v3 (BROKEN)" -n $NAMESPACE --overwrite

echo "Waiting for broken deployment to fail..."
sleep 15

echo "Setup complete. The deployment is failing with a bad image. Roll it back!"
