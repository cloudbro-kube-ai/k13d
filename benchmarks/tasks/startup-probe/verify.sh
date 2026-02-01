#!/bin/bash
# Verifier script for startup-probe task

set -e

echo "Verifying startup-probe task..."

NAMESPACE="probe-test"

# Check if pod exists
if ! kubectl get pod startup-app --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Pod 'startup-app' not found"
    exit 1
fi

# Check for startup probe
STARTUP=$(kubectl get pod startup-app --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].startupProbe}')
if [ -z "$STARTUP" ]; then
    echo "ERROR: Startup probe not configured"
    exit 1
fi

# Check startup probe httpGet
STARTUP_PATH=$(kubectl get pod startup-app --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].startupProbe.httpGet.path}')
if [ "$STARTUP_PATH" != "/" ]; then
    echo "ERROR: Startup probe httpGet path should be '/', got '$STARTUP_PATH'"
    exit 1
fi

# Check startup probe failureThreshold
STARTUP_FAILURE=$(kubectl get pod startup-app --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].startupProbe.failureThreshold}')
if [ "$STARTUP_FAILURE" != "30" ]; then
    echo "ERROR: Startup probe failureThreshold should be 30, got '$STARTUP_FAILURE'"
    exit 1
fi

# Check startup probe periodSeconds
STARTUP_PERIOD=$(kubectl get pod startup-app --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].startupProbe.periodSeconds}')
if [ "$STARTUP_PERIOD" != "10" ]; then
    echo "ERROR: Startup probe periodSeconds should be 10, got '$STARTUP_PERIOD'"
    exit 1
fi

# Check for liveness probe
LIVENESS=$(kubectl get pod startup-app --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].livenessProbe}')
if [ -z "$LIVENESS" ]; then
    echo "ERROR: Liveness probe not configured"
    exit 1
fi

# Check liveness probe httpGet
LIVENESS_PATH=$(kubectl get pod startup-app --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].livenessProbe.httpGet.path}')
if [ "$LIVENESS_PATH" != "/" ]; then
    echo "ERROR: Liveness probe httpGet path should be '/', got '$LIVENESS_PATH'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get pod startup-app --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].image}')
if [[ "$IMAGE" != *"nginx"* ]]; then
    echo "ERROR: Pod should use nginx image, got $IMAGE"
    exit 1
fi

echo "Verification PASSED: Pod 'startup-app' created with both startup and liveness probes"
exit 0
