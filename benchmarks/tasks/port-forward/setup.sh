#!/bin/bash
set -e

echo "Setting up port-forward task..."

kubectl create namespace port-fwd-test --dry-run=client -o yaml | kubectl apply -f -

# Create nginx pod
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: web-server
  namespace: port-fwd-test
  labels:
    app: web
spec:
  containers:
  - name: nginx
    image: nginx:1.25
    ports:
    - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: web-svc
  namespace: port-fwd-test
spec:
  selector:
    app: web
  ports:
  - port: 80
    targetPort: 80
EOF

echo "Waiting for pod to be ready..."
kubectl wait --for=condition=Ready pod/web-server -n port-fwd-test --timeout=120s

echo "Setup complete. Pod and service are ready."
