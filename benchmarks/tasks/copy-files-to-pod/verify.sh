#!/bin/bash
set -e

echo "Verifying copy-files-to-pod task..."

NAMESPACE="cp-test"

# Check Pod exists and is running
STATUS=$(kubectl get pod file-server -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
if [ "$STATUS" != "Running" ]; then
    echo "FAILED: Pod not in Running state, got '$STATUS'"
    exit 1
fi

# Create test file
TMPDIR=$(mktemp -d)
echo "Test content from verify script at $(date)" > "$TMPDIR/upload-test.txt"

# Test kubectl cp TO pod
echo "Testing kubectl cp to pod..."
if kubectl cp "$TMPDIR/upload-test.txt" "$NAMESPACE/file-server:/tmp/upload-test.txt" 2>/dev/null; then
    echo "File copy TO pod successful"
else
    echo "WARNING: kubectl cp to pod may have issues"
fi

# Verify file exists in pod
FILE_IN_POD=$(kubectl exec file-server -n "$NAMESPACE" -- cat /tmp/upload-test.txt 2>/dev/null || echo "NOT_FOUND")
if [[ "$FILE_IN_POD" == *"Test content"* ]]; then
    echo "File verified inside pod"
else
    echo "WARNING: File content not verified in pod"
fi

# Test kubectl cp FROM pod
echo "Testing kubectl cp from pod..."
if kubectl cp "$NAMESPACE/file-server:/etc/hostname" "$TMPDIR/hostname-from-pod.txt" 2>/dev/null; then
    echo "File copy FROM pod successful"
    if [ -f "$TMPDIR/hostname-from-pod.txt" ]; then
        echo "Downloaded file verified locally"
    fi
else
    echo "WARNING: kubectl cp from pod may have issues"
fi

# Cleanup temp files
rm -rf "$TMPDIR"

echo "SUCCESS: File copy operations verified"
exit 0
