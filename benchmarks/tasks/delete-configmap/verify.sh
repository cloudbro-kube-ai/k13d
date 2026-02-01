#!/bin/bash
set -e

# Check that old-config is deleted
if kubectl get configmap old-config -n benchmark &>/dev/null; then
    echo "FAIL: ConfigMap 'old-config' still exists"
    exit 1
fi

# Check that keep-config still exists
if ! kubectl get configmap keep-config -n benchmark &>/dev/null; then
    echo "FAIL: ConfigMap 'keep-config' was incorrectly deleted"
    exit 1
fi

echo "PASS: ConfigMap 'old-config' successfully deleted"
exit 0
