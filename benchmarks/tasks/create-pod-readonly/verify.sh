#!/bin/bash
set -e

# Check if pod exists
if ! kubectl get pod readonly-pod -n benchmark &>/dev/null; then
    echo "FAIL: Pod 'readonly-pod' not found in namespace 'benchmark'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get pod readonly-pod -n benchmark -o jsonpath='{.spec.containers[0].image}')
if [[ "$IMAGE" != "nginx:alpine" ]]; then
    echo "FAIL: Pod image is '$IMAGE', expected 'nginx:alpine'"
    exit 1
fi

# Check read-only root filesystem
READONLY=$(kubectl get pod readonly-pod -n benchmark -o jsonpath='{.spec.containers[0].securityContext.readOnlyRootFilesystem}')
if [[ "$READONLY" != "true" ]]; then
    echo "FAIL: readOnlyRootFilesystem is '$READONLY', expected 'true'"
    exit 1
fi

# Check for volume mounts (at least /var/cache/nginx and /var/run should be mounted)
VOLUME_MOUNTS=$(kubectl get pod readonly-pod -n benchmark -o jsonpath='{.spec.containers[0].volumeMounts[*].mountPath}')
if [[ ! "$VOLUME_MOUNTS" =~ "/var/cache/nginx" ]] || [[ ! "$VOLUME_MOUNTS" =~ "/var/run" ]]; then
    echo "FAIL: Expected volume mounts at /var/cache/nginx and /var/run. Found: $VOLUME_MOUNTS"
    exit 1
fi

echo "PASS: Pod 'readonly-pod' created correctly with read-only root filesystem and required volume mounts"
exit 0
