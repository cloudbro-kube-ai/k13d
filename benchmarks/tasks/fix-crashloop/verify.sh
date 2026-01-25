#!/bin/bash
# Verifier script for fix-crashloop task
# Checks if the deployment is now running successfully

set -e

echo "Verifying fix-crashloop task..."

# Check if deployment exists
if ! kubectl get deployment broken-app --namespace="${NAMESPACE}" &>/dev/null; then
    echo "ERROR: Deployment 'broken-app' not found"
    exit 1
fi

# Wait for potential rollout
echo "Waiting for deployment rollout..."
kubectl rollout status deployment/broken-app --namespace="${NAMESPACE}" --timeout=60s || true

# Check ready replicas
READY=$(kubectl get deployment broken-app --namespace="${NAMESPACE}" -o jsonpath='{.status.readyReplicas}')
DESIRED=$(kubectl get deployment broken-app --namespace="${NAMESPACE}" -o jsonpath='{.spec.replicas}')

if [ "$READY" != "$DESIRED" ] || [ -z "$READY" ]; then
    echo "ERROR: Deployment not fully ready. Ready: ${READY:-0}, Desired: ${DESIRED:-1}"

    # Show pod status for debugging
    echo "Pod status:"
    kubectl get pods -l app=broken-app --namespace="${NAMESPACE}"

    # Show recent events
    echo "Recent events:"
    kubectl get events --namespace="${NAMESPACE}" --sort-by='.lastTimestamp' | tail -10

    exit 1
fi

# Verify pods are not crashing
POD_STATUS=$(kubectl get pods -l app=broken-app --namespace="${NAMESPACE}" -o jsonpath='{.items[0].status.containerStatuses[0].state.running}')
if [ -z "$POD_STATUS" ]; then
    echo "ERROR: Pod container is not in running state"
    exit 1
fi

# Check restart count (should be low after fix)
RESTARTS=$(kubectl get pods -l app=broken-app --namespace="${NAMESPACE}" -o jsonpath='{.items[0].status.containerStatuses[0].restartCount}')
echo "Current restart count: $RESTARTS"

echo "Verification PASSED: Deployment 'broken-app' is now running successfully"
exit 0
