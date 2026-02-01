#!/bin/bash
# Verifier script for pod-affinity task

set -e

echo "Verifying pod-affinity task..."

NAMESPACE="schedule-test"

# Check if web-app deployment exists
if ! kubectl get deployment web-app --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Deployment 'web-app' not found"
    exit 1
fi

# Check web-app has correct labels
WEB_LABEL=$(kubectl get deployment web-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.metadata.labels.app}')
if [ "$WEB_LABEL" != "web" ]; then
    echo "ERROR: Deployment 'web-app' should have label app=web"
    exit 1
fi

# Check if cache pod exists
if ! kubectl get pod cache-pod --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Pod 'cache-pod' not found"
    exit 1
fi

# Check cache pod labels
CACHE_LABEL=$(kubectl get pod cache-pod --namespace="$NAMESPACE" -o jsonpath='{.metadata.labels.app}')
if [ "$CACHE_LABEL" != "cache" ]; then
    echo "ERROR: Pod 'cache-pod' should have label app=cache"
    exit 1
fi

# Check for pod affinity
POD_AFFINITY=$(kubectl get pod cache-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution}')
if [ -z "$POD_AFFINITY" ]; then
    echo "ERROR: Required pod affinity not found"
    exit 1
fi

# Check for app=web label selector in affinity
AFFINITY_SELECTOR=$(kubectl get pod cache-pod --namespace="$NAMESPACE" -o json | grep -o '"key":"app"' | head -1)
if [ -z "$AFFINITY_SELECTOR" ]; then
    echo "ERROR: Pod affinity should match pods with label 'app'"
    exit 1
fi

# Check topology key
TOPOLOGY=$(kubectl get pod cache-pod --namespace="$NAMESPACE" -o json | grep -o '"topologyKey":"kubernetes.io/hostname"' | head -1)
if [ -z "$TOPOLOGY" ]; then
    echo "ERROR: Pod affinity should use topologyKey 'kubernetes.io/hostname'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get pod cache-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].image}')
if [[ "$IMAGE" != *"redis"* ]]; then
    echo "ERROR: Pod should use redis image, got $IMAGE"
    exit 1
fi

echo "Verification PASSED: Pod 'cache-pod' created with correct pod affinity rules"
exit 0
