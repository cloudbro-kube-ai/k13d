#!/bin/bash
# Verifier script for exec-probe task

set -e

echo "Verifying exec-probe task..."

NAMESPACE="probe-test"

# Check if pod exists
if ! kubectl get pod exec-probe-pod --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Pod 'exec-probe-pod' not found"
    exit 1
fi

# Check for liveness probe with exec
LIVENESS_EXEC=$(kubectl get pod exec-probe-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].livenessProbe.exec}')
if [ -z "$LIVENESS_EXEC" ]; then
    echo "ERROR: Liveness probe exec not configured"
    exit 1
fi

# Check liveness probe command contains "cat"
LIVENESS_CMD=$(kubectl get pod exec-probe-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].livenessProbe.exec.command}')
if [[ "$LIVENESS_CMD" != *"cat"* ]]; then
    echo "ERROR: Liveness probe command should contain 'cat', got '$LIVENESS_CMD'"
    exit 1
fi

# Check liveness probe initialDelaySeconds
LIVENESS_DELAY=$(kubectl get pod exec-probe-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].livenessProbe.initialDelaySeconds}')
if [ "$LIVENESS_DELAY" != "5" ]; then
    echo "ERROR: Liveness probe initialDelaySeconds should be 5, got '$LIVENESS_DELAY'"
    exit 1
fi

# Check liveness probe periodSeconds
LIVENESS_PERIOD=$(kubectl get pod exec-probe-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].livenessProbe.periodSeconds}')
if [ "$LIVENESS_PERIOD" != "5" ]; then
    echo "ERROR: Liveness probe periodSeconds should be 5, got '$LIVENESS_PERIOD'"
    exit 1
fi

# Check for readiness probe with exec
READINESS_EXEC=$(kubectl get pod exec-probe-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].readinessProbe.exec}')
if [ -z "$READINESS_EXEC" ]; then
    echo "ERROR: Readiness probe exec not configured"
    exit 1
fi

# Check readiness probe command contains "test"
READINESS_CMD=$(kubectl get pod exec-probe-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].readinessProbe.exec.command}')
if [[ "$READINESS_CMD" != *"test"* ]]; then
    echo "ERROR: Readiness probe command should contain 'test', got '$READINESS_CMD'"
    exit 1
fi

# Check readiness probe initialDelaySeconds
READINESS_DELAY=$(kubectl get pod exec-probe-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].readinessProbe.initialDelaySeconds}')
if [ "$READINESS_DELAY" != "5" ]; then
    echo "ERROR: Readiness probe initialDelaySeconds should be 5, got '$READINESS_DELAY'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get pod exec-probe-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].image}')
if [[ "$IMAGE" != *"busybox"* ]]; then
    echo "ERROR: Pod should use busybox image, got $IMAGE"
    exit 1
fi

echo "Verification PASSED: Pod 'exec-probe-pod' created with correct exec-based probes"
exit 0
