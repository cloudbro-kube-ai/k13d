#!/bin/bash
# Setup script for blue-green-switch task

set -e

echo "Setting up blue-green-switch task..."

# Create namespace if not exists
kubectl create namespace deploy-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing resources
kubectl delete deployment app-blue app-green --namespace=deploy-test --ignore-not-found=true 2>/dev/null || true
kubectl delete service app-service --namespace=deploy-test --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create blue deployment
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-blue
  namespace: deploy-test
spec:
  replicas: 2
  selector:
    matchLabels:
      app: myapp
      version: blue
  template:
    metadata:
      labels:
        app: myapp
        version: blue
    spec:
      containers:
      - name: nginx
        image: nginx:1.24
        ports:
        - containerPort: 80
EOF

# Create green deployment
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-green
  namespace: deploy-test
spec:
  replicas: 2
  selector:
    matchLabels:
      app: myapp
      version: green
  template:
    metadata:
      labels:
        app: myapp
        version: green
    spec:
      containers:
      - name: nginx
        image: nginx:1.25
        ports:
        - containerPort: 80
EOF

# Create service pointing to blue
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: app-service
  namespace: deploy-test
spec:
  selector:
    app: myapp
    version: blue
  ports:
  - port: 80
    targetPort: 80
EOF

# Wait for deployments to be ready
kubectl rollout status deployment/app-blue --namespace=deploy-test --timeout=60s || true
kubectl rollout status deployment/app-green --namespace=deploy-test --timeout=60s || true

echo "Setup complete. Service 'app-service' currently points to blue deployment."
