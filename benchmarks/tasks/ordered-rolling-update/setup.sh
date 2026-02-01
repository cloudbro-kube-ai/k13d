#!/bin/bash
set -e

NAMESPACE="ordered-update"

echo "Setting up ordered-rolling-update task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create StatefulSet
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: v1
kind: Service
metadata:
  name: app-cluster
spec:
  clusterIP: None
  selector:
    app: app-cluster
  ports:
  - port: 80
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: app-cluster
spec:
  serviceName: app-cluster
  replicas: 5
  selector:
    matchLabels:
      app: app-cluster
  updateStrategy:
    type: OnDelete
  template:
    metadata:
      labels:
        app: app-cluster
    spec:
      containers:
      - name: nginx
        image: nginx:1.24-alpine
        ports:
        - containerPort: 80
        resources:
          requests:
            memory: "32Mi"
            cpu: "25m"
EOF

echo "Waiting for StatefulSet pods..."
for i in {0..4}; do
    kubectl wait --for=condition=Ready pod app-cluster-$i -n $NAMESPACE --timeout=120s || true
done

echo "Setup complete. Implement ordered rolling update using partitions."
