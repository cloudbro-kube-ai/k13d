#!/bin/bash
set -e

# Check if Secret exists
if ! kubectl get secret db-credentials -n benchmark &>/dev/null; then
    echo "FAIL: Secret 'db-credentials' not found in namespace 'benchmark'"
    exit 1
fi

# Decode and check username
USERNAME=$(kubectl get secret db-credentials -n benchmark -o jsonpath='{.data.username}' | base64 -d)
if [[ "$USERNAME" != "admin" ]]; then
    echo "FAIL: username is '$USERNAME', expected 'admin'"
    exit 1
fi

# Decode and check password
PASSWORD=$(kubectl get secret db-credentials -n benchmark -o jsonpath='{.data.password}' | base64 -d)
if [[ "$PASSWORD" != "secretpass123" ]]; then
    echo "FAIL: password is '$PASSWORD', expected 'secretpass123'"
    exit 1
fi

echo "PASS: Secret 'db-credentials' created correctly with username and password"
exit 0
