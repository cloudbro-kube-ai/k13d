#!/bin/bash
set -euo pipefail

NAMESPACE="secure-apps"

echo "Verifying egress-policy..."

# Check NetworkPolicy worker-egress exists
if ! kubectl get networkpolicy worker-egress -n $NAMESPACE &>/dev/null; then
    echo "ERROR: NetworkPolicy 'worker-egress' not found"
    exit 1
fi

# Check NetworkPolicy api-egress exists
if ! kubectl get networkpolicy api-egress -n $NAMESPACE &>/dev/null; then
    echo "ERROR: NetworkPolicy 'api-egress' not found"
    exit 1
fi

# Verify worker-egress targets security=high
WORKER_SELECTOR=$(kubectl get networkpolicy worker-egress -n $NAMESPACE -o jsonpath='{.spec.podSelector.matchLabels.security}')
if [[ "$WORKER_SELECTOR" != "high" ]]; then
    echo "ERROR: worker-egress doesn't target security=high pods"
    exit 1
fi

# Verify worker-egress has Egress policy type
WORKER_TYPES=$(kubectl get networkpolicy worker-egress -n $NAMESPACE -o jsonpath='{.spec.policyTypes[*]}')
if [[ ! "$WORKER_TYPES" =~ "Egress" ]]; then
    echo "ERROR: worker-egress doesn't have Egress policy type"
    exit 1
fi

# Verify worker-egress allows to app=api
WORKER_TO_API=$(kubectl get networkpolicy worker-egress -n $NAMESPACE -o json | jq -r '.spec.egress[]?.to[]? | select(.podSelector.matchLabels.app == "api") | .podSelector.matchLabels.app')
if [[ "$WORKER_TO_API" != "api" ]]; then
    echo "ERROR: worker-egress doesn't allow egress to app=api"
    exit 1
fi

# Verify worker-egress allows DNS (port 53)
WORKER_DNS=$(kubectl get networkpolicy worker-egress -n $NAMESPACE -o json | jq -r '.spec.egress[]?.ports[]? | select(.port == 53) | .port')
if [[ -z "$WORKER_DNS" ]]; then
    echo "ERROR: worker-egress doesn't allow DNS (port 53)"
    exit 1
fi

# Verify api-egress targets app=api
API_SELECTOR=$(kubectl get networkpolicy api-egress -n $NAMESPACE -o jsonpath='{.spec.podSelector.matchLabels.app}')
if [[ "$API_SELECTOR" != "api" ]]; then
    echo "ERROR: api-egress doesn't target app=api pods"
    exit 1
fi

# Verify api-egress has Egress policy type
API_TYPES=$(kubectl get networkpolicy api-egress -n $NAMESPACE -o jsonpath='{.spec.policyTypes[*]}')
if [[ ! "$API_TYPES" =~ "Egress" ]]; then
    echo "ERROR: api-egress doesn't have Egress policy type"
    exit 1
fi

# Verify api-egress allows port 443
API_HTTPS=$(kubectl get networkpolicy api-egress -n $NAMESPACE -o json | jq -r '.spec.egress[]?.ports[]? | select(.port == 443) | .port')
if [[ "$API_HTTPS" != "443" ]]; then
    echo "ERROR: api-egress doesn't allow HTTPS (port 443)"
    exit 1
fi

# Verify api-egress allows DNS
API_DNS=$(kubectl get networkpolicy api-egress -n $NAMESPACE -o json | jq -r '.spec.egress[]?.ports[]? | select(.port == 53) | .port')
if [[ -z "$API_DNS" ]]; then
    echo "ERROR: api-egress doesn't allow DNS (port 53)"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Egress network policies are correctly configured."
exit 0
