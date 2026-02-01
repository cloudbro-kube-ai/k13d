#!/bin/bash
set -e

NAMESPACE="network-debug"

echo "Setting up debug-network-partition task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create deployments and broken services
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      labels:
        app: frontend
        tier: web
    spec:
      containers:
      - name: frontend
        image: nginx:alpine
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: backend-api
  template:
    metadata:
      labels:
        app: backend-api
        tier: api
    spec:
      containers:
      - name: backend
        image: nginx:alpine
        ports:
        - containerPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: database
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
        tier: data
    spec:
      containers:
      - name: db
        image: redis:alpine
        ports:
        - containerPort: 5432
---
apiVersion: v1
kind: Service
metadata:
  name: frontend
spec:
  selector:
    app: frontend
  ports:
  - port: 80
    targetPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: backend
spec:
  selector:
    app: backend  # WRONG - should be backend-api
  ports:
  - port: 8080
    targetPort: 8080
EOF
# Note: database service is intentionally missing

echo "Waiting for deployments..."
kubectl wait --for=condition=Available deployment/frontend -n $NAMESPACE --timeout=60s || true
kubectl wait --for=condition=Available deployment/backend -n $NAMESPACE --timeout=60s || true
kubectl wait --for=condition=Available deployment/database -n $NAMESPACE --timeout=60s || true

echo "Setup complete. Network connectivity is broken - diagnose and fix."
