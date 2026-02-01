#!/bin/bash
set -e

NAMESPACE="affinity-ha"

echo "Setting up anti-affinity-ha task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create workloads without anti-affinity (to be fixed)
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: critical-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: critical-api
  template:
    metadata:
      labels:
        app: critical-api
        tier: api
    spec:
      containers:
      - name: api
        image: nginx:alpine
        resources:
          requests:
            memory: "32Mi"
            cpu: "25m"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: worker
spec:
  replicas: 4
  selector:
    matchLabels:
      app: worker
  template:
    metadata:
      labels:
        app: worker
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
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: cache
spec:
  serviceName: cache
  replicas: 3
  selector:
    matchLabels:
      app: cache
  template:
    metadata:
      labels:
        app: cache
        tier: cache
    spec:
      containers:
      - name: redis
        image: redis:alpine
        resources:
          requests:
            memory: "32Mi"
            cpu: "25m"
EOF

echo "Waiting for initial deployments..."
sleep 10

echo "Setup complete. Add anti-affinity rules to spread pods across nodes."
