#!/bin/bash
# Verifier script for create-secret-env task

set -e

echo "Verifying create-secret-env task..."

# Check if secret exists
if ! kubectl get secret app-secrets --namespace="${NAMESPACE}" &>/dev/null; then
    echo "ERROR: Secret 'app-secrets' not found"
    exit 1
fi

# Check if secret has the required keys
DB_URL=$(kubectl get secret app-secrets --namespace="${NAMESPACE}" -o jsonpath='{.data.DATABASE_URL}' | base64 -d 2>/dev/null || true)
API_KEY=$(kubectl get secret app-secrets --namespace="${NAMESPACE}" -o jsonpath='{.data.API_KEY}' | base64 -d 2>/dev/null || true)

if [ -z "$DB_URL" ]; then
    echo "ERROR: Secret missing 'DATABASE_URL' key"
    exit 1
fi

if [ -z "$API_KEY" ]; then
    echo "ERROR: Secret missing 'API_KEY' key"
    exit 1
fi

# Check if pod exists and is running
if ! kubectl get pod app-pod --namespace="${NAMESPACE}" &>/dev/null; then
    echo "ERROR: Pod 'app-pod' not found"
    exit 1
fi

STATUS=$(kubectl get pod app-pod --namespace="${NAMESPACE}" -o jsonpath='{.status.phase}')
if [ "$STATUS" != "Running" ]; then
    echo "ERROR: Pod is not running. Current status: $STATUS"
    exit 1
fi

# Check if pod has environment variables from secret
ENV_CHECK=$(kubectl exec app-pod --namespace="${NAMESPACE}" -- env 2>/dev/null | grep -E "DATABASE_URL|API_KEY" || true)
if [ -z "$ENV_CHECK" ]; then
    echo "ERROR: Pod does not have secret environment variables injected"
    exit 1
fi

echo "Verification PASSED: Secret created and injected as environment variables"
exit 0
