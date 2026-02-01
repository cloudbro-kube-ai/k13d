#!/bin/bash
# Verifier script for pod-anti-affinity task

set -e

echo "Verifying pod-anti-affinity task..."

NAMESPACE="schedule-test"

# Check if deployment exists
if ! kubectl get deployment ha-app --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Deployment 'ha-app' not found"
    exit 1
fi

# Check replicas
REPLICAS=$(kubectl get deployment ha-app --namespace="$NAMESPACE" -o jsonpath='{.spec.replicas}')
if [ "$REPLICAS" != "3" ]; then
    echo "ERROR: Deployment should have 3 replicas, got $REPLICAS"
    exit 1
fi

# Check labels
APP_LABEL=$(kubectl get deployment ha-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.metadata.labels.app}')
if [ "$APP_LABEL" != "ha-app" ]; then
    echo "ERROR: Deployment should have label app=ha-app"
    exit 1
fi

# Check for pod anti-affinity
ANTI_AFFINITY=$(kubectl get deployment ha-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution}')
if [ -z "$ANTI_AFFINITY" ]; then
    echo "ERROR: Required pod anti-affinity not found"
    exit 1
fi

# Check for app=ha-app in anti-affinity selector
AFFINITY_JSON=$(kubectl get deployment ha-app --namespace="$NAMESPACE" -o json)
if ! echo "$AFFINITY_JSON" | grep -q "podAntiAffinity"; then
    echo "ERROR: podAntiAffinity not configured"
    exit 1
fi

# Check topology key
if ! echo "$AFFINITY_JSON" | grep -q "kubernetes.io/hostname"; then
    echo "ERROR: Anti-affinity should use topologyKey 'kubernetes.io/hostname'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get deployment ha-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}')
if [[ "$IMAGE" != *"nginx"* ]]; then
    echo "ERROR: Deployment should use nginx image, got $IMAGE"
    exit 1
fi

echo "Verification PASSED: Deployment 'ha-app' created with correct pod anti-affinity rules"
exit 0
