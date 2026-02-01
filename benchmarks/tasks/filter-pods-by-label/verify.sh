#!/bin/bash
set -e

# Check if output file exists
if [[ ! -f /tmp/frontend-pods.txt ]]; then
    echo "FAIL: Output file '/tmp/frontend-pods.txt' not found"
    exit 1
fi

# Read the file content and check for frontend pods
CONTENT=$(cat /tmp/frontend-pods.txt)

# Check that frontend-1 and frontend-2 are in the file
if [[ ! "$CONTENT" =~ "frontend-1" ]]; then
    echo "FAIL: 'frontend-1' not found in output file"
    exit 1
fi

if [[ ! "$CONTENT" =~ "frontend-2" ]]; then
    echo "FAIL: 'frontend-2' not found in output file"
    exit 1
fi

# Check that backend pods are NOT in the file
if [[ "$CONTENT" =~ "backend-1" ]] || [[ "$CONTENT" =~ "backend-2" ]]; then
    echo "FAIL: Backend pods should not be in the output (they don't have tier=frontend label)"
    exit 1
fi

echo "PASS: Successfully listed frontend pods to /tmp/frontend-pods.txt"
exit 0
