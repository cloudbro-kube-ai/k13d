#!/bin/bash
set -e

echo "Verifying create-network-policy task..."

# Check if NetworkPolicy exists
if ! kubectl get networkpolicy api-allow -n secure-app &>/dev/null; then
    echo "FAILED: NetworkPolicy 'api-allow' not found"
    exit 1
fi

# Verify podSelector targets app=api
POD_SELECTOR=$(kubectl get networkpolicy api-allow -n secure-app -o jsonpath='{.spec.podSelector.matchLabels.app}')
if [ "$POD_SELECTOR" != "api" ]; then
    echo "FAILED: podSelector should target app=api, got '$POD_SELECTOR'"
    exit 1
fi

# Verify ingress from app=frontend exists
INGRESS_FROM=$(kubectl get networkpolicy api-allow -n secure-app -o json | grep -c "frontend" || echo "0")
if [ "$INGRESS_FROM" -eq "0" ]; then
    echo "FAILED: ingress from app=frontend not found"
    exit 1
fi

# Verify port 8080
PORT=$(kubectl get networkpolicy api-allow -n secure-app -o json | grep -c "8080" || echo "0")
if [ "$PORT" -eq "0" ]; then
    echo "FAILED: port 8080 not specified"
    exit 1
fi

echo "SUCCESS: NetworkPolicy correctly configured"
exit 0
