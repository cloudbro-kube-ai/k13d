#!/bin/bash
set -euo pipefail

NAMESPACE="production"

echo "Verifying deny-all-policy..."

# Check default-deny-all exists
if ! kubectl get networkpolicy default-deny-all -n $NAMESPACE &>/dev/null; then
    echo "ERROR: NetworkPolicy 'default-deny-all' not found"
    exit 1
fi

# Verify default-deny has empty podSelector (applies to all pods)
POD_SELECTOR=$(kubectl get networkpolicy default-deny-all -n $NAMESPACE -o json | jq -r '.spec.podSelector | keys | length')
if [[ "$POD_SELECTOR" != "0" ]]; then
    echo "ERROR: default-deny-all should have empty podSelector to apply to all pods"
    exit 1
fi

# Verify policy types include both Ingress and Egress
POLICY_TYPES=$(kubectl get networkpolicy default-deny-all -n $NAMESPACE -o jsonpath='{.spec.policyTypes[*]}')
if [[ ! "$POLICY_TYPES" =~ "Ingress" ]] || [[ ! "$POLICY_TYPES" =~ "Egress" ]]; then
    echo "ERROR: default-deny-all should have both Ingress and Egress policy types"
    exit 1
fi

# Verify default-deny has no ingress/egress rules (empty = deny all)
INGRESS_RULES=$(kubectl get networkpolicy default-deny-all -n $NAMESPACE -o json | jq -r '.spec.ingress // empty | length')
EGRESS_RULES=$(kubectl get networkpolicy default-deny-all -n $NAMESPACE -o json | jq -r '.spec.egress // empty | length')
if [[ -n "$INGRESS_RULES" && "$INGRESS_RULES" != "0" ]] || [[ -n "$EGRESS_RULES" && "$EGRESS_RULES" != "0" ]]; then
    echo "ERROR: default-deny-all should have no ingress/egress rules (deny all)"
    exit 1
fi

# Check allow-web-to-app exists
if ! kubectl get networkpolicy allow-web-to-app -n $NAMESPACE &>/dev/null; then
    echo "ERROR: NetworkPolicy 'allow-web-to-app' not found"
    exit 1
fi

# Verify allow-web-to-app configuration
WEB_TO_APP_FROM=$(kubectl get networkpolicy allow-web-to-app -n $NAMESPACE -o json | jq -r '.spec.ingress[]?.from[]?.podSelector.matchLabels.tier // empty')
WEB_TO_APP_TARGET=$(kubectl get networkpolicy allow-web-to-app -n $NAMESPACE -o jsonpath='{.spec.podSelector.matchLabels.tier}')
WEB_TO_APP_PORT=$(kubectl get networkpolicy allow-web-to-app -n $NAMESPACE -o json | jq -r '.spec.ingress[]?.ports[]?.port // empty')

if [[ "$WEB_TO_APP_FROM" != "web" ]] || [[ "$WEB_TO_APP_TARGET" != "app" ]] || [[ "$WEB_TO_APP_PORT" != "8080" ]]; then
    echo "ERROR: allow-web-to-app is not correctly configured"
    exit 1
fi

# Check allow-app-to-cache exists
if ! kubectl get networkpolicy allow-app-to-cache -n $NAMESPACE &>/dev/null; then
    echo "ERROR: NetworkPolicy 'allow-app-to-cache' not found"
    exit 1
fi

# Verify allow-app-to-cache configuration
APP_TO_CACHE_FROM=$(kubectl get networkpolicy allow-app-to-cache -n $NAMESPACE -o json | jq -r '.spec.ingress[]?.from[]?.podSelector.matchLabels.tier // empty')
APP_TO_CACHE_TARGET=$(kubectl get networkpolicy allow-app-to-cache -n $NAMESPACE -o jsonpath='{.spec.podSelector.matchLabels.tier}')
APP_TO_CACHE_PORT=$(kubectl get networkpolicy allow-app-to-cache -n $NAMESPACE -o json | jq -r '.spec.ingress[]?.ports[]?.port // empty')

if [[ "$APP_TO_CACHE_FROM" != "app" ]] || [[ "$APP_TO_CACHE_TARGET" != "cache" ]] || [[ "$APP_TO_CACHE_PORT" != "6379" ]]; then
    echo "ERROR: allow-app-to-cache is not correctly configured"
    exit 1
fi

# Check allow-dns exists
if ! kubectl get networkpolicy allow-dns -n $NAMESPACE &>/dev/null; then
    echo "ERROR: NetworkPolicy 'allow-dns' not found"
    exit 1
fi

# Verify allow-dns has port 53
DNS_PORT=$(kubectl get networkpolicy allow-dns -n $NAMESPACE -o json | jq -r '.spec.egress[]?.ports[]? | select(.port == 53) | .port')
if [[ "$DNS_PORT" != "53" ]]; then
    echo "ERROR: allow-dns doesn't allow port 53"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Zero-trust networking with default deny is correctly configured."
exit 0
