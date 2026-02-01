#!/bin/bash
set -e

echo "Setting up copy-files-to-pod task..."

kubectl create namespace cp-test --dry-run=client -o yaml | kubectl apply -f -

# Create a pod with tar installed (needed for kubectl cp)
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: file-server
  namespace: cp-test
spec:
  containers:
  - name: alpine
    image: alpine:3.19
    command: ["/bin/sh", "-c"]
    args:
    - |
      apk add --no-cache tar
      echo "Pod ready for file operations"
      sleep 3600
EOF

# Create a test file locally
TMPDIR=$(mktemp -d)
echo "This is a test file created at $(date)" > "$TMPDIR/test-file.txt"
echo "Test file location: $TMPDIR/test-file.txt"

echo "Waiting for pod to be ready..."
kubectl wait --for=condition=Ready pod/file-server -n cp-test --timeout=120s

# Wait for tar to be installed
sleep 5

echo "Setup complete. Pod is ready."
echo "Test file created at: $TMPDIR/test-file.txt"
