#!/bin/bash
set -e

echo "Verifying host-network task..."

NAMESPACE="host-net-test"

# Check Pod exists
if ! kubectl get pod host-network-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'host-network-pod' not found"
    exit 1
fi

# Check hostNetwork is true
HOST_NETWORK=$(kubectl get pod host-network-pod -n "$NAMESPACE" -o jsonpath='{.spec.hostNetwork}' 2>/dev/null || echo "")
if [[ "$HOST_NETWORK" != "true" ]]; then
    echo "FAILED: hostNetwork should be 'true', got '$HOST_NETWORK'"
    exit 1
fi

# Check dnsPolicy is ClusterFirstWithHostNet
DNS_POLICY=$(kubectl get pod host-network-pod -n "$NAMESPACE" -o jsonpath='{.spec.dnsPolicy}' 2>/dev/null || echo "")
if [[ "$DNS_POLICY" != "ClusterFirstWithHostNet" ]]; then
    echo "FAILED: dnsPolicy should be 'ClusterFirstWithHostNet', got '$DNS_POLICY'"
    exit 1
fi

# Check image is nginx
IMAGE=$(kubectl get pod host-network-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].image}' 2>/dev/null || echo "")
if [[ "$IMAGE" != *"nginx"* ]]; then
    echo "FAILED: Image should be nginx, got '$IMAGE'"
    exit 1
fi

# Check pod status
STATUS=$(kubectl get pod host-network-pod -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
echo "INFO: Pod status is '$STATUS'"

echo "SUCCESS: Host network correctly configured"
exit 0
