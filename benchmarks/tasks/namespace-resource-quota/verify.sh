#!/bin/bash
set -e

# Check if ResourceQuota exists
if ! kubectl get resourcequota compute-quota -n benchmark &>/dev/null; then
    echo "FAIL: ResourceQuota 'compute-quota' not found in namespace 'benchmark'"
    exit 1
fi

# Check requests.cpu
REQ_CPU=$(kubectl get resourcequota compute-quota -n benchmark -o jsonpath='{.spec.hard.requests\.cpu}')
if [[ "$REQ_CPU" != "2" ]]; then
    echo "FAIL: requests.cpu is '$REQ_CPU', expected '2'"
    exit 1
fi

# Check requests.memory
REQ_MEM=$(kubectl get resourcequota compute-quota -n benchmark -o jsonpath='{.spec.hard.requests\.memory}')
if [[ "$REQ_MEM" != "4Gi" ]]; then
    echo "FAIL: requests.memory is '$REQ_MEM', expected '4Gi'"
    exit 1
fi

# Check limits.cpu
LIM_CPU=$(kubectl get resourcequota compute-quota -n benchmark -o jsonpath='{.spec.hard.limits\.cpu}')
if [[ "$LIM_CPU" != "4" ]]; then
    echo "FAIL: limits.cpu is '$LIM_CPU', expected '4'"
    exit 1
fi

# Check limits.memory
LIM_MEM=$(kubectl get resourcequota compute-quota -n benchmark -o jsonpath='{.spec.hard.limits\.memory}')
if [[ "$LIM_MEM" != "8Gi" ]]; then
    echo "FAIL: limits.memory is '$LIM_MEM', expected '8Gi'"
    exit 1
fi

# Check pods
PODS=$(kubectl get resourcequota compute-quota -n benchmark -o jsonpath='{.spec.hard.pods}')
if [[ "$PODS" != "10" ]]; then
    echo "FAIL: pods is '$PODS', expected '10'"
    exit 1
fi

echo "PASS: ResourceQuota 'compute-quota' created correctly with all limits"
exit 0
