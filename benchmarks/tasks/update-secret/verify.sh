#!/bin/bash
set -e

# Check if Secret exists
if ! kubectl get secret api-secret -n benchmark &>/dev/null; then
    echo "FAIL: Secret 'api-secret' not found in namespace 'benchmark'"
    exit 1
fi

# Decode and check API_KEY
API_KEY=$(kubectl get secret api-secret -n benchmark -o jsonpath='{.data.API_KEY}' | base64 -d)
if [[ "$API_KEY" != "new-api-key-2024" ]]; then
    echo "FAIL: API_KEY is '$API_KEY', expected 'new-api-key-2024'"
    exit 1
fi

# Verify API_SECRET is still intact
API_SECRET=$(kubectl get secret api-secret -n benchmark -o jsonpath='{.data.API_SECRET}' | base64 -d)
if [[ "$API_SECRET" != "keep-this-value" ]]; then
    echo "FAIL: API_SECRET was incorrectly modified to '$API_SECRET'"
    exit 1
fi

echo "PASS: Secret 'api-secret' updated correctly - API_KEY is now 'new-api-key-2024'"
exit 0
