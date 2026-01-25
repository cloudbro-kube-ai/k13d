#!/bin/bash
# Verifier script for scale-deployment task

set -e

echo "Verifying scale-deployment task..."

# Check if deployment exists
if ! kubectl get deployment web-app --namespace="${NAMESPACE}" &>/dev/null; then
    echo "ERROR: Deployment 'web-app' not found"
    exit 1
fi

# Check desired replicas
DESIRED=$(kubectl get deployment web-app --namespace="${NAMESPACE}" -o jsonpath='{.spec.replicas}')
if [ "$DESIRED" != "3" ]; then
    echo "ERROR: Deployment should have 3 desired replicas, but has $DESIRED"
    exit 1
fi

# Wait for scaling to complete
echo "Waiting for all replicas to be ready..."
kubectl rollout status deployment/web-app --namespace="${NAMESPACE}" --timeout=120s || true

# Check ready replicas
READY=$(kubectl get deployment web-app --namespace="${NAMESPACE}" -o jsonpath='{.status.readyReplicas}')
if [ "$READY" != "3" ]; then
    echo "ERROR: Expected 3 ready replicas, but got ${READY:-0}"
    kubectl get pods -l app=web-app --namespace="${NAMESPACE}"
    exit 1
fi

# Verify pod count
POD_COUNT=$(kubectl get pods -l app=web-app --namespace="${NAMESPACE}" --no-headers | wc -l | tr -d ' ')
if [ "$POD_COUNT" -lt 3 ]; then
    echo "ERROR: Expected at least 3 pods, but found $POD_COUNT"
    exit 1
fi

echo "Verification PASSED: Deployment 'web-app' successfully scaled to 3 replicas"
exit 0
