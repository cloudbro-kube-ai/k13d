#!/bin/bash
set -e

echo "Verifying seccomp-profile task..."

NAMESPACE="seccomp-test"

# Check Pod exists
if ! kubectl get pod seccomp-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'seccomp-pod' not found"
    exit 1
fi

# Check seccomp profile is set
SECCOMP_TYPE=$(kubectl get pod seccomp-pod -n "$NAMESPACE" -o jsonpath='{.spec.securityContext.seccompProfile.type}' 2>/dev/null || echo "")
if [[ "$SECCOMP_TYPE" != "RuntimeDefault" ]]; then
    echo "FAILED: seccompProfile.type should be 'RuntimeDefault', got '$SECCOMP_TYPE'"
    exit 1
fi

# Check runAsNonRoot
RUN_AS_NONROOT=$(kubectl get pod seccomp-pod -n "$NAMESPACE" -o jsonpath='{.spec.securityContext.runAsNonRoot}' 2>/dev/null || echo "")
if [[ "$RUN_AS_NONROOT" != "true" ]]; then
    echo "FAILED: runAsNonRoot should be 'true', got '$RUN_AS_NONROOT'"
    exit 1
fi

# Check runAsUser
RUN_AS_USER=$(kubectl get pod seccomp-pod -n "$NAMESPACE" -o jsonpath='{.spec.securityContext.runAsUser}' 2>/dev/null || echo "")
if [[ "$RUN_AS_USER" != "1000" ]]; then
    echo "FAILED: runAsUser should be '1000', got '$RUN_AS_USER'"
    exit 1
fi

# Check image is alpine
IMAGE=$(kubectl get pod seccomp-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].image}' 2>/dev/null || echo "")
if [[ "$IMAGE" != *"alpine"* ]]; then
    echo "FAILED: Image should be alpine, got '$IMAGE'"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/seccomp-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

echo "SUCCESS: Seccomp profile correctly configured"
exit 0
