#!/bin/bash
set -euo pipefail

NAMESPACE="external-access"

echo "Verifying cidr-policy..."

# Check api-gateway-external policy exists
if ! kubectl get networkpolicy api-gateway-external -n $NAMESPACE &>/dev/null; then
    echo "ERROR: NetworkPolicy 'api-gateway-external' not found"
    exit 1
fi

# Check worker-restricted policy exists
if ! kubectl get networkpolicy worker-restricted -n $NAMESPACE &>/dev/null; then
    echo "ERROR: NetworkPolicy 'worker-restricted' not found"
    exit 1
fi

# Check ingress-from-lb policy exists
if ! kubectl get networkpolicy ingress-from-lb -n $NAMESPACE &>/dev/null; then
    echo "ERROR: NetworkPolicy 'ingress-from-lb' not found"
    exit 1
fi

# Verify api-gateway-external targets api-gateway
API_GW_SELECTOR=$(kubectl get networkpolicy api-gateway-external -n $NAMESPACE -o jsonpath='{.spec.podSelector.matchLabels.app}')
if [[ "$API_GW_SELECTOR" != "api-gateway" ]]; then
    echo "ERROR: api-gateway-external doesn't target app=api-gateway"
    exit 1
fi

# Verify api-gateway-external has Egress type
API_GW_TYPES=$(kubectl get networkpolicy api-gateway-external -n $NAMESPACE -o jsonpath='{.spec.policyTypes[*]}')
if [[ ! "$API_GW_TYPES" =~ "Egress" ]]; then
    echo "ERROR: api-gateway-external should have Egress policy type"
    exit 1
fi

# Check for CIDR blocks in api-gateway-external
API_GW_CIDRS=$(kubectl get networkpolicy api-gateway-external -n $NAMESPACE -o json | jq -r '.spec.egress[]?.to[]?.ipBlock.cidr // empty')
if [[ -z "$API_GW_CIDRS" ]]; then
    echo "ERROR: api-gateway-external doesn't have any CIDR blocks"
    exit 1
fi

# Check for 10.0.0.0/8 CIDR
if ! echo "$API_GW_CIDRS" | grep -q "10.0.0.0/8"; then
    echo "ERROR: api-gateway-external should allow 10.0.0.0/8"
    exit 1
fi

# Check for ipBlock.except (blocking 172.16.0.0/12)
EXCEPT_CIDR=$(kubectl get networkpolicy api-gateway-external -n $NAMESPACE -o json | jq -r '.spec.egress[]?.to[]?.ipBlock.except[]? // empty' | head -1)
if [[ -z "$EXCEPT_CIDR" ]] && ! echo "$EXCEPT_CIDR" | grep -q "172.16"; then
    echo "WARNING: api-gateway-external should use ipBlock.except to block 172.16.0.0/12"
fi

# Verify worker-restricted targets worker
WORKER_SELECTOR=$(kubectl get networkpolicy worker-restricted -n $NAMESPACE -o jsonpath='{.spec.podSelector.matchLabels.app}')
if [[ "$WORKER_SELECTOR" != "worker" ]]; then
    echo "ERROR: worker-restricted doesn't target app=worker"
    exit 1
fi

# Check worker only allows 10.0.0.0/8
WORKER_CIDRS=$(kubectl get networkpolicy worker-restricted -n $NAMESPACE -o json | jq -r '.spec.egress[]?.to[]?.ipBlock.cidr // empty')
# Should have 10.0.0.0/8
if ! echo "$WORKER_CIDRS" | grep -q "10.0.0.0/8"; then
    echo "ERROR: worker-restricted should allow 10.0.0.0/8"
    exit 1
fi

# Verify ingress-from-lb targets api-gateway
LB_TARGET=$(kubectl get networkpolicy ingress-from-lb -n $NAMESPACE -o jsonpath='{.spec.podSelector.matchLabels.app}')
if [[ "$LB_TARGET" != "api-gateway" ]]; then
    echo "ERROR: ingress-from-lb doesn't target app=api-gateway"
    exit 1
fi

# Check ingress-from-lb has CIDR for load balancer
LB_CIDR=$(kubectl get networkpolicy ingress-from-lb -n $NAMESPACE -o json | jq -r '.spec.ingress[]?.from[]?.ipBlock.cidr // empty')
if [[ "$LB_CIDR" != "203.0.113.0/24" ]]; then
    echo "ERROR: ingress-from-lb should allow from 203.0.113.0/24, got '$LB_CIDR'"
    exit 1
fi

# Check ingress-from-lb port
LB_PORT=$(kubectl get networkpolicy ingress-from-lb -n $NAMESPACE -o json | jq -r '.spec.ingress[]?.ports[]?.port // empty')
if [[ "$LB_PORT" != "8080" ]]; then
    echo "ERROR: ingress-from-lb should allow port 8080"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "CIDR-based network policies are correctly configured."
exit 0
