#!/bin/bash
set -e

# Check if output file exists
if [[ ! -f /tmp/namespaces.txt ]]; then
    echo "FAIL: Output file '/tmp/namespaces.txt' not found"
    exit 1
fi

# Read the file content
CONTENT=$(cat /tmp/namespaces.txt)

# Check that default system namespaces are listed
if [[ ! "$CONTENT" =~ "default" ]]; then
    echo "FAIL: 'default' namespace not found in output file"
    exit 1
fi

if [[ ! "$CONTENT" =~ "kube-system" ]]; then
    echo "FAIL: 'kube-system' namespace not found in output file"
    exit 1
fi

# Check that our test namespaces are listed
if [[ ! "$CONTENT" =~ "test-ns-alpha" ]]; then
    echo "FAIL: 'test-ns-alpha' namespace not found in output file"
    exit 1
fi

if [[ ! "$CONTENT" =~ "test-ns-beta" ]]; then
    echo "FAIL: 'test-ns-beta' namespace not found in output file"
    exit 1
fi

echo "PASS: All namespaces listed correctly in /tmp/namespaces.txt"
exit 0
