#!/bin/bash
set -euo pipefail

NAMESPACE="pause-demo"
TIMEOUT="120s"

echo "Verifying deployment-pause-resume..."

# Check deployment is available and NOT paused
PAUSED=$(kubectl get deployment web-app -n $NAMESPACE -o jsonpath='{.spec.paused}')
if [[ "$PAUSED" == "true" ]]; then
    echo "ERROR: Deployment should not be paused after completion"
    exit 1
fi

# Wait for deployment to be available
if ! kubectl wait --for=condition=Available deployment/web-app -n $NAMESPACE --timeout=$TIMEOUT; then
    echo "ERROR: Deployment is not available"
    exit 1
fi

# Check image was updated
IMAGE=$(kubectl get deployment web-app -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}')
if [[ "$IMAGE" != "nginx:1.25-alpine" ]]; then
    echo "ERROR: Image should be nginx:1.25-alpine, got '$IMAGE'"
    exit 1
fi

# Check environment variables
ENV_APP=$(kubectl get deployment web-app -n $NAMESPACE -o json | jq -r '.spec.template.spec.containers[0].env[] | select(.name == "APP_ENV") | .value // empty')
if [[ "$ENV_APP" != "production" ]]; then
    echo "ERROR: APP_ENV should be 'production'"
    exit 1
fi

ENV_LOG=$(kubectl get deployment web-app -n $NAMESPACE -o json | jq -r '.spec.template.spec.containers[0].env[] | select(.name == "LOG_LEVEL") | .value // empty')
if [[ "$ENV_LOG" != "info" ]]; then
    echo "ERROR: LOG_LEVEL should be 'info'"
    exit 1
fi

ENV_VER=$(kubectl get deployment web-app -n $NAMESPACE -o json | jq -r '.spec.template.spec.containers[0].env[] | select(.name == "VERSION") | .value // empty')
if [[ "$ENV_VER" != "2.0" ]]; then
    echo "ERROR: VERSION should be '2.0'"
    exit 1
fi

# Check resource limits
MEM_LIMIT=$(kubectl get deployment web-app -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].resources.limits.memory}')
if [[ "$MEM_LIMIT" != "128Mi" ]]; then
    echo "ERROR: Memory limit should be 128Mi, got '$MEM_LIMIT'"
    exit 1
fi

CPU_LIMIT=$(kubectl get deployment web-app -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].resources.limits.cpu}')
if [[ "$CPU_LIMIT" != "200m" ]]; then
    echo "ERROR: CPU limit should be 200m, got '$CPU_LIMIT'"
    exit 1
fi

MEM_REQ=$(kubectl get deployment web-app -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].resources.requests.memory}')
if [[ "$MEM_REQ" != "64Mi" ]]; then
    echo "ERROR: Memory request should be 64Mi, got '$MEM_REQ'"
    exit 1
fi

CPU_REQ=$(kubectl get deployment web-app -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].resources.requests.cpu}')
if [[ "$CPU_REQ" != "100m" ]]; then
    echo "ERROR: CPU request should be 100m, got '$CPU_REQ'"
    exit 1
fi

# Check labels
VERSION_LABEL=$(kubectl get deployment web-app -n $NAMESPACE -o jsonpath='{.spec.template.metadata.labels.version}')
if [[ "$VERSION_LABEL" != "2.0" ]]; then
    echo "ERROR: version label should be '2.0', got '$VERSION_LABEL'"
    exit 1
fi

UPDATED_BY=$(kubectl get deployment web-app -n $NAMESPACE -o jsonpath='{.spec.template.metadata.labels.updated-by}')
if [[ "$UPDATED_BY" != "batch-update" ]]; then
    echo "ERROR: updated-by label should be 'batch-update'"
    exit 1
fi

# Check annotations
CHANGES=$(kubectl get deployment web-app -n $NAMESPACE -o jsonpath='{.metadata.annotations.batch-update\.k13d\.io/changes}')
if [[ -z "$CHANGES" ]]; then
    echo "ERROR: batch-update.k13d.io/changes annotation should be set"
    exit 1
fi

TIMESTAMP=$(kubectl get deployment web-app -n $NAMESPACE -o jsonpath='{.metadata.annotations.batch-update\.k13d\.io/timestamp}')
if [[ -z "$TIMESTAMP" ]]; then
    echo "ERROR: batch-update.k13d.io/timestamp annotation should be set"
    exit 1
fi

# Check that there aren't too many ReplicaSets (should be 2: old and new)
RS_COUNT=$(kubectl get rs -n $NAMESPACE -l app=web-app --no-headers | wc -l | tr -d ' ')
if [[ "$RS_COUNT" -gt 3 ]]; then
    echo "WARNING: Too many ReplicaSets ($RS_COUNT), changes may not have been batched"
fi

echo "--- Verification Successful! ---"
echo "Deployment pause/resume with batched changes completed."
exit 0
