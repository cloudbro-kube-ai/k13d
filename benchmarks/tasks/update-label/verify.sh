#!/bin/bash
set -e

# Check if pod exists
if ! kubectl get pod app-pod -n benchmark &>/dev/null; then
    echo "FAIL: Pod 'app-pod' not found in namespace 'benchmark'"
    exit 1
fi

# Check that version label is updated to v2
VERSION_LABEL=$(kubectl get pod app-pod -n benchmark -o jsonpath='{.metadata.labels.version}')
if [[ "$VERSION_LABEL" != "v2" ]]; then
    echo "FAIL: Label 'version' is '$VERSION_LABEL', expected 'v2'"
    exit 1
fi

# Check that app label is still present
APP_LABEL=$(kubectl get pod app-pod -n benchmark -o jsonpath='{.metadata.labels.app}')
if [[ "$APP_LABEL" != "myapp" ]]; then
    echo "FAIL: Label 'app' was incorrectly modified"
    exit 1
fi

echo "PASS: Label 'version' successfully updated to 'v2' on pod 'app-pod'"
exit 0
