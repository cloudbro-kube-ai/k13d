#!/bin/bash
# Setup script for pvc-resize task

set -e

echo "Setting up pvc-resize task..."

# Create namespace if not exists
kubectl create namespace volume-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing resources
kubectl delete pvc data-pvc --namespace=volume-test --ignore-not-found=true 2>/dev/null || true

# Wait for deletion
sleep 2

# Create the initial PVC with 1Gi
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: data-pvc
  namespace: volume-test
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
EOF

echo "Setup complete. Created PVC 'data-pvc' with 1Gi capacity."
