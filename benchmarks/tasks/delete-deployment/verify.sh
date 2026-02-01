#!/bin/bash
set -e

# Check that old-deployment is deleted
if kubectl get deployment old-deployment -n benchmark &>/dev/null; then
    echo "FAIL: Deployment 'old-deployment' still exists"
    exit 1
fi

# Check that active-deployment still exists
if ! kubectl get deployment active-deployment -n benchmark &>/dev/null; then
    echo "FAIL: Deployment 'active-deployment' was incorrectly deleted"
    exit 1
fi

echo "PASS: Deployment 'old-deployment' successfully deleted"
exit 0
