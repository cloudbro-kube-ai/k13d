#!/bin/bash
set -e

echo "Setting up fix-service-routing task..."

kubectl create namespace web --dry-run=client -o yaml | kubectl apply -f -

# Create deployment with label 'app=nginx'
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: web
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.25
        ports:
        - containerPort: 80
EOF

# Create service with WRONG selector (app=web instead of app=nginx)
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
  namespace: web
spec:
  selector:
    app: web
  ports:
  - port: 80
    targetPort: 80
EOF

kubectl rollout status deployment/nginx-deployment -n web --timeout=60s

echo "Setup complete. Service selector mismatch - no endpoints."
kubectl get endpoints nginx-service -n web || true
