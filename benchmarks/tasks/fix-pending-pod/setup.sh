#!/bin/bash
set -e

echo "Setting up fix-pending-pod task..."

# Create namespace
kubectl create namespace homepage-ns --dry-run=client -o yaml | kubectl apply -f -

# Create a pod with non-existent node selector (will stay Pending)
# AI should remove or fix the nodeSelector to schedule the pod
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: homepage-pod
  namespace: homepage-ns
spec:
  nodeSelector:
    non-existent-label: "true"
  containers:
  - name: nginx
    image: nginx:1.25
    resources:
      requests:
        cpu: "100m"
        memory: "128Mi"
EOF

echo "Setup complete. Pod should be in Pending state due to nodeSelector."
kubectl get pod -n homepage-ns homepage-pod || true
