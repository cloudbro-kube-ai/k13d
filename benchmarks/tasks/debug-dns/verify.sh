#!/bin/bash
# Verifier script for debug-dns task

set -e

echo "Verifying debug-dns task..."

# Check if pod exists and is running
if ! kubectl get pod dns-test --namespace="${NAMESPACE}" &>/dev/null; then
    echo "ERROR: Pod 'dns-test' not found"
    exit 1
fi

STATUS=$(kubectl get pod dns-test --namespace="${NAMESPACE}" -o jsonpath='{.status.phase}')
if [ "$STATUS" != "Running" ]; then
    echo "ERROR: Pod is not running. Current status: $STATUS"
    exit 1
fi

# Test DNS resolution from the pod
echo "Testing DNS resolution..."
DNS_RESULT=$(kubectl exec dns-test --namespace="${NAMESPACE}" -- nslookup kubernetes.default 2>&1 || true)

if echo "$DNS_RESULT" | grep -q "Address"; then
    echo "Verification PASSED: DNS resolution is working"
    echo "DNS lookup result: $DNS_RESULT"
    exit 0
else
    echo "ERROR: DNS resolution is still not working"
    echo "DNS lookup result: $DNS_RESULT"
    exit 1
fi
