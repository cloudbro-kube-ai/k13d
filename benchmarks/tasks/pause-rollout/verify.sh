#!/bin/bash
# Verifier script for pause-rollout task

set -e

echo "Verifying pause-rollout task..."

NAMESPACE="deploy-test"

# Check if deployment exists
if ! kubectl get deployment rolling-app --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Deployment 'rolling-app' not found"
    exit 1
fi

# Check if deployment is paused
PAUSED=$(kubectl get deployment rolling-app --namespace="$NAMESPACE" -o jsonpath='{.spec.paused}')
if [ "$PAUSED" != "true" ]; then
    echo "ERROR: Deployment rollout should be paused"
    exit 1
fi

echo "Verification PASSED: Deployment 'rolling-app' rollout is paused"
exit 0
