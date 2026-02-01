#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create a pod
kubectl run backend-pod --image=nginx:alpine -n benchmark --labels="app=backend" --restart=Never --dry-run=client -o yaml | kubectl apply -f -

# Create the Service with initial port configuration
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: backend-svc
  namespace: benchmark
spec:
  selector:
    app: backend
  ports:
  - port: 80
    targetPort: 80
EOF

echo "Setup complete: Service 'backend-svc' created with port 80"
