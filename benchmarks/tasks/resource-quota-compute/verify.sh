#!/bin/bash
set -e

echo "Verifying resource-quota-compute task..."

NAMESPACE="quota-compute-test"

# Check ResourceQuota exists
if ! kubectl get resourcequota compute-quota -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: ResourceQuota 'compute-quota' not found"
    exit 1
fi

# Check quota has CPU limits
QUOTA_CPU=$(kubectl get resourcequota compute-quota -n "$NAMESPACE" -o jsonpath='{.spec.hard.requests\.cpu}' 2>/dev/null || echo "")
if [[ -z "$QUOTA_CPU" ]]; then
    echo "FAILED: ResourceQuota should have requests.cpu set"
    exit 1
fi

# Check quota has memory limits
QUOTA_MEM=$(kubectl get resourcequota compute-quota -n "$NAMESPACE" -o jsonpath='{.spec.hard.requests\.memory}' 2>/dev/null || echo "")
if [[ -z "$QUOTA_MEM" ]]; then
    echo "FAILED: ResourceQuota should have requests.memory set"
    exit 1
fi

# Check quota has pod limit
QUOTA_PODS=$(kubectl get resourcequota compute-quota -n "$NAMESPACE" -o jsonpath='{.spec.hard.pods}' 2>/dev/null || echo "")
if [[ "$QUOTA_PODS" != "10" ]]; then
    echo "FAILED: ResourceQuota pods limit should be '10', got '$QUOTA_PODS'"
    exit 1
fi

# Check Deployment exists
if ! kubectl get deployment quota-test-deploy -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Deployment 'quota-test-deploy' not found"
    exit 1
fi

# Wait for deployment to be ready
if ! kubectl rollout status deployment/quota-test-deploy -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Deployment did not become ready in time"
    exit 1
fi

# Verify quota usage is being tracked
USED_CPU=$(kubectl get resourcequota compute-quota -n "$NAMESPACE" -o jsonpath='{.status.used.requests\.cpu}' 2>/dev/null || echo "")
if [[ -z "$USED_CPU" ]]; then
    echo "WARNING: Quota usage not being tracked"
fi

echo "SUCCESS: ResourceQuota for compute correctly configured"
exit 0
