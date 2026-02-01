#!/bin/bash
# Verifier script for liveness-probe-http task

set -e

echo "Verifying liveness-probe-http task..."

NAMESPACE="probe-test"

# Check if pod exists
if ! kubectl get pod liveness-http --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Pod 'liveness-http' not found"
    exit 1
fi

# Check for liveness probe
LIVENESS=$(kubectl get pod liveness-http --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].livenessProbe}')
if [ -z "$LIVENESS" ]; then
    echo "ERROR: Liveness probe not configured"
    exit 1
fi

# Check httpGet path
HTTP_PATH=$(kubectl get pod liveness-http --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].livenessProbe.httpGet.path}')
if [ "$HTTP_PATH" != "/" ]; then
    echo "ERROR: Liveness probe httpGet path should be '/', got '$HTTP_PATH'"
    exit 1
fi

# Check httpGet port
HTTP_PORT=$(kubectl get pod liveness-http --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].livenessProbe.httpGet.port}')
if [ "$HTTP_PORT" != "80" ]; then
    echo "ERROR: Liveness probe httpGet port should be 80, got '$HTTP_PORT'"
    exit 1
fi

# Check initialDelaySeconds
INITIAL_DELAY=$(kubectl get pod liveness-http --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].livenessProbe.initialDelaySeconds}')
if [ "$INITIAL_DELAY" != "10" ]; then
    echo "ERROR: initialDelaySeconds should be 10, got '$INITIAL_DELAY'"
    exit 1
fi

# Check periodSeconds
PERIOD=$(kubectl get pod liveness-http --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].livenessProbe.periodSeconds}')
if [ "$PERIOD" != "5" ]; then
    echo "ERROR: periodSeconds should be 5, got '$PERIOD'"
    exit 1
fi

# Check timeoutSeconds
TIMEOUT=$(kubectl get pod liveness-http --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].livenessProbe.timeoutSeconds}')
if [ "$TIMEOUT" != "3" ]; then
    echo "ERROR: timeoutSeconds should be 3, got '$TIMEOUT'"
    exit 1
fi

# Check failureThreshold
FAILURE=$(kubectl get pod liveness-http --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].livenessProbe.failureThreshold}')
if [ "$FAILURE" != "3" ]; then
    echo "ERROR: failureThreshold should be 3, got '$FAILURE'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get pod liveness-http --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].image}')
if [[ "$IMAGE" != *"nginx"* ]]; then
    echo "ERROR: Pod should use nginx image, got $IMAGE"
    exit 1
fi

echo "Verification PASSED: Pod 'liveness-http' created with correct HTTP liveness probe"
exit 0
