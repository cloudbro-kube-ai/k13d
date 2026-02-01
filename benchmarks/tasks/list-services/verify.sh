#!/bin/bash
set -e

# Check if output file exists
if [[ ! -f /tmp/services.txt ]]; then
    echo "FAIL: Output file '/tmp/services.txt' not found"
    exit 1
fi

# Read the file content
CONTENT=$(cat /tmp/services.txt)

# Check that all Services are listed
if [[ ! "$CONTENT" =~ "frontend-svc" ]]; then
    echo "FAIL: 'frontend-svc' not found in output file"
    exit 1
fi

if [[ ! "$CONTENT" =~ "backend-svc" ]]; then
    echo "FAIL: 'backend-svc' not found in output file"
    exit 1
fi

if [[ ! "$CONTENT" =~ "database-svc" ]]; then
    echo "FAIL: 'database-svc' not found in output file"
    exit 1
fi

echo "PASS: All Services listed correctly in /tmp/services.txt"
exit 0
