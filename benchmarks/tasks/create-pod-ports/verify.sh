#!/bin/bash
set -e

# Check if pod exists
if ! kubectl get pod ports-pod -n benchmark &>/dev/null; then
    echo "FAIL: Pod 'ports-pod' not found in namespace 'benchmark'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get pod ports-pod -n benchmark -o jsonpath='{.spec.containers[0].image}')
if [[ "$IMAGE" != "nginx:alpine" ]]; then
    echo "FAIL: Pod image is '$IMAGE', expected 'nginx:alpine'"
    exit 1
fi

# Check port 80 exists
PORT_80=$(kubectl get pod ports-pod -n benchmark -o jsonpath='{.spec.containers[0].ports[?(@.containerPort==80)].containerPort}')
if [[ "$PORT_80" != "80" ]]; then
    echo "FAIL: Container port 80 not found"
    exit 1
fi

# Check port 443 exists
PORT_443=$(kubectl get pod ports-pod -n benchmark -o jsonpath='{.spec.containers[0].ports[?(@.containerPort==443)].containerPort}')
if [[ "$PORT_443" != "443" ]]; then
    echo "FAIL: Container port 443 not found"
    exit 1
fi

# Check port names
PORT_80_NAME=$(kubectl get pod ports-pod -n benchmark -o jsonpath='{.spec.containers[0].ports[?(@.containerPort==80)].name}')
PORT_443_NAME=$(kubectl get pod ports-pod -n benchmark -o jsonpath='{.spec.containers[0].ports[?(@.containerPort==443)].name}')

if [[ "$PORT_80_NAME" != "http" ]]; then
    echo "FAIL: Port 80 name is '$PORT_80_NAME', expected 'http'"
    exit 1
fi

if [[ "$PORT_443_NAME" != "https" ]]; then
    echo "FAIL: Port 443 name is '$PORT_443_NAME', expected 'https'"
    exit 1
fi

echo "PASS: Pod 'ports-pod' created correctly with ports 80 (http) and 443 (https)"
exit 0
