#!/bin/bash
set -euo pipefail

NAMESPACE="pdb-demo"

echo "Verifying pod-disruption-budget..."

# Check web-frontend-pdb
if ! kubectl get pdb web-frontend-pdb -n $NAMESPACE &>/dev/null; then
    echo "ERROR: PDB 'web-frontend-pdb' not found"
    exit 1
fi

WEB_MIN=$(kubectl get pdb web-frontend-pdb -n $NAMESPACE -o jsonpath='{.spec.minAvailable}')
if [[ "$WEB_MIN" != "2" ]]; then
    echo "ERROR: web-frontend-pdb minAvailable should be 2, got '$WEB_MIN'"
    exit 1
fi

WEB_SELECTOR=$(kubectl get pdb web-frontend-pdb -n $NAMESPACE -o jsonpath='{.spec.selector.matchLabels.app}')
if [[ "$WEB_SELECTOR" != "web-frontend" ]]; then
    echo "ERROR: web-frontend-pdb selector should match app=web-frontend"
    exit 1
fi

# Check api-backend-pdb
if ! kubectl get pdb api-backend-pdb -n $NAMESPACE &>/dev/null; then
    echo "ERROR: PDB 'api-backend-pdb' not found"
    exit 1
fi

API_MAX=$(kubectl get pdb api-backend-pdb -n $NAMESPACE -o jsonpath='{.spec.maxUnavailable}')
if [[ "$API_MAX" != "1" ]]; then
    echo "ERROR: api-backend-pdb maxUnavailable should be 1, got '$API_MAX'"
    exit 1
fi

API_SELECTOR=$(kubectl get pdb api-backend-pdb -n $NAMESPACE -o jsonpath='{.spec.selector.matchLabels.app}')
if [[ "$API_SELECTOR" != "api-backend" ]]; then
    echo "ERROR: api-backend-pdb selector should match app=api-backend"
    exit 1
fi

# Check cache-pdb
if ! kubectl get pdb cache-pdb -n $NAMESPACE &>/dev/null; then
    echo "ERROR: PDB 'cache-pdb' not found"
    exit 1
fi

CACHE_MAX=$(kubectl get pdb cache-pdb -n $NAMESPACE -o jsonpath='{.spec.maxUnavailable}')
if [[ "$CACHE_MAX" != "1" ]]; then
    echo "ERROR: cache-pdb maxUnavailable should be 1, got '$CACHE_MAX'"
    exit 1
fi

CACHE_SELECTOR=$(kubectl get pdb cache-pdb -n $NAMESPACE -o jsonpath='{.spec.selector.matchLabels.app}')
if [[ "$CACHE_SELECTOR" != "cache" ]]; then
    echo "ERROR: cache-pdb selector should match app=cache"
    exit 1
fi

# Check batch-worker-pdb
if ! kubectl get pdb batch-worker-pdb -n $NAMESPACE &>/dev/null; then
    echo "ERROR: PDB 'batch-worker-pdb' not found"
    exit 1
fi

BATCH_MAX=$(kubectl get pdb batch-worker-pdb -n $NAMESPACE -o jsonpath='{.spec.maxUnavailable}')
if [[ "$BATCH_MAX" != "50%" ]]; then
    echo "ERROR: batch-worker-pdb maxUnavailable should be 50%, got '$BATCH_MAX'"
    exit 1
fi

BATCH_SELECTOR=$(kubectl get pdb batch-worker-pdb -n $NAMESPACE -o jsonpath='{.spec.selector.matchLabels.app}')
if [[ "$BATCH_SELECTOR" != "batch-worker" ]]; then
    echo "ERROR: batch-worker-pdb selector should match app=batch-worker"
    exit 1
fi

# Verify PDBs are protecting pods (check status)
WEB_ALLOWED=$(kubectl get pdb web-frontend-pdb -n $NAMESPACE -o jsonpath='{.status.disruptionsAllowed}')
if [[ "$WEB_ALLOWED" -lt 1 ]]; then
    echo "WARNING: web-frontend-pdb allows no disruptions (may need more replicas)"
fi

echo "--- Verification Successful! ---"
echo "Pod Disruption Budgets configured correctly."
exit 0
