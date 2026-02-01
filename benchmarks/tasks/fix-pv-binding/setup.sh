#!/bin/bash
# Setup script for fix-pv-binding task
# Creates a PV and a mismatched PVC

set -e

echo "Setting up fix-pv-binding task..."

# Cleanup any existing resources
kubectl delete pvc app-data --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete pv task-pv-volume --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create a PV with specific labels
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolume
metadata:
  name: task-pv-volume
  labels:
    type: local
    app: storage
spec:
  storageClassName: manual
  capacity:
    storage: 5Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: "/tmp/task-pv"
EOF

# Create a PVC with mismatched storage class (will stay pending)
cat <<EOF | kubectl apply --namespace="${NAMESPACE}" -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: app-data
spec:
  storageClassName: standard
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 2Gi
EOF

sleep 2

echo "Setup complete. PVC 'app-data' should be in Pending state."
