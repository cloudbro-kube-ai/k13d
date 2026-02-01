#!/bin/bash
set -e

# Check the current default namespace
CURRENT_NS=$(kubectl config view --minify -o jsonpath='{..namespace}')

if [[ "$CURRENT_NS" != "development" ]]; then
    echo "FAIL: Default namespace is '$CURRENT_NS', expected 'development'"
    exit 1
fi

echo "PASS: Default namespace set to 'development'"
exit 0
