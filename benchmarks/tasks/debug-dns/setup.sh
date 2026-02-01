#!/bin/bash
# Setup script for debug-dns task
# Creates a pod with broken DNS configuration

set -e

echo "Setting up debug-dns task..."

# Cleanup any existing resources
kubectl delete pod dns-test --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete networkpolicy deny-dns --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create a network policy that blocks DNS traffic (port 53)
cat <<EOF | kubectl apply --namespace="${NAMESPACE}" -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-dns
spec:
  podSelector:
    matchLabels:
      app: dns-test
  policyTypes:
    - Egress
  egress:
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0
      ports:
        - protocol: TCP
          port: 80
        - protocol: TCP
          port: 443
EOF

# Create a test pod with the label that matches the network policy
cat <<EOF | kubectl apply --namespace="${NAMESPACE}" -f -
apiVersion: v1
kind: Pod
metadata:
  name: dns-test
  labels:
    app: dns-test
spec:
  containers:
    - name: dnsutils
      image: registry.k8s.io/e2e-test-images/jessie-dnsutils:1.3
      command:
        - sleep
        - "3600"
EOF

echo "Waiting for pod to be ready..."
kubectl wait --for=condition=Ready pod/dns-test --namespace="${NAMESPACE}" --timeout=60s 2>/dev/null || true

echo "Setup complete. Pod 'dns-test' should have DNS issues due to network policy."
