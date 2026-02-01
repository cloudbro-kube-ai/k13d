#!/bin/bash
set -e

# Check if ConfigMap exists
if ! kubectl get configmap settings -n benchmark &>/dev/null; then
    echo "FAIL: ConfigMap 'settings' not found in namespace 'benchmark'"
    exit 1
fi

# Check MAX_CONNECTIONS is updated to 500
MAX_CONN=$(kubectl get configmap settings -n benchmark -o jsonpath='{.data.MAX_CONNECTIONS}')
if [[ "$MAX_CONN" != "500" ]]; then
    echo "FAIL: MAX_CONNECTIONS is '$MAX_CONN', expected '500'"
    exit 1
fi

# Verify other values are still intact
TIMEOUT=$(kubectl get configmap settings -n benchmark -o jsonpath='{.data.TIMEOUT}')
if [[ "$TIMEOUT" != "30" ]]; then
    echo "FAIL: TIMEOUT was incorrectly modified to '$TIMEOUT', expected '30'"
    exit 1
fi

echo "PASS: ConfigMap 'settings' updated correctly - MAX_CONNECTIONS is now '500'"
exit 0
