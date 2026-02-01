#!/bin/bash
set -e

echo "Verifying pod-dns-search task..."

NAMESPACE="dns-search-test"

# Check Service exists
if ! kubectl get service backend-svc -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Service 'backend-svc' not found"
    exit 1
fi

# Check Pod exists
if ! kubectl get pod dns-search-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'dns-search-pod' not found"
    exit 1
fi

# Check dnsPolicy is ClusterFirst
DNS_POLICY=$(kubectl get pod dns-search-pod -n "$NAMESPACE" -o jsonpath='{.spec.dnsPolicy}' 2>/dev/null || echo "")
if [[ "$DNS_POLICY" != "ClusterFirst" ]]; then
    echo "FAILED: dnsPolicy should be 'ClusterFirst', got '$DNS_POLICY'"
    exit 1
fi

# Check searches include custom domains
SEARCHES=$(kubectl get pod dns-search-pod -n "$NAMESPACE" -o jsonpath='{.spec.dnsConfig.searches}' 2>/dev/null || echo "")
if [[ "$SEARCHES" != *"dns-search-test.svc.cluster.local"* ]]; then
    echo "FAILED: searches should include 'dns-search-test.svc.cluster.local'"
    exit 1
fi
if [[ "$SEARCHES" != *"prod.svc.cluster.local"* ]]; then
    echo "FAILED: searches should include 'prod.svc.cluster.local'"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/dns-search-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

# Verify resolv.conf contains search domains
RESOLV_CONF=$(kubectl exec dns-search-pod -n "$NAMESPACE" -- cat /etc/resolv.conf 2>/dev/null || echo "")
if [[ "$RESOLV_CONF" != *"search"* ]]; then
    echo "WARNING: resolv.conf may not contain search domains"
fi

echo "SUCCESS: Pod DNS search correctly configured"
exit 0
