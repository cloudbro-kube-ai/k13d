#!/bin/bash
# Verifier script for fix-resource-quota task

set -e

echo "Verifying fix-resource-quota task..."

# Check if deployment exists
if ! kubectl get deployment web-app --namespace="${NAMESPACE}" &>/dev/null; then
    echo "ERROR: Deployment 'web-app' not found"
    exit 1
fi

# Check ready replicas
READY_REPLICAS=$(kubectl get deployment web-app --namespace="${NAMESPACE}" -o jsonpath='{.status.readyReplicas}')
DESIRED_REPLICAS=$(kubectl get deployment web-app --namespace="${NAMESPACE}" -o jsonpath='{.spec.replicas}')

if [ -z "$READY_REPLICAS" ]; then
    READY_REPLICAS=0
fi

if [ "$READY_REPLICAS" -lt 3 ]; then
    echo "ERROR: Deployment has $READY_REPLICAS ready replicas, expected at least 3"
    echo "The quota issue has not been resolved."
    exit 1
fi

echo "Verification PASSED: Deployment 'web-app' has $READY_REPLICAS ready replicas"
exit 0
