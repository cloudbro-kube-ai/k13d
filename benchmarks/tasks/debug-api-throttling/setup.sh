#!/bin/bash
set -e

NAMESPACE="api-stress"

echo "Setting up debug-api-throttling task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create problematic deployments
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-heavy-client
spec:
  replicas: 5
  selector:
    matchLabels:
      app: api-heavy-client
  template:
    metadata:
      labels:
        app: api-heavy-client
    spec:
      containers:
      - name: client
        image: busybox
        command: ["sleep", "infinity"]
        resources:
          requests:
            memory: "32Mi"
            cpu: "25m"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: monitor-spam
spec:
  replicas: 3
  selector:
    matchLabels:
      app: monitor-spam
  template:
    metadata:
      labels:
        app: monitor-spam
    spec:
      containers:
      - name: spam
        image: busybox
        command: ["sleep", "infinity"]
        resources:
          requests:
            memory: "16Mi"
            cpu: "10m"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: legitimate-app
spec:
  replicas: 2
  selector:
    matchLabels:
      app: legitimate-app
  template:
    metadata:
      labels:
        app: legitimate-app
    spec:
      containers:
      - name: app
        image: nginx:alpine
EOF

echo "Waiting for deployments..."
kubectl wait --for=condition=Available deployment/api-heavy-client -n $NAMESPACE --timeout=60s || true
kubectl wait --for=condition=Available deployment/monitor-spam -n $NAMESPACE --timeout=60s || true
kubectl wait --for=condition=Available deployment/legitimate-app -n $NAMESPACE --timeout=60s || true

echo "Setup complete. The cluster is experiencing API throttling - diagnose and fix."
