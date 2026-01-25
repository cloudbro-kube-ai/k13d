#!/bin/bash
# Verifier script for create-service task

set -e

echo "Verifying create-service task..."

# Check if service exists
if ! kubectl get service backend-service --namespace="${NAMESPACE}" &>/dev/null; then
    echo "ERROR: Service 'backend-service' not found"
    exit 1
fi

# Check service type
TYPE=$(kubectl get service backend-service --namespace="${NAMESPACE}" -o jsonpath='{.spec.type}')
if [ "$TYPE" != "ClusterIP" ]; then
    echo "ERROR: Service type should be ClusterIP, but got $TYPE"
    exit 1
fi

# Check port
PORT=$(kubectl get service backend-service --namespace="${NAMESPACE}" -o jsonpath='{.spec.ports[0].port}')
if [ "$PORT" != "80" ]; then
    echo "ERROR: Service port should be 80, but got $PORT"
    exit 1
fi

# Check selector
SELECTOR=$(kubectl get service backend-service --namespace="${NAMESPACE}" -o jsonpath='{.spec.selector.app}')
if [ "$SELECTOR" != "backend-app" ]; then
    echo "ERROR: Service selector should be 'app: backend-app', but got 'app: $SELECTOR'"
    exit 1
fi

# Check endpoints
ENDPOINTS=$(kubectl get endpoints backend-service --namespace="${NAMESPACE}" -o jsonpath='{.subsets[0].addresses}')
if [ -z "$ENDPOINTS" ]; then
    echo "ERROR: Service has no endpoints"
    kubectl get endpoints backend-service --namespace="${NAMESPACE}" -o yaml
    exit 1
fi

# Count endpoints (should match pod count)
ENDPOINT_COUNT=$(kubectl get endpoints backend-service --namespace="${NAMESPACE}" -o jsonpath='{.subsets[0].addresses}' | grep -o '"ip"' | wc -l | tr -d ' ')
if [ "$ENDPOINT_COUNT" -lt 1 ]; then
    echo "ERROR: Expected at least 1 endpoint, but got $ENDPOINT_COUNT"
    exit 1
fi

echo "Verification PASSED: Service 'backend-service' created with $ENDPOINT_COUNT endpoint(s)"
exit 0
