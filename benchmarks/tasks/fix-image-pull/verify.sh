#!/bin/bash
# Verifier script for fix-image-pull task

set -e

echo "Verifying fix-image-pull task..."

# Check if deployment exists
if ! kubectl get deployment image-app --namespace="${NAMESPACE}" &>/dev/null; then
    echo "ERROR: Deployment 'image-app' not found"
    exit 1
fi

# Wait for potential rollout
echo "Waiting for deployment rollout..."
kubectl rollout status deployment/image-app --namespace="${NAMESPACE}" --timeout=120s || true

# Check ready replicas
READY=$(kubectl get deployment image-app --namespace="${NAMESPACE}" -o jsonpath='{.status.readyReplicas}')
DESIRED=$(kubectl get deployment image-app --namespace="${NAMESPACE}" -o jsonpath='{.spec.replicas}')

if [ "$READY" != "$DESIRED" ] || [ -z "$READY" ]; then
    echo "ERROR: Deployment not fully ready. Ready: ${READY:-0}, Desired: ${DESIRED:-1}"
    echo "Pod status:"
    kubectl get pods -l app=image-app --namespace="${NAMESPACE}"
    exit 1
fi

# Check that image was changed to a valid one
IMAGE=$(kubectl get deployment image-app --namespace="${NAMESPACE}" -o jsonpath='{.spec.template.spec.containers[0].image}')
if [[ "$IMAGE" == *"nonexistent"* ]]; then
    echo "ERROR: Image was not changed. Still using invalid image: $IMAGE"
    exit 1
fi

# Verify pods are running (not in ImagePullBackOff)
POD_STATUS=$(kubectl get pods -l app=image-app --namespace="${NAMESPACE}" -o jsonpath='{.items[0].status.phase}')
if [ "$POD_STATUS" != "Running" ]; then
    echo "ERROR: Pod is not running. Status: $POD_STATUS"
    exit 1
fi

echo "Verification PASSED: Deployment 'image-app' is now running with image: $IMAGE"
exit 0
