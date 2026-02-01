#!/bin/bash
set -e

echo "Verifying run-as-non-root task..."

NAMESPACE="nonroot-test"

# Check Pod exists
if ! kubectl get pod nonroot-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'nonroot-pod' not found"
    exit 1
fi

# Check runAsNonRoot is set
RUN_AS_NONROOT=$(kubectl get pod nonroot-pod -n "$NAMESPACE" -o jsonpath='{.spec.securityContext.runAsNonRoot}' 2>/dev/null || echo "")
if [[ "$RUN_AS_NONROOT" != "true" ]]; then
    echo "FAILED: runAsNonRoot should be 'true', got '$RUN_AS_NONROOT'"
    exit 1
fi

# Check runAsUser is set to 1000
RUN_AS_USER=$(kubectl get pod nonroot-pod -n "$NAMESPACE" -o jsonpath='{.spec.securityContext.runAsUser}' 2>/dev/null || echo "")
if [[ "$RUN_AS_USER" != "1000" ]]; then
    echo "FAILED: runAsUser should be '1000', got '$RUN_AS_USER'"
    exit 1
fi

# Check runAsGroup is set to 3000
RUN_AS_GROUP=$(kubectl get pod nonroot-pod -n "$NAMESPACE" -o jsonpath='{.spec.securityContext.runAsGroup}' 2>/dev/null || echo "")
if [[ "$RUN_AS_GROUP" != "3000" ]]; then
    echo "FAILED: runAsGroup should be '3000', got '$RUN_AS_GROUP'"
    exit 1
fi

# Check fsGroup is set to 2000
FS_GROUP=$(kubectl get pod nonroot-pod -n "$NAMESPACE" -o jsonpath='{.spec.securityContext.fsGroup}' 2>/dev/null || echo "")
if [[ "$FS_GROUP" != "2000" ]]; then
    echo "FAILED: fsGroup should be '2000', got '$FS_GROUP'"
    exit 1
fi

# Check pod status (may or may not be running depending on image compatibility)
STATUS=$(kubectl get pod nonroot-pod -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
echo "INFO: Pod status is '$STATUS'"

echo "SUCCESS: Run as non-root security context correctly configured"
exit 0
