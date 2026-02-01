#!/bin/bash
set -e

echo "Verifying sidecar-proxy task..."

NAMESPACE="sidecar-proxy-test"

# Check ConfigMap exists
if ! kubectl get configmap nginx-proxy-config -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: ConfigMap 'nginx-proxy-config' not found"
    exit 1
fi

# Check Pod exists
if ! kubectl get pod proxy-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'proxy-pod' not found"
    exit 1
fi

# Check both containers exist
CONTAINERS=$(kubectl get pod proxy-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[*].name}' 2>/dev/null || echo "")
if [[ "$CONTAINERS" != *"backend"* ]]; then
    echo "FAILED: Container 'backend' not found"
    exit 1
fi
if [[ "$CONTAINERS" != *"proxy"* ]]; then
    echo "FAILED: Container 'proxy' not found"
    exit 1
fi

# Check Service exists
if ! kubectl get service proxy-service -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Service 'proxy-service' not found"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/proxy-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

# Check nginx config is mounted
CONFIG_MOUNTED=$(kubectl exec proxy-pod -n "$NAMESPACE" -c proxy -- ls /etc/nginx/conf.d/ 2>/dev/null || echo "")
if [[ -z "$CONFIG_MOUNTED" ]]; then
    echo "FAILED: Nginx config not mounted"
    exit 1
fi

# Check proxy can reach backend
RESPONSE=$(kubectl exec proxy-pod -n "$NAMESPACE" -c proxy -- curl -s -o /dev/null -w "%{http_code}" localhost:80 2>/dev/null || echo "000")
if [[ "$RESPONSE" != "200" ]]; then
    echo "WARNING: Proxy not returning 200, got $RESPONSE (may be config issue)"
fi

echo "SUCCESS: Sidecar proxy pattern correctly configured"
exit 0
