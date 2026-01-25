#!/bin/bash
set -e

echo "Setting up rolling-update task..."

kubectl create namespace rolling --dry-run=client -o yaml | kubectl apply -f -

cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
  namespace: rolling
spec:
  replicas: 3
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
      - name: nginx
        image: nginx:1.14
        ports:
        - containerPort: 80
EOF

kubectl rollout status deployment/web-app -n rolling --timeout=60s

echo "Setup complete. Deployment using nginx:1.14"
kubectl get deployment web-app -n rolling -o jsonpath='{.spec.template.spec.containers[0].image}'
echo ""
