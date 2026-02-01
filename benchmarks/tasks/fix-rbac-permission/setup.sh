#!/bin/bash
# Setup script for fix-rbac-permission task

set -e

echo "Setting up fix-rbac-permission task..."

# Clean up any existing resources
kubectl delete pod api-client --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete serviceaccount api-client-sa --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete role api-client-role --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete rolebinding api-client-binding --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
sleep 2

# Create ServiceAccount (without any permissions)
cat <<EOF | kubectl apply --namespace="${NAMESPACE}" -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: api-client-sa
EOF

# Create pod that tries to access API
cat <<EOF | kubectl apply --namespace="${NAMESPACE}" -f -
apiVersion: v1
kind: Pod
metadata:
  name: api-client
  labels:
    app: api-client
spec:
  serviceAccountName: api-client-sa
  containers:
  - name: client
    image: bitnami/kubectl:latest
    command: ["sleep", "infinity"]
EOF

echo "Waiting for pod to be ready..."
kubectl wait --for=condition=Ready pod/api-client --namespace="${NAMESPACE}" --timeout=60s

echo "Setup complete. Pod 'api-client' is running but cannot access the API."
