#!/bin/bash
set -euo pipefail

NAMESPACE="microservices"

echo "Verifying ingress-policy..."

# Check NetworkPolicy api-ingress exists
if ! kubectl get networkpolicy api-ingress -n $NAMESPACE &>/dev/null; then
    echo "ERROR: NetworkPolicy 'api-ingress' not found"
    exit 1
fi

# Check NetworkPolicy db-ingress exists
if ! kubectl get networkpolicy db-ingress -n $NAMESPACE &>/dev/null; then
    echo "ERROR: NetworkPolicy 'db-ingress' not found"
    exit 1
fi

# Verify api-ingress targets app=api
API_SELECTOR=$(kubectl get networkpolicy api-ingress -n $NAMESPACE -o jsonpath='{.spec.podSelector.matchLabels.app}')
if [[ "$API_SELECTOR" != "api" ]]; then
    echo "ERROR: api-ingress doesn't target app=api pods"
    exit 1
fi

# Verify api-ingress allows from tier=frontend
API_FROM_FRONTEND=$(kubectl get networkpolicy api-ingress -n $NAMESPACE -o json | jq -r '.spec.ingress[].from[] | select(.podSelector.matchLabels.tier == "frontend") | .podSelector.matchLabels.tier')
if [[ "$API_FROM_FRONTEND" != "frontend" ]]; then
    echo "ERROR: api-ingress doesn't allow from tier=frontend"
    exit 1
fi

# Verify api-ingress has namespace selector for access=api-allowed
API_NS_SELECTOR=$(kubectl get networkpolicy api-ingress -n $NAMESPACE -o json | jq -r '.spec.ingress[].from[] | select(.namespaceSelector != null) | .namespaceSelector.matchLabels.access')
if [[ "$API_NS_SELECTOR" != "api-allowed" ]]; then
    echo "ERROR: api-ingress doesn't allow from namespaces with access=api-allowed"
    exit 1
fi

# Verify api-ingress allows port 8080
API_PORT=$(kubectl get networkpolicy api-ingress -n $NAMESPACE -o json | jq -r '.spec.ingress[].ports[]? | select(.port == 8080) | .port')
if [[ "$API_PORT" != "8080" ]]; then
    echo "ERROR: api-ingress doesn't specify port 8080"
    exit 1
fi

# Verify db-ingress targets app=db
DB_SELECTOR=$(kubectl get networkpolicy db-ingress -n $NAMESPACE -o jsonpath='{.spec.podSelector.matchLabels.app}')
if [[ "$DB_SELECTOR" != "db" ]]; then
    echo "ERROR: db-ingress doesn't target app=db pods"
    exit 1
fi

# Verify db-ingress only allows from app=api
DB_FROM_API=$(kubectl get networkpolicy db-ingress -n $NAMESPACE -o json | jq -r '.spec.ingress[].from[] | select(.podSelector.matchLabels.app == "api") | .podSelector.matchLabels.app')
if [[ "$DB_FROM_API" != "api" ]]; then
    echo "ERROR: db-ingress doesn't restrict to app=api only"
    exit 1
fi

# Verify db-ingress port 5432
DB_PORT=$(kubectl get networkpolicy db-ingress -n $NAMESPACE -o json | jq -r '.spec.ingress[].ports[]? | select(.port == 5432) | .port')
if [[ "$DB_PORT" != "5432" ]]; then
    echo "ERROR: db-ingress doesn't specify port 5432"
    exit 1
fi

# Check external-service namespace exists with correct label
if ! kubectl get namespace external-service &>/dev/null; then
    echo "ERROR: Namespace 'external-service' not found"
    exit 1
fi

EXT_LABEL=$(kubectl get namespace external-service -o jsonpath='{.metadata.labels.access}')
if [[ "$EXT_LABEL" != "api-allowed" ]]; then
    echo "ERROR: Namespace 'external-service' doesn't have label 'access=api-allowed'"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Ingress network policies are correctly configured."
exit 0
