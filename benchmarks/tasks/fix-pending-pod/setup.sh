#!/bin/bash
set -e

echo "Setting up fix-pending-pod task..."

# Create namespace
kubectl create namespace homepage-ns --dry-run=client -o yaml | kubectl apply -f -

# Create a pod requesting more resources than available (will stay Pending)
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: homepage-pod
  namespace: homepage-ns
spec:
  containers:
  - name: nginx
    image: nginx:1.25
    resources:
      requests:
        cpu: "100"
        memory: "1000Gi"
EOF

echo "Setup complete. Pod should be in Pending state."
kubectl get pod -n homepage-ns homepage-pod || true
