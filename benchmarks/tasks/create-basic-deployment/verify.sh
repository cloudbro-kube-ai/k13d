#!/bin/bash
set -e

# Check if Deployment exists
if ! kubectl get deployment nginx-deployment -n benchmark &>/dev/null; then
    echo "FAIL: Deployment 'nginx-deployment' not found in namespace 'benchmark'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get deployment nginx-deployment -n benchmark -o jsonpath='{.spec.template.spec.containers[0].image}')
if [[ "$IMAGE" != "nginx:alpine" ]]; then
    echo "FAIL: Image is '$IMAGE', expected 'nginx:alpine'"
    exit 1
fi

# Check replicas
REPLICAS=$(kubectl get deployment nginx-deployment -n benchmark -o jsonpath='{.spec.replicas}')
if [[ "$REPLICAS" != "3" ]]; then
    echo "FAIL: Replicas is '$REPLICAS', expected '3'"
    exit 1
fi

# Check selector label
SELECTOR=$(kubectl get deployment nginx-deployment -n benchmark -o jsonpath='{.spec.selector.matchLabels.app}')
if [[ "$SELECTOR" != "nginx" ]]; then
    echo "FAIL: Selector 'app' is '$SELECTOR', expected 'nginx'"
    exit 1
fi

# Check pod template label
POD_LABEL=$(kubectl get deployment nginx-deployment -n benchmark -o jsonpath='{.spec.template.metadata.labels.app}')
if [[ "$POD_LABEL" != "nginx" ]]; then
    echo "FAIL: Pod template label 'app' is '$POD_LABEL', expected 'nginx'"
    exit 1
fi

echo "PASS: Deployment 'nginx-deployment' created correctly with 3 replicas and nginx:alpine image"
exit 0
