#!/bin/bash
# Setup script for create-ingress task

set -e

echo "Setting up create-ingress task..."

# Cleanup any existing ingress
kubectl delete ingress app-ingress --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

# Create backend services that the ingress will route to
cat <<EOF | kubectl apply --namespace="${NAMESPACE}" -f -
apiVersion: v1
kind: Service
metadata:
  name: api-svc
spec:
  selector:
    app: api
  ports:
    - port: 8080
      targetPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: web-svc
spec:
  selector:
    app: web
  ports:
    - port: 80
      targetPort: 80
EOF

sleep 2

echo "Setup complete. Backend services 'api-svc' and 'web-svc' are ready."
