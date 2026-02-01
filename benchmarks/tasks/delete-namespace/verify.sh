#!/bin/bash
set -e

# Check that deprecated-ns is deleted (or in Terminating state)
NS_STATUS=$(kubectl get namespace deprecated-ns -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
if [[ "$NS_STATUS" != "NotFound" && "$NS_STATUS" != "Terminating" ]]; then
    echo "FAIL: Namespace 'deprecated-ns' still exists with status '$NS_STATUS'"
    exit 1
fi

# Check that keep-ns still exists
if ! kubectl get namespace keep-ns &>/dev/null; then
    echo "FAIL: Namespace 'keep-ns' was incorrectly deleted"
    exit 1
fi

echo "PASS: Namespace 'deprecated-ns' successfully deleted"
exit 0
