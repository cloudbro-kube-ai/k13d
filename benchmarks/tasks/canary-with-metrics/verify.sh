#!/bin/bash
set -euo pipefail

NAMESPACE="canary-metrics"

echo "Verifying canary-with-metrics..."

# Check app-stable deployment
STABLE_REPLICAS=$(kubectl get deployment app-stable -n $NAMESPACE -o jsonpath='{.spec.replicas}')
if [[ "$STABLE_REPLICAS" != "3" ]]; then
    echo "ERROR: app-stable should have 3 replicas"
    exit 1
fi

STABLE_VERSION_LABEL=$(kubectl get deployment app-stable -n $NAMESPACE -o jsonpath='{.spec.template.metadata.labels.version}')
if [[ "$STABLE_VERSION_LABEL" != "v1" ]]; then
    echo "ERROR: app-stable should have version=v1 label"
    exit 1
fi

STABLE_ROLE=$(kubectl get deployment app-stable -n $NAMESPACE -o jsonpath='{.metadata.annotations.deployment\.k13d\.io/role}')
if [[ "$STABLE_ROLE" != "stable" ]]; then
    echo "ERROR: app-stable should have role=stable annotation"
    exit 1
fi

STABLE_VERSION=$(kubectl get deployment app-stable -n $NAMESPACE -o jsonpath='{.metadata.annotations.deployment\.k13d\.io/version}')
if [[ "$STABLE_VERSION" != "1.0.0" ]]; then
    echo "ERROR: app-stable should have version=1.0.0 annotation"
    exit 1
fi

# Check app-canary deployment exists
if ! kubectl get deployment app-canary -n $NAMESPACE &>/dev/null; then
    echo "ERROR: Deployment 'app-canary' not found"
    exit 1
fi

CANARY_REPLICAS=$(kubectl get deployment app-canary -n $NAMESPACE -o jsonpath='{.spec.replicas}')
if [[ "$CANARY_REPLICAS" != "1" ]]; then
    echo "ERROR: app-canary should have 1 replica"
    exit 1
fi

CANARY_VERSION_LABEL=$(kubectl get deployment app-canary -n $NAMESPACE -o jsonpath='{.spec.template.metadata.labels.version}')
if [[ "$CANARY_VERSION_LABEL" != "v2" ]]; then
    echo "ERROR: app-canary should have version=v2 label"
    exit 1
fi

CANARY_ROLE=$(kubectl get deployment app-canary -n $NAMESPACE -o jsonpath='{.metadata.annotations.deployment\.k13d\.io/role}')
if [[ "$CANARY_ROLE" != "canary" ]]; then
    echo "ERROR: app-canary should have role=canary annotation"
    exit 1
fi

CANARY_WEIGHT=$(kubectl get deployment app-canary -n $NAMESPACE -o jsonpath='{.metadata.annotations.deployment\.k13d\.io/traffic-weight}')
if [[ "$CANARY_WEIGHT" != "10" ]]; then
    echo "ERROR: app-canary should have traffic-weight=10 annotation"
    exit 1
fi

# Check Service configuration
SVC_SELECTOR=$(kubectl get service app-service -n $NAMESPACE -o jsonpath='{.spec.selector.app}')
if [[ "$SVC_SELECTOR" != "myapp" ]]; then
    echo "ERROR: app-service should select app=myapp"
    exit 1
fi

# Service should NOT have version selector (to match both stable and canary)
SVC_VERSION=$(kubectl get service app-service -n $NAMESPACE -o jsonpath='{.spec.selector.version}' 2>/dev/null || echo "")
if [[ -n "$SVC_VERSION" ]]; then
    echo "ERROR: app-service should not have version selector to match both deployments"
    exit 1
fi

SVC_CANARY_ENABLED=$(kubectl get service app-service -n $NAMESPACE -o jsonpath='{.metadata.annotations.service\.k13d\.io/canary-enabled}')
if [[ "$SVC_CANARY_ENABLED" != "true" ]]; then
    echo "ERROR: app-service should have canary-enabled=true annotation"
    exit 1
fi

SVC_CANARY_WEIGHT=$(kubectl get service app-service -n $NAMESPACE -o jsonpath='{.metadata.annotations.service\.k13d\.io/canary-weight}')
if [[ "$SVC_CANARY_WEIGHT" != "10" ]]; then
    echo "ERROR: app-service should have canary-weight=10 annotation"
    exit 1
fi

# Check ConfigMap
if ! kubectl get configmap canary-config -n $NAMESPACE &>/dev/null; then
    echo "ERROR: ConfigMap 'canary-config' not found"
    exit 1
fi

THRESHOLD=$(kubectl get configmap canary-config -n $NAMESPACE -o jsonpath='{.data.CANARY_THRESHOLD}')
if [[ "$THRESHOLD" != "5" ]]; then
    echo "ERROR: canary-config CANARY_THRESHOLD should be 5"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Canary deployment with metrics configured correctly."
exit 0
