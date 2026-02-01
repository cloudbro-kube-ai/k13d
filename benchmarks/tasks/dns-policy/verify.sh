#!/bin/bash
set -e

echo "Verifying dns-policy task..."

NAMESPACE="dns-policy-test"

# Check first pod exists
if ! kubectl get pod cluster-first-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'cluster-first-pod' not found"
    exit 1
fi

# Check first pod DNS policy
DNS_POLICY1=$(kubectl get pod cluster-first-pod -n "$NAMESPACE" -o jsonpath='{.spec.dnsPolicy}' 2>/dev/null || echo "")
if [[ "$DNS_POLICY1" != "ClusterFirst" ]]; then
    echo "FAILED: cluster-first-pod dnsPolicy should be 'ClusterFirst', got '$DNS_POLICY1'"
    exit 1
fi

# Check second pod exists
if ! kubectl get pod cluster-first-host-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'cluster-first-host-pod' not found"
    exit 1
fi

# Check second pod DNS policy
DNS_POLICY2=$(kubectl get pod cluster-first-host-pod -n "$NAMESPACE" -o jsonpath='{.spec.dnsPolicy}' 2>/dev/null || echo "")
if [[ "$DNS_POLICY2" != "ClusterFirstWithHostNet" ]]; then
    echo "FAILED: cluster-first-host-pod dnsPolicy should be 'ClusterFirstWithHostNet', got '$DNS_POLICY2'"
    exit 1
fi

# Check second pod has hostNetwork
HOST_NET=$(kubectl get pod cluster-first-host-pod -n "$NAMESPACE" -o jsonpath='{.spec.hostNetwork}' 2>/dev/null || echo "")
if [[ "$HOST_NET" != "true" ]]; then
    echo "FAILED: cluster-first-host-pod should have hostNetwork: true, got '$HOST_NET'"
    exit 1
fi

# Wait for first pod to be ready
if ! kubectl wait --for=condition=Ready pod/cluster-first-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: cluster-first-pod did not become ready in time"
    exit 1
fi

echo "SUCCESS: DNS policies correctly configured"
exit 0
