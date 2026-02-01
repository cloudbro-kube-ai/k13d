#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create the Service to be deleted
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: old-service
  namespace: benchmark
spec:
  selector:
    app: old
  ports:
  - port: 80
    targetPort: 80
EOF

# Create another Service that should NOT be deleted
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: keep-service
  namespace: benchmark
spec:
  selector:
    app: keep
  ports:
  - port: 80
    targetPort: 80
EOF

echo "Setup complete: Services 'old-service' and 'keep-service' created"
