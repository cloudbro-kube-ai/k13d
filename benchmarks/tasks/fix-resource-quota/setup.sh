#!/bin/bash
# Setup script for fix-resource-quota task
# Creates a tight resource quota and a deployment that exceeds it

set -e

echo "Setting up fix-resource-quota task..."

# Cleanup any existing resources
kubectl delete deployment web-app --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete resourcequota compute-quota --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create a restrictive ResourceQuota
cat <<EOF | kubectl apply --namespace="${NAMESPACE}" -f -
apiVersion: v1
kind: ResourceQuota
metadata:
  name: compute-quota
spec:
  hard:
    requests.cpu: "500m"
    requests.memory: "512Mi"
    limits.cpu: "1"
    limits.memory: "1Gi"
    pods: "2"
EOF

# Create a deployment that requests more than the quota allows
cat <<EOF | kubectl apply --namespace="${NAMESPACE}" -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
spec:
  replicas: 3
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
          resources:
            requests:
              cpu: "200m"
              memory: "256Mi"
            limits:
              cpu: "400m"
              memory: "512Mi"
EOF

sleep 3

echo "Setup complete. Deployment 'web-app' should have quota issues."
echo "Check: kubectl describe deployment web-app -n ${NAMESPACE}"
