#!/bin/bash
set -e

# Check if Service exists
if ! kubectl get service backend-svc -n benchmark &>/dev/null; then
    echo "FAIL: Service 'backend-svc' not found in namespace 'benchmark'"
    exit 1
fi

# Check port is updated to 8080
PORT=$(kubectl get service backend-svc -n benchmark -o jsonpath='{.spec.ports[0].port}')
if [[ "$PORT" != "8080" ]]; then
    echo "FAIL: Port is '$PORT', expected '8080'"
    exit 1
fi

# Check target port is still 80
TARGET_PORT=$(kubectl get service backend-svc -n benchmark -o jsonpath='{.spec.ports[0].targetPort}')
if [[ "$TARGET_PORT" != "80" ]]; then
    echo "FAIL: Target port is '$TARGET_PORT', expected '80'"
    exit 1
fi

echo "PASS: Service 'backend-svc' updated correctly - port is now 8080"
exit 0
