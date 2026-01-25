#!/bin/bash
set -e

echo "Verifying fix-pending-pod task..."

# Check if pod exists and is Running
STATUS=$(kubectl get pod homepage-pod -n homepage-ns -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")

if [ "$STATUS" = "Running" ]; then
    echo "SUCCESS: Pod is now in Running state"
    exit 0
else
    echo "FAILED: Pod is in '$STATUS' state, expected 'Running'"
    kubectl describe pod homepage-pod -n homepage-ns 2>/dev/null || true
    exit 1
fi
