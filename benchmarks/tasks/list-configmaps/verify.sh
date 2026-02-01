#!/bin/bash
set -e

# Check if output file exists
if [[ ! -f /tmp/configmaps.txt ]]; then
    echo "FAIL: Output file '/tmp/configmaps.txt' not found"
    exit 1
fi

# Read the file content
CONTENT=$(cat /tmp/configmaps.txt)

# Check that all ConfigMaps are listed
if [[ ! "$CONTENT" =~ "config-alpha" ]]; then
    echo "FAIL: 'config-alpha' not found in output file"
    exit 1
fi

if [[ ! "$CONTENT" =~ "config-beta" ]]; then
    echo "FAIL: 'config-beta' not found in output file"
    exit 1
fi

if [[ ! "$CONTENT" =~ "config-gamma" ]]; then
    echo "FAIL: 'config-gamma' not found in output file"
    exit 1
fi

echo "PASS: All ConfigMaps listed correctly in /tmp/configmaps.txt"
exit 0
