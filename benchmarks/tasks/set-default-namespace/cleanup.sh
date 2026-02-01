#!/bin/bash
set -e

# Restore original namespace if saved
if [[ -f /tmp/original-namespace.txt ]]; then
    ORIGINAL_NS=$(cat /tmp/original-namespace.txt)
    if [[ -n "$ORIGINAL_NS" ]]; then
        kubectl config set-context --current --namespace="$ORIGINAL_NS"
    else
        # If original was empty, set to default
        kubectl config set-context --current --namespace=default
    fi
    rm -f /tmp/original-namespace.txt
else
    kubectl config set-context --current --namespace=default
fi

# Delete the development namespace
kubectl delete namespace development --ignore-not-found=true

echo "Cleanup complete"
