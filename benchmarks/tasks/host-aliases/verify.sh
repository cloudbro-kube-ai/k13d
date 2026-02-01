#!/bin/bash
set -e

echo "Verifying host-aliases task..."

NAMESPACE="host-alias-test"

# Check Pod exists
if ! kubectl get pod host-alias-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'host-alias-pod' not found"
    exit 1
fi

# Check hostAliases contain expected entries
HOST_ALIASES=$(kubectl get pod host-alias-pod -n "$NAMESPACE" -o jsonpath='{.spec.hostAliases}' 2>/dev/null || echo "")
if [[ "$HOST_ALIASES" != *"10.0.0.1"* ]]; then
    echo "FAILED: hostAliases should contain IP '10.0.0.1'"
    exit 1
fi
if [[ "$HOST_ALIASES" != *"db.local"* ]]; then
    echo "FAILED: hostAliases should contain hostname 'db.local'"
    exit 1
fi
if [[ "$HOST_ALIASES" != *"10.0.0.2"* ]]; then
    echo "FAILED: hostAliases should contain IP '10.0.0.2'"
    exit 1
fi
if [[ "$HOST_ALIASES" != *"cache.local"* ]]; then
    echo "FAILED: hostAliases should contain hostname 'cache.local'"
    exit 1
fi
if [[ "$HOST_ALIASES" != *"10.0.0.3"* ]]; then
    echo "FAILED: hostAliases should contain IP '10.0.0.3'"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/host-alias-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

# Verify entries are in /etc/hosts
HOSTS_CONTENT=$(kubectl exec host-alias-pod -n "$NAMESPACE" -- cat /etc/hosts 2>/dev/null || echo "")
if [[ "$HOSTS_CONTENT" != *"db.local"* ]]; then
    echo "WARNING: db.local not found in /etc/hosts"
fi
if [[ "$HOSTS_CONTENT" != *"cache.local"* ]]; then
    echo "WARNING: cache.local not found in /etc/hosts"
fi

echo "SUCCESS: Host aliases correctly configured"
exit 0
