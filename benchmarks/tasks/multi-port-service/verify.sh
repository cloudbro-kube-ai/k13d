#!/bin/bash
# Verifier script for multi-port-service task

set -e

echo "Verifying multi-port-service task..."

NAMESPACE="service-test"

# Check if service exists
if ! kubectl get service web-service --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Service 'web-service' not found"
    exit 1
fi

# Check selector
SELECTOR=$(kubectl get service web-service --namespace="$NAMESPACE" -o jsonpath='{.spec.selector.app}')
if [ "$SELECTOR" != "web-app" ]; then
    echo "ERROR: Service selector 'app' should be 'web-app', got '$SELECTOR'"
    exit 1
fi

# Check port count
PORT_COUNT=$(kubectl get service web-service --namespace="$NAMESPACE" -o jsonpath='{.spec.ports}' | grep -o '"port"' | wc -l)
if [ "$PORT_COUNT" -lt 3 ]; then
    echo "ERROR: Service should have at least 3 ports, found $PORT_COUNT"
    exit 1
fi

# Check HTTP port (80)
HTTP_PORT=$(kubectl get service web-service --namespace="$NAMESPACE" -o jsonpath='{.spec.ports[?(@.port==80)].port}')
if [ "$HTTP_PORT" != "80" ]; then
    echo "ERROR: HTTP port 80 not found"
    exit 1
fi

# Check HTTPS port (443)
HTTPS_PORT=$(kubectl get service web-service --namespace="$NAMESPACE" -o jsonpath='{.spec.ports[?(@.port==443)].port}')
if [ "$HTTPS_PORT" != "443" ]; then
    echo "ERROR: HTTPS port 443 not found"
    exit 1
fi

# Check metrics port (9090)
METRICS_PORT=$(kubectl get service web-service --namespace="$NAMESPACE" -o jsonpath='{.spec.ports[?(@.port==9090)].port}')
if [ "$METRICS_PORT" != "9090" ]; then
    echo "ERROR: Metrics port 9090 not found"
    exit 1
fi

# Check deployment exists
if ! kubectl get deployment web-app --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Deployment 'web-app' not found"
    exit 1
fi

echo "Verification PASSED: Multi-port service 'web-service' created with ports 80, 443, and 9090"
exit 0
