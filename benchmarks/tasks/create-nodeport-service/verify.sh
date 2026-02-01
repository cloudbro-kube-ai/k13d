#!/bin/bash
set -e

# Check if Service exists
if ! kubectl get service web-nodeport -n benchmark &>/dev/null; then
    echo "FAIL: Service 'web-nodeport' not found in namespace 'benchmark'"
    exit 1
fi

# Check Service type
SVC_TYPE=$(kubectl get service web-nodeport -n benchmark -o jsonpath='{.spec.type}')
if [[ "$SVC_TYPE" != "NodePort" ]]; then
    echo "FAIL: Service type is '$SVC_TYPE', expected 'NodePort'"
    exit 1
fi

# Check selector
SELECTOR=$(kubectl get service web-nodeport -n benchmark -o jsonpath='{.spec.selector.app}')
if [[ "$SELECTOR" != "web" ]]; then
    echo "FAIL: Selector 'app' is '$SELECTOR', expected 'web'"
    exit 1
fi

# Check port
PORT=$(kubectl get service web-nodeport -n benchmark -o jsonpath='{.spec.ports[0].port}')
if [[ "$PORT" != "80" ]]; then
    echo "FAIL: Port is '$PORT', expected '80'"
    exit 1
fi

# Check target port
TARGET_PORT=$(kubectl get service web-nodeport -n benchmark -o jsonpath='{.spec.ports[0].targetPort}')
if [[ "$TARGET_PORT" != "8080" ]]; then
    echo "FAIL: Target port is '$TARGET_PORT', expected '8080'"
    exit 1
fi

# Check NodePort
NODE_PORT=$(kubectl get service web-nodeport -n benchmark -o jsonpath='{.spec.ports[0].nodePort}')
if [[ "$NODE_PORT" != "30080" ]]; then
    echo "FAIL: NodePort is '$NODE_PORT', expected '30080'"
    exit 1
fi

echo "PASS: NodePort Service 'web-nodeport' created correctly"
exit 0
