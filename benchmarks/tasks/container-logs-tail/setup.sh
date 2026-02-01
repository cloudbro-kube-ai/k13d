#!/bin/bash
set -e

echo "Setting up container-logs-tail task..."

kubectl create namespace logs-tail-test --dry-run=client -o yaml | kubectl apply -f -

# Create a pod that generates logs
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: log-generator
  namespace: logs-tail-test
spec:
  containers:
  - name: logger
    image: busybox:1.36
    command: ["/bin/sh", "-c"]
    args:
    - |
      i=0
      while true; do
        echo "[LOG] Message number $i at $(date)"
        i=$((i+1))
        sleep 2
      done
EOF

echo "Waiting for pod to be ready..."
kubectl wait --for=condition=Ready pod/log-generator -n logs-tail-test --timeout=120s || true

# Wait a bit for logs to accumulate
sleep 10

echo "Setup complete. Pod is generating logs."
