#!/bin/bash
# Verifier script for minready-config task

set -e

echo "Verifying minready-config task..."

NAMESPACE="deploy-test"

# Check if deployment exists
if ! kubectl get deployment stable-app --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Deployment 'stable-app' not found"
    exit 1
fi

# Check minReadySeconds
MIN_READY=$(kubectl get deployment stable-app --namespace="$NAMESPACE" -o jsonpath='{.spec.minReadySeconds}')
if [ "$MIN_READY" != "30" ]; then
    echo "ERROR: minReadySeconds should be 30, got '$MIN_READY'"
    exit 1
fi

# Check replicas
REPLICAS=$(kubectl get deployment stable-app --namespace="$NAMESPACE" -o jsonpath='{.spec.replicas}')
if [ "$REPLICAS" != "3" ]; then
    echo "ERROR: Deployment should have 3 replicas, got $REPLICAS"
    exit 1
fi

# Check image
IMAGE=$(kubectl get deployment stable-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}')
if [[ "$IMAGE" != *"nginx"* ]]; then
    echo "ERROR: Deployment should use nginx image, got $IMAGE"
    exit 1
fi

echo "Verification PASSED: Deployment 'stable-app' created with minReadySeconds=30"
exit 0
