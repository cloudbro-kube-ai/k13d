#!/bin/bash
set -e

NAMESPACE="microservices"

echo "Setting up ingress-policy task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace external-service --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create pods
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: v1
kind: Pod
metadata:
  name: api-server
  labels:
    app: api
    tier: backend
spec:
  containers:
  - name: api
    image: nginx:alpine
    ports:
    - containerPort: 8080
    command: ["/bin/sh", "-c", "nginx -g 'daemon off;'"]
---
apiVersion: v1
kind: Pod
metadata:
  name: web-frontend
  labels:
    app: web
    tier: frontend
spec:
  containers:
  - name: web
    image: nginx:alpine
---
apiVersion: v1
kind: Pod
metadata:
  name: database
  labels:
    app: db
    tier: data
spec:
  containers:
  - name: db
    image: redis:alpine
    ports:
    - containerPort: 5432
EOF

echo "Waiting for pods to be ready..."
kubectl wait --for=condition=Ready pod -l app=api -n $NAMESPACE --timeout=60s || true
kubectl wait --for=condition=Ready pod -l app=web -n $NAMESPACE --timeout=60s || true
kubectl wait --for=condition=Ready pod -l app=db -n $NAMESPACE --timeout=60s || true

echo "Setup complete. Create the ingress network policies."
