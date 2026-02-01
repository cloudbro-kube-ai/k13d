#!/bin/bash
set -e

echo "Setting up events-filter task..."

kubectl create namespace events-test --dry-run=client -o yaml | kubectl apply -f -

# Create a running pod
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: healthy-pod
  namespace: events-test
spec:
  containers:
  - name: nginx
    image: nginx:1.25
EOF

# Create a pod that will generate warning events (image pull error)
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: failing-pod
  namespace: events-test
spec:
  containers:
  - name: bad-image
    image: nonexistent-image:v999
    imagePullPolicy: Always
EOF

# Create a pod with resource issues (will be pending)
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: resource-issue-pod
  namespace: events-test
spec:
  containers:
  - name: huge-resources
    image: nginx:1.25
    resources:
      requests:
        memory: "100Gi"
        cpu: "100"
EOF

echo "Waiting for events to be generated..."
sleep 30

echo "Setup complete. Various events have been generated."
