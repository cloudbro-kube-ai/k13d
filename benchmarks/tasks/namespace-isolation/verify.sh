#!/bin/bash
set -euo pipefail

echo "Verifying namespace-isolation..."

# Check all namespaces exist with correct labels
for TENANT in alpha beta gamma; do
    NS="tenant-$TENANT"
    if ! kubectl get namespace $NS &>/dev/null; then
        echo "ERROR: Namespace '$NS' not found"
        exit 1
    fi

    LABEL=$(kubectl get namespace $NS -o jsonpath='{.metadata.labels.tenant}')
    if [[ "$LABEL" != "$TENANT" ]]; then
        echo "ERROR: Namespace '$NS' doesn't have label tenant=$TENANT"
        exit 1
    fi
done

# Check shared-services namespace
if ! kubectl get namespace shared-services &>/dev/null; then
    echo "ERROR: Namespace 'shared-services' not found"
    exit 1
fi

SHARED_LABEL=$(kubectl get namespace shared-services -o jsonpath='{.metadata.labels.shared}')
if [[ "$SHARED_LABEL" != "true" ]]; then
    echo "ERROR: Namespace 'shared-services' doesn't have label shared=true"
    exit 1
fi

# Check metrics-collector pod in shared-services
if ! kubectl get pod metrics-collector -n shared-services &>/dev/null; then
    echo "ERROR: Pod 'metrics-collector' not found in shared-services"
    exit 1
fi

# Check tenant-isolation policy in each tenant namespace
for TENANT in alpha beta gamma; do
    NS="tenant-$TENANT"

    if ! kubectl get networkpolicy tenant-isolation -n $NS &>/dev/null; then
        echo "ERROR: NetworkPolicy 'tenant-isolation' not found in $NS"
        exit 1
    fi

    # Verify empty podSelector (applies to all)
    POD_SELECTOR=$(kubectl get networkpolicy tenant-isolation -n $NS -o json | jq -r '.spec.podSelector | keys | length')
    if [[ "$POD_SELECTOR" != "0" ]]; then
        echo "ERROR: tenant-isolation in $NS should have empty podSelector"
        exit 1
    fi

    # Verify policy types
    TYPES=$(kubectl get networkpolicy tenant-isolation -n $NS -o jsonpath='{.spec.policyTypes[*]}')
    if [[ ! "$TYPES" =~ "Ingress" ]] || [[ ! "$TYPES" =~ "Egress" ]]; then
        echo "ERROR: tenant-isolation in $NS should have both Ingress and Egress types"
        exit 1
    fi

    # Check allow-shared-services policy exists
    if ! kubectl get networkpolicy allow-shared-services -n $NS &>/dev/null; then
        echo "ERROR: NetworkPolicy 'allow-shared-services' not found in $NS"
        exit 1
    fi

    # Verify allow-shared-services allows from shared=true namespaces
    SHARED_SELECTOR=$(kubectl get networkpolicy allow-shared-services -n $NS -o json | jq -r '.spec.ingress[]?.from[]?.namespaceSelector.matchLabels.shared // empty')
    if [[ "$SHARED_SELECTOR" != "true" ]]; then
        echo "ERROR: allow-shared-services in $NS doesn't allow from shared=true namespaces"
        exit 1
    fi
done

echo "--- Verification Successful! ---"
echo "Namespace network isolation is correctly configured."
exit 0
