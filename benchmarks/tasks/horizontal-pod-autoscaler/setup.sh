#!/bin/bash
set -e

echo "Setting up horizontal-pod-autoscaler task..."

kubectl create namespace autoscale-ns --dry-run=client -o yaml | kubectl apply -f -

cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
  namespace: autoscale-ns
spec:
  replicas: 1
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
        image: nginx:1.25
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 200m
            memory: 256Mi
EOF

kubectl rollout status deployment/web-app -n autoscale-ns --timeout=60s

echo "Setup complete. Deployment created without HPA."
