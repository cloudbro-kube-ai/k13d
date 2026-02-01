#!/bin/bash
set -e

# Check if pod exists
if ! kubectl get pod target-pod -n benchmark &>/dev/null; then
    echo "FAIL: Pod 'target-pod' not found in namespace 'benchmark'"
    exit 1
fi

# Check for the environment label
LABEL_VALUE=$(kubectl get pod target-pod -n benchmark -o jsonpath='{.metadata.labels.environment}')
if [[ "$LABEL_VALUE" != "staging" ]]; then
    echo "FAIL: Label 'environment' is '$LABEL_VALUE', expected 'staging'"
    exit 1
fi

echo "PASS: Label 'environment=staging' successfully added to pod 'target-pod'"
exit 0
