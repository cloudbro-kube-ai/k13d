#!/bin/bash
set -e

echo "Setting up create-network-policy task..."

kubectl create namespace secure-app --dry-run=client -o yaml | kubectl apply -f -

# Create API deployment
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
  namespace: secure-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: api
  template:
    metadata:
      labels:
        app: api
    spec:
      containers:
      - name: api
        image: nginx:1.25
        ports:
        - containerPort: 8080
EOF

# Create frontend deployment
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  namespace: secure-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
      - name: frontend
        image: nginx:1.25
EOF

echo "Setup complete. No NetworkPolicy exists yet."
