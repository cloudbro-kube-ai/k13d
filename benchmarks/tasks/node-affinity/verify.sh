#!/bin/bash
# Verifier script for node-affinity task

set -e

echo "Verifying node-affinity task..."

NAMESPACE="schedule-test"

# Check if pod exists
if ! kubectl get pod affinity-pod --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Pod 'affinity-pod' not found"
    exit 1
fi

# Check for required node affinity
REQUIRED_AFFINITY=$(kubectl get pod affinity-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution}')
if [ -z "$REQUIRED_AFFINITY" ]; then
    echo "ERROR: Required node affinity not found"
    exit 1
fi

# Check for disktype=ssd in required affinity
DISKTYPE_KEY=$(kubectl get pod affinity-pod --namespace="$NAMESPACE" -o json | grep -o '"key":"disktype"' | head -1)
if [ -z "$DISKTYPE_KEY" ]; then
    echo "ERROR: Node affinity should match nodes with label 'disktype'"
    exit 1
fi

# Check for preferred node affinity
PREFERRED_AFFINITY=$(kubectl get pod affinity-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution}')
if [ -z "$PREFERRED_AFFINITY" ]; then
    echo "ERROR: Preferred node affinity not found"
    exit 1
fi

# Check for zone preference
ZONE_KEY=$(kubectl get pod affinity-pod --namespace="$NAMESPACE" -o json | grep -o '"key":"zone"' | head -1)
if [ -z "$ZONE_KEY" ]; then
    echo "ERROR: Node affinity should prefer nodes with label 'zone'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get pod affinity-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].image}')
if [[ "$IMAGE" != *"nginx"* ]]; then
    echo "ERROR: Pod should use nginx image, got $IMAGE"
    exit 1
fi

echo "Verification PASSED: Pod 'affinity-pod' created with correct node affinity rules"
exit 0
