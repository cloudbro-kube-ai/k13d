#!/bin/bash
# Verifier script for readiness-probe-tcp task

set -e

echo "Verifying readiness-probe-tcp task..."

NAMESPACE="probe-test"

# Check if pod exists
if ! kubectl get pod readiness-tcp --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Pod 'readiness-tcp' not found"
    exit 1
fi

# Check for readiness probe
READINESS=$(kubectl get pod readiness-tcp --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].readinessProbe}')
if [ -z "$READINESS" ]; then
    echo "ERROR: Readiness probe not configured"
    exit 1
fi

# Check tcpSocket port
TCP_PORT=$(kubectl get pod readiness-tcp --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].readinessProbe.tcpSocket.port}')
if [ "$TCP_PORT" != "6379" ]; then
    echo "ERROR: Readiness probe tcpSocket port should be 6379, got '$TCP_PORT'"
    exit 1
fi

# Check initialDelaySeconds
INITIAL_DELAY=$(kubectl get pod readiness-tcp --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].readinessProbe.initialDelaySeconds}')
if [ "$INITIAL_DELAY" != "5" ]; then
    echo "ERROR: initialDelaySeconds should be 5, got '$INITIAL_DELAY'"
    exit 1
fi

# Check periodSeconds
PERIOD=$(kubectl get pod readiness-tcp --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].readinessProbe.periodSeconds}')
if [ "$PERIOD" != "10" ]; then
    echo "ERROR: periodSeconds should be 10, got '$PERIOD'"
    exit 1
fi

# Check successThreshold
SUCCESS=$(kubectl get pod readiness-tcp --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].readinessProbe.successThreshold}')
if [ "$SUCCESS" != "1" ]; then
    echo "ERROR: successThreshold should be 1, got '$SUCCESS'"
    exit 1
fi

# Check failureThreshold
FAILURE=$(kubectl get pod readiness-tcp --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].readinessProbe.failureThreshold}')
if [ "$FAILURE" != "3" ]; then
    echo "ERROR: failureThreshold should be 3, got '$FAILURE'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get pod readiness-tcp --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].image}')
if [[ "$IMAGE" != *"redis"* ]]; then
    echo "ERROR: Pod should use redis image, got $IMAGE"
    exit 1
fi

echo "Verification PASSED: Pod 'readiness-tcp' created with correct TCP readiness probe"
exit 0
