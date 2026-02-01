#!/bin/bash
# Verifier script for job-backoff-limit task

set -e

echo "Verifying job-backoff-limit task..."

NAMESPACE="job-test"

# Check if job exists
if ! kubectl get job retry-job --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Job 'retry-job' not found"
    exit 1
fi

# Check backoffLimit
BACKOFF=$(kubectl get job retry-job --namespace="$NAMESPACE" -o jsonpath='{.spec.backoffLimit}')
if [ "$BACKOFF" != "5" ]; then
    echo "ERROR: Job backoffLimit should be 5, got '$BACKOFF'"
    exit 1
fi

# Check activeDeadlineSeconds
DEADLINE=$(kubectl get job retry-job --namespace="$NAMESPACE" -o jsonpath='{.spec.activeDeadlineSeconds}')
if [ "$DEADLINE" != "120" ]; then
    echo "ERROR: activeDeadlineSeconds should be 120, got '$DEADLINE'"
    exit 1
fi

# Check restartPolicy
RESTART_POLICY=$(kubectl get job retry-job --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.restartPolicy}')
if [ "$RESTART_POLICY" != "Never" ]; then
    echo "ERROR: restartPolicy should be 'Never', got '$RESTART_POLICY'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get job retry-job --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}')
if [[ "$IMAGE" != *"busybox"* ]]; then
    echo "ERROR: Job should use busybox image, got $IMAGE"
    exit 1
fi

echo "Verification PASSED: Job 'retry-job' created with backoffLimit=5 and activeDeadlineSeconds=120"
exit 0
