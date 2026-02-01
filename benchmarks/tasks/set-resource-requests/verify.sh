#!/bin/bash
set -e

echo "Verifying set-resource-requests task..."

NAMESPACE="resource-req-test"

# Check Deployment exists
if ! kubectl get deployment web-app -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Deployment 'web-app' not found"
    exit 1
fi

# Check replicas
REPLICAS=$(kubectl get deployment web-app -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
if [ "$REPLICAS" != "2" ]; then
    echo "FAILED: Expected 2 replicas, got '$REPLICAS'"
    exit 1
fi

# Check CPU request
CPU_REQUEST=$(kubectl get deployment web-app -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].resources.requests.cpu}' 2>/dev/null || echo "")
if [[ "$CPU_REQUEST" != "100m" ]]; then
    echo "FAILED: CPU request should be '100m', got '$CPU_REQUEST'"
    exit 1
fi

# Check Memory request
MEM_REQUEST=$(kubectl get deployment web-app -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].resources.requests.memory}' 2>/dev/null || echo "")
if [[ "$MEM_REQUEST" != "128Mi" ]]; then
    echo "FAILED: Memory request should be '128Mi', got '$MEM_REQUEST'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get deployment web-app -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || echo "")
if [[ "$IMAGE" != *"nginx"* ]]; then
    echo "FAILED: Image should be nginx, got '$IMAGE'"
    exit 1
fi

# Wait for deployment to be ready
if ! kubectl rollout status deployment/web-app -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Deployment did not become ready in time"
    exit 1
fi

echo "SUCCESS: Resource requests correctly configured"
exit 0
