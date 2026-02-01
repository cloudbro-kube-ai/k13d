#!/bin/bash
set -e

echo "Setting up exec-into-pod task..."

kubectl create namespace exec-test --dry-run=client -o yaml | kubectl apply -f -

# Create a pod with nginx
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: debug-target
  namespace: exec-test
spec:
  containers:
  - name: nginx
    image: nginx:1.25
    ports:
    - containerPort: 80
EOF

echo "Waiting for pod to be ready..."
kubectl wait --for=condition=Ready pod/debug-target -n exec-test --timeout=120s

echo "Setup complete. Pod is ready for debugging."
