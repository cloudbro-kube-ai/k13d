#!/bin/bash
set -e

echo "Verifying custom-dns-config task..."

NAMESPACE="dns-config-test"

# Check Pod exists
if ! kubectl get pod custom-dns-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'custom-dns-pod' not found"
    exit 1
fi

# Check dnsPolicy is None
DNS_POLICY=$(kubectl get pod custom-dns-pod -n "$NAMESPACE" -o jsonpath='{.spec.dnsPolicy}' 2>/dev/null || echo "")
if [[ "$DNS_POLICY" != "None" ]]; then
    echo "FAILED: dnsPolicy should be 'None', got '$DNS_POLICY'"
    exit 1
fi

# Check nameservers include 8.8.8.8
NAMESERVERS=$(kubectl get pod custom-dns-pod -n "$NAMESPACE" -o jsonpath='{.spec.dnsConfig.nameservers}' 2>/dev/null || echo "")
if [[ "$NAMESERVERS" != *"8.8.8.8"* ]]; then
    echo "FAILED: dnsConfig.nameservers should include '8.8.8.8', got '$NAMESERVERS'"
    exit 1
fi

# Check searches include custom.local
SEARCHES=$(kubectl get pod custom-dns-pod -n "$NAMESPACE" -o jsonpath='{.spec.dnsConfig.searches}' 2>/dev/null || echo "")
if [[ "$SEARCHES" != *"custom.local"* ]]; then
    echo "FAILED: dnsConfig.searches should include 'custom.local', got '$SEARCHES'"
    exit 1
fi

# Check options include ndots
OPTIONS=$(kubectl get pod custom-dns-pod -n "$NAMESPACE" -o jsonpath='{.spec.dnsConfig.options}' 2>/dev/null || echo "")
if [[ "$OPTIONS" != *"ndots"* ]]; then
    echo "FAILED: dnsConfig.options should include 'ndots', got '$OPTIONS'"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/custom-dns-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

# Verify DNS config is applied inside the pod
RESOLV_CONF=$(kubectl exec custom-dns-pod -n "$NAMESPACE" -- cat /etc/resolv.conf 2>/dev/null || echo "")
if [[ "$RESOLV_CONF" != *"8.8.8.8"* ]]; then
    echo "WARNING: resolv.conf may not reflect custom DNS config"
fi

echo "SUCCESS: Custom DNS configuration correctly applied"
exit 0
