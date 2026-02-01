#!/bin/bash
set -e

NAMESPACE="zone-spread"

echo "Setting up zone-spread-ha task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create workloads without topology spread (to be fixed)
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
spec:
  replicas: 6
  selector:
    matchLabels:
      app: web-app
  template:
    metadata:
      labels:
        app: web-app
        tier: frontend
    spec:
      containers:
      - name: web
        image: nginx:alpine
        resources:
          requests:
            memory: "32Mi"
            cpu: "25m"
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: database
spec:
  serviceName: database
  replicas: 3
  selector:
    matchLabels:
      app: database
  template:
    metadata:
      labels:
        app: database
        tier: data
    spec:
      containers:
      - name: db
        image: redis:alpine
        resources:
          requests:
            memory: "32Mi"
            cpu: "25m"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: background
spec:
  replicas: 4
  selector:
    matchLabels:
      app: background
  template:
    metadata:
      labels:
        app: background
        tier: worker
    spec:
      containers:
      - name: worker
        image: busybox
        command: ["sleep", "infinity"]
        resources:
          requests:
            memory: "16Mi"
            cpu: "10m"
EOF

echo "Waiting for deployments..."
sleep 10

echo "Setup complete. Add topology spread constraints for zone distribution."
