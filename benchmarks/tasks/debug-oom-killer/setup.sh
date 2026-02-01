#!/bin/bash
set -e

NAMESPACE="memory-issues"

echo "Setting up debug-oom-killer task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create deployment with insufficient memory (will OOMKill)
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: memory-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: memory-app
  template:
    metadata:
      labels:
        app: memory-app
    spec:
      containers:
      - name: memory-hog
        image: python:alpine
        command: ["/bin/sh", "-c"]
        args:
          - |
            python3 -c "
            import time
            # Allocate ~200MB of memory
            data = []
            for i in range(200):
                data.append('x' * 1024 * 1024)  # 1MB strings
                time.sleep(0.1)
            print('Memory allocated, sleeping...')
            while True:
                time.sleep(60)
            "
        resources:
          requests:
            memory: "32Mi"
            cpu: "50m"
          limits:
            memory: "64Mi"
            cpu: "100m"
EOF

echo "Waiting for deployment to attempt startup (will OOMKill)..."
sleep 15

# Wait for OOMKill to occur
for i in {1..20}; do
    OOM_COUNT=$(kubectl get pods -n $NAMESPACE -l app=memory-app -o jsonpath='{.items[0].status.containerStatuses[0].restartCount}' 2>/dev/null || echo "0")
    if [[ "$OOM_COUNT" -gt 0 ]]; then
        echo "OOMKill detected after $OOM_COUNT restarts."
        break
    fi
    sleep 3
done

echo "Setup complete. The deployment has OOM issues - diagnose and fix them."
