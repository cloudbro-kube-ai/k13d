#!/bin/bash
set -e

echo "Setting up previous-container-logs task..."

kubectl create namespace prev-logs-test --dry-run=client -o yaml | kubectl apply -f -

# Create a pod that logs and then exits (will be restarted by K8s)
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: crash-loop-pod
  namespace: prev-logs-test
spec:
  restartPolicy: Always
  containers:
  - name: crasher
    image: busybox:1.36
    command: ["/bin/sh", "-c"]
    args:
    - |
      echo "[STARTUP] Container starting at $(date)"
      echo "[INFO] Processing initialization..."
      echo "[ERROR] Critical error occurred - simulated crash"
      echo "[FATAL] Container will now exit"
      exit 1
EOF

echo "Waiting for pod to crash and restart..."
sleep 30

# Check restart count
RESTARTS=$(kubectl get pod crash-loop-pod -n prev-logs-test -o jsonpath='{.status.containerStatuses[0].restartCount}' 2>/dev/null || echo "0")
echo "Pod has restarted $RESTARTS times"

echo "Setup complete. Pod has crashed and restarted."
