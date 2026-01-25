#!/bin/bash
# Setup script for fix-image-pull task
# Creates a deployment with a non-existent image

set -e

echo "Setting up fix-image-pull task..."

# Clean up any existing resources
kubectl delete deployment image-app --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
sleep 2

# Create deployment with invalid image
cat <<EOF | kubectl apply --namespace="${NAMESPACE}" -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: image-app
  labels:
    app: image-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: image-app
  template:
    metadata:
      labels:
        app: image-app
    spec:
      containers:
      - name: app
        image: nginx:nonexistent-tag-12345
        ports:
        - containerPort: 80
EOF

# Wait for pods to start failing
echo "Waiting for pods to enter ImagePullBackOff..."
sleep 15

echo "Setup complete. Deployment 'image-app' has pods with ImagePullBackOff."
