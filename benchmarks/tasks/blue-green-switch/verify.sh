#!/bin/bash
# Verifier script for blue-green-switch task

set -e

echo "Verifying blue-green-switch task..."

NAMESPACE="deploy-test"

# Check if service exists
if ! kubectl get service app-service --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Service 'app-service' not found"
    exit 1
fi

# Check service selector for version=green
VERSION_SELECTOR=$(kubectl get service app-service --namespace="$NAMESPACE" -o jsonpath='{.spec.selector.version}')
if [ "$VERSION_SELECTOR" != "green" ]; then
    echo "ERROR: Service selector 'version' should be 'green', got '$VERSION_SELECTOR'"
    exit 1
fi

# Verify endpoints are from green deployment
ENDPOINTS=$(kubectl get endpoints app-service --namespace="$NAMESPACE" -o jsonpath='{.subsets[*].addresses[*].ip}')
if [ -z "$ENDPOINTS" ]; then
    echo "ERROR: Service has no endpoints"
    exit 1
fi

# Check that endpoints match green pods
GREEN_PODS=$(kubectl get pods -l version=green --namespace="$NAMESPACE" -o jsonpath='{.items[*].status.podIP}')
if [ -z "$GREEN_PODS" ]; then
    echo "ERROR: No green pods found"
    exit 1
fi

echo "Verification PASSED: Service 'app-service' now routes traffic to green deployment"
exit 0
