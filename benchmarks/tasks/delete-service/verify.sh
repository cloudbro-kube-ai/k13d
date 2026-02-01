#!/bin/bash
set -e

# Check that old-service is deleted
if kubectl get service old-service -n benchmark &>/dev/null; then
    echo "FAIL: Service 'old-service' still exists"
    exit 1
fi

# Check that keep-service still exists
if ! kubectl get service keep-service -n benchmark &>/dev/null; then
    echo "FAIL: Service 'keep-service' was incorrectly deleted"
    exit 1
fi

echo "PASS: Service 'old-service' successfully deleted"
exit 0
