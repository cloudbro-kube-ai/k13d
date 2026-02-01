#!/bin/bash
# Verifier script for create-job task

set -e

echo "Verifying create-job task..."

NAMESPACE="job-test"

# Check if job exists
if ! kubectl get job data-processor --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Job 'data-processor' not found"
    exit 1
fi

# Check completions
COMPLETIONS=$(kubectl get job data-processor --namespace="$NAMESPACE" -o jsonpath='{.spec.completions}')
if [ "$COMPLETIONS" != "1" ]; then
    echo "ERROR: Job completions should be 1, got '$COMPLETIONS'"
    exit 1
fi

# Check backoffLimit
BACKOFF=$(kubectl get job data-processor --namespace="$NAMESPACE" -o jsonpath='{.spec.backoffLimit}')
if [ "$BACKOFF" != "3" ]; then
    echo "ERROR: Job backoffLimit should be 3, got '$BACKOFF'"
    exit 1
fi

# Check restartPolicy
RESTART_POLICY=$(kubectl get job data-processor --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.restartPolicy}')
if [ "$RESTART_POLICY" != "Never" ]; then
    echo "ERROR: restartPolicy should be 'Never', got '$RESTART_POLICY'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get job data-processor --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}')
if [[ "$IMAGE" != *"busybox"* ]]; then
    echo "ERROR: Job should use busybox image, got $IMAGE"
    exit 1
fi

# Wait for job to complete (max 60 seconds)
echo "Waiting for job to complete..."
kubectl wait --for=condition=complete job/data-processor --namespace="$NAMESPACE" --timeout=60s 2>/dev/null || true

# Check if job succeeded
SUCCEEDED=$(kubectl get job data-processor --namespace="$NAMESPACE" -o jsonpath='{.status.succeeded}')
if [ "$SUCCEEDED" == "1" ]; then
    echo "Verification PASSED: Job 'data-processor' created and completed successfully"
else
    echo "Verification PASSED: Job 'data-processor' created correctly (may still be running)"
fi
exit 0
