#!/bin/bash
set -e

echo "Verifying rolling-update task..."

# Check current image
IMAGE=$(kubectl get deployment web-app -n rolling -o jsonpath='{.spec.template.spec.containers[0].image}')

if [ "$IMAGE" = "nginx:1.25" ]; then
    # Check rollout status
    if kubectl rollout status deployment/web-app -n rolling --timeout=30s; then
        echo "SUCCESS: Deployment updated to nginx:1.25 and rollout complete"
        exit 0
    else
        echo "FAILED: Rollout not complete"
        exit 1
    fi
else
    echo "FAILED: Expected image nginx:1.25, got '$IMAGE'"
    exit 1
fi
