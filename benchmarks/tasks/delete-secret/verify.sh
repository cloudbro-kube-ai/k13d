#!/bin/bash
set -e

# Check that deprecated-secret is deleted
if kubectl get secret deprecated-secret -n benchmark &>/dev/null; then
    echo "FAIL: Secret 'deprecated-secret' still exists"
    exit 1
fi

# Check that active-secret still exists
if ! kubectl get secret active-secret -n benchmark &>/dev/null; then
    echo "FAIL: Secret 'active-secret' was incorrectly deleted"
    exit 1
fi

echo "PASS: Secret 'deprecated-secret' successfully deleted"
exit 0
