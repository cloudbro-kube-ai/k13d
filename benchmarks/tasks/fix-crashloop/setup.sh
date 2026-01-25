#!/bin/bash
# Setup script for fix-crashloop task
# Creates a deployment with an invalid command that causes CrashLoopBackOff

set -e

echo "Setting up fix-crashloop task..."

# Clean up any existing resources
kubectl delete deployment broken-app --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
sleep 2

# Create deployment with broken command
cat <<EOF | kubectl apply --namespace="${NAMESPACE}" -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: broken-app
  labels:
    app: broken-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: broken-app
  template:
    metadata:
      labels:
        app: broken-app
    spec:
      containers:
      - name: app
        image: nginx:1.25
        command: ["/bin/sh"]
        args: ["-c", "nonexistent-command"]
        ports:
        - containerPort: 80
EOF

# Wait for pods to start crashing
echo "Waiting for pods to enter CrashLoopBackOff..."
sleep 15

echo "Setup complete. Deployment 'broken-app' is now in CrashLoopBackOff state."
