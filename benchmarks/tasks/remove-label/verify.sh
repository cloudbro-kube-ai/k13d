#!/bin/bash
set -e

# Check if pod exists
if ! kubectl get pod legacy-pod -n benchmark &>/dev/null; then
    echo "FAIL: Pod 'legacy-pod' not found in namespace 'benchmark'"
    exit 1
fi

# Check that deprecated label is removed
DEPRECATED_LABEL=$(kubectl get pod legacy-pod -n benchmark -o jsonpath='{.metadata.labels.deprecated}')
if [[ -n "$DEPRECATED_LABEL" ]]; then
    echo "FAIL: Label 'deprecated' still exists with value '$DEPRECATED_LABEL'"
    exit 1
fi

# Check that other labels are still present
APP_LABEL=$(kubectl get pod legacy-pod -n benchmark -o jsonpath='{.metadata.labels.app}')
if [[ "$APP_LABEL" != "legacy" ]]; then
    echo "FAIL: Label 'app' was incorrectly modified or removed"
    exit 1
fi

echo "PASS: Label 'deprecated' successfully removed from pod 'legacy-pod'"
exit 0
