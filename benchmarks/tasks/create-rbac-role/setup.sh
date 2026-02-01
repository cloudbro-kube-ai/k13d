#!/bin/bash
# Setup script for create-rbac-role task

set -e

echo "Setting up create-rbac-role task..."

# Clean up any existing resources
kubectl delete serviceaccount app-reader --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete role pod-reader --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete rolebinding app-reader-binding --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

# Create a test pod for verification
cat <<EOF | kubectl apply --namespace="${NAMESPACE}" -f -
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  labels:
    app: test
spec:
  containers:
  - name: nginx
    image: nginx:1.25
EOF

echo "Setup complete. Ready for RBAC creation."
