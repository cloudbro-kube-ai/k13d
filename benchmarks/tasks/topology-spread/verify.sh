#!/bin/bash
# Verifier script for topology-spread task

set -e

echo "Verifying topology-spread task..."

NAMESPACE="schedule-test"

# Check if deployment exists
if ! kubectl get deployment spread-app --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Deployment 'spread-app' not found"
    exit 1
fi

# Check replicas
REPLICAS=$(kubectl get deployment spread-app --namespace="$NAMESPACE" -o jsonpath='{.spec.replicas}')
if [ "$REPLICAS" != "6" ]; then
    echo "ERROR: Deployment should have 6 replicas, got $REPLICAS"
    exit 1
fi

# Check labels
APP_LABEL=$(kubectl get deployment spread-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.metadata.labels.app}')
if [ "$APP_LABEL" != "spread-app" ]; then
    echo "ERROR: Deployment should have label app=spread-app"
    exit 1
fi

# Check for topology spread constraints
TOPOLOGY_SPREAD=$(kubectl get deployment spread-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.topologySpreadConstraints}')
if [ -z "$TOPOLOGY_SPREAD" ] || [ "$TOPOLOGY_SPREAD" == "[]" ]; then
    echo "ERROR: Topology spread constraints not found"
    exit 1
fi

# Check maxSkew
MAX_SKEW=$(kubectl get deployment spread-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.topologySpreadConstraints[0].maxSkew}')
if [ "$MAX_SKEW" != "1" ]; then
    echo "ERROR: maxSkew should be 1, got '$MAX_SKEW'"
    exit 1
fi

# Check topologyKey
TOPOLOGY_KEY=$(kubectl get deployment spread-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.topologySpreadConstraints[0].topologyKey}')
if [ "$TOPOLOGY_KEY" != "kubernetes.io/hostname" ]; then
    echo "ERROR: topologyKey should be 'kubernetes.io/hostname', got '$TOPOLOGY_KEY'"
    exit 1
fi

# Check whenUnsatisfiable
WHEN_UNSATISFIABLE=$(kubectl get deployment spread-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.topologySpreadConstraints[0].whenUnsatisfiable}')
if [ "$WHEN_UNSATISFIABLE" != "DoNotSchedule" ]; then
    echo "ERROR: whenUnsatisfiable should be 'DoNotSchedule', got '$WHEN_UNSATISFIABLE'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get deployment spread-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}')
if [[ "$IMAGE" != *"nginx"* ]]; then
    echo "ERROR: Deployment should use nginx image, got $IMAGE"
    exit 1
fi

echo "Verification PASSED: Deployment 'spread-app' created with correct topology spread constraints"
exit 0
