#!/bin/bash
set -e

# Check if Service exists
if ! kubectl get service api-lb -n benchmark &>/dev/null; then
    echo "FAIL: Service 'api-lb' not found in namespace 'benchmark'"
    exit 1
fi

# Check Service type
SVC_TYPE=$(kubectl get service api-lb -n benchmark -o jsonpath='{.spec.type}')
if [[ "$SVC_TYPE" != "LoadBalancer" ]]; then
    echo "FAIL: Service type is '$SVC_TYPE', expected 'LoadBalancer'"
    exit 1
fi

# Check selector
SELECTOR=$(kubectl get service api-lb -n benchmark -o jsonpath='{.spec.selector.app}')
if [[ "$SELECTOR" != "api" ]]; then
    echo "FAIL: Selector 'app' is '$SELECTOR', expected 'api'"
    exit 1
fi

# Check port
PORT=$(kubectl get service api-lb -n benchmark -o jsonpath='{.spec.ports[0].port}')
if [[ "$PORT" != "443" ]]; then
    echo "FAIL: Port is '$PORT', expected '443'"
    exit 1
fi

# Check target port
TARGET_PORT=$(kubectl get service api-lb -n benchmark -o jsonpath='{.spec.ports[0].targetPort}')
if [[ "$TARGET_PORT" != "8443" ]]; then
    echo "FAIL: Target port is '$TARGET_PORT', expected '8443'"
    exit 1
fi

echo "PASS: LoadBalancer Service 'api-lb' created correctly"
exit 0
