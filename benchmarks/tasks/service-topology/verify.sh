#!/bin/bash
# Verifier script for service-topology task

set -e

echo "Verifying service-topology task..."

NAMESPACE="service-test"

# Check if service exists
if ! kubectl get service local-svc --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Service 'local-svc' not found"
    exit 1
fi

# Check internalTrafficPolicy
TRAFFIC_POLICY=$(kubectl get service local-svc --namespace="$NAMESPACE" -o jsonpath='{.spec.internalTrafficPolicy}')
if [ "$TRAFFIC_POLICY" != "Local" ]; then
    echo "ERROR: Service internalTrafficPolicy should be 'Local', got '$TRAFFIC_POLICY'"
    exit 1
fi

# Check selector
SELECTOR=$(kubectl get service local-svc --namespace="$NAMESPACE" -o jsonpath='{.spec.selector.app}')
if [ "$SELECTOR" != "local-app" ]; then
    echo "ERROR: Service selector 'app' should be 'local-app', got '$SELECTOR'"
    exit 1
fi

# Check port
PORT=$(kubectl get service local-svc --namespace="$NAMESPACE" -o jsonpath='{.spec.ports[0].port}')
if [ "$PORT" != "80" ]; then
    echo "ERROR: Service port should be 80, got '$PORT'"
    exit 1
fi

# Check deployment exists
if ! kubectl get deployment local-app --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Deployment 'local-app' not found"
    exit 1
fi

echo "Verification PASSED: Service 'local-svc' created with internalTrafficPolicy=Local"
exit 0
