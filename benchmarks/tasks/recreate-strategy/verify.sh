#!/bin/bash
# Verifier script for recreate-strategy task

set -e

echo "Verifying recreate-strategy task..."

NAMESPACE="deploy-test"

# Check if deployment exists
if ! kubectl get deployment stateful-app --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Deployment 'stateful-app' not found"
    exit 1
fi

# Check strategy type
STRATEGY=$(kubectl get deployment stateful-app --namespace="$NAMESPACE" -o jsonpath='{.spec.strategy.type}')
if [ "$STRATEGY" != "Recreate" ]; then
    echo "ERROR: Deployment strategy should be 'Recreate', got '$STRATEGY'"
    exit 1
fi

# Check replicas
REPLICAS=$(kubectl get deployment stateful-app --namespace="$NAMESPACE" -o jsonpath='{.spec.replicas}')
if [ "$REPLICAS" != "3" ]; then
    echo "ERROR: Deployment should have 3 replicas, got $REPLICAS"
    exit 1
fi

# Check image
IMAGE=$(kubectl get deployment stateful-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}')
if [[ "$IMAGE" != *"redis"* ]]; then
    echo "ERROR: Deployment should use redis image, got $IMAGE"
    exit 1
fi

echo "Verification PASSED: Deployment 'stateful-app' created with Recreate strategy"
exit 0
