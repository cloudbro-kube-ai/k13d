#!/bin/bash
# Verifier script for probe-failure-threshold task

set -e

echo "Verifying probe-failure-threshold task..."

NAMESPACE="probe-test"

# Check if deployment exists
if ! kubectl get deployment resilient-app --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Deployment 'resilient-app' not found"
    exit 1
fi

# Check replicas
REPLICAS=$(kubectl get deployment resilient-app --namespace="$NAMESPACE" -o jsonpath='{.spec.replicas}')
if [ "$REPLICAS" != "2" ]; then
    echo "ERROR: Deployment should have 2 replicas, got $REPLICAS"
    exit 1
fi

# Check for liveness probe
LIVENESS=$(kubectl get deployment resilient-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].livenessProbe}')
if [ -z "$LIVENESS" ]; then
    echo "ERROR: Liveness probe not configured"
    exit 1
fi

# Check liveness failureThreshold
LIVENESS_FAILURE=$(kubectl get deployment resilient-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].livenessProbe.failureThreshold}')
if [ "$LIVENESS_FAILURE" != "5" ]; then
    echo "ERROR: Liveness probe failureThreshold should be 5, got '$LIVENESS_FAILURE'"
    exit 1
fi

# Check liveness initialDelaySeconds
LIVENESS_DELAY=$(kubectl get deployment resilient-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].livenessProbe.initialDelaySeconds}')
if [ "$LIVENESS_DELAY" != "15" ]; then
    echo "ERROR: Liveness probe initialDelaySeconds should be 15, got '$LIVENESS_DELAY'"
    exit 1
fi

# Check liveness periodSeconds
LIVENESS_PERIOD=$(kubectl get deployment resilient-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].livenessProbe.periodSeconds}')
if [ "$LIVENESS_PERIOD" != "20" ]; then
    echo "ERROR: Liveness probe periodSeconds should be 20, got '$LIVENESS_PERIOD'"
    exit 1
fi

# Check for readiness probe
READINESS=$(kubectl get deployment resilient-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].readinessProbe}')
if [ -z "$READINESS" ]; then
    echo "ERROR: Readiness probe not configured"
    exit 1
fi

# Check readiness failureThreshold
READINESS_FAILURE=$(kubectl get deployment resilient-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].readinessProbe.failureThreshold}')
if [ "$READINESS_FAILURE" != "3" ]; then
    echo "ERROR: Readiness probe failureThreshold should be 3, got '$READINESS_FAILURE'"
    exit 1
fi

# Check readiness successThreshold
READINESS_SUCCESS=$(kubectl get deployment resilient-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].readinessProbe.successThreshold}')
if [ "$READINESS_SUCCESS" != "2" ]; then
    echo "ERROR: Readiness probe successThreshold should be 2, got '$READINESS_SUCCESS'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get deployment resilient-app --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}')
if [[ "$IMAGE" != *"httpd"* ]]; then
    echo "ERROR: Deployment should use httpd image, got $IMAGE"
    exit 1
fi

echo "Verification PASSED: Deployment 'resilient-app' created with customized probe thresholds"
exit 0
