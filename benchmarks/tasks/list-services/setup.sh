#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create multiple Services
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: frontend-svc
  namespace: benchmark
spec:
  selector:
    app: frontend
  ports:
  - port: 80
---
apiVersion: v1
kind: Service
metadata:
  name: backend-svc
  namespace: benchmark
spec:
  selector:
    app: backend
  ports:
  - port: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: database-svc
  namespace: benchmark
spec:
  selector:
    app: database
  ports:
  - port: 5432
EOF

# Remove any existing output file
rm -f /tmp/services.txt

echo "Setup complete: Services created in namespace 'benchmark'"
