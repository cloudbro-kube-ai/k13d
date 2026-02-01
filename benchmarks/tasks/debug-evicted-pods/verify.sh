#!/bin/bash
set -euo pipefail

NAMESPACE="eviction-demo"
TIMEOUT="120s"

echo "Verifying debug-evicted-pods..."

# Check PriorityClass 'critical-priority' exists
if ! kubectl get priorityclass critical-priority &>/dev/null; then
    echo "ERROR: PriorityClass 'critical-priority' not found"
    exit 1
fi

CRITICAL_VALUE=$(kubectl get priorityclass critical-priority -o jsonpath='{.value}')
if [[ "$CRITICAL_VALUE" -lt 1000000 ]]; then
    echo "ERROR: critical-priority should have value >= 1000000, got '$CRITICAL_VALUE'"
    exit 1
fi

# Check PriorityClass 'low-priority' exists
if ! kubectl get priorityclass low-priority &>/dev/null; then
    echo "ERROR: PriorityClass 'low-priority' not found"
    exit 1
fi

LOW_VALUE=$(kubectl get priorityclass low-priority -o jsonpath='{.value}')
if [[ "$LOW_VALUE" -gt 1000 ]]; then
    echo "ERROR: low-priority should have low value, got '$LOW_VALUE'"
    exit 1
fi

# Check critical-app uses critical-priority
CRITICAL_PC=$(kubectl get deployment critical-app -n $NAMESPACE -o jsonpath='{.spec.template.spec.priorityClassName}')
if [[ "$CRITICAL_PC" != "critical-priority" ]]; then
    echo "ERROR: critical-app should use priorityClassName 'critical-priority', got '$CRITICAL_PC'"
    exit 1
fi

# Check background-job uses low-priority
BACKGROUND_PC=$(kubectl get deployment background-job -n $NAMESPACE -o jsonpath='{.spec.template.spec.priorityClassName}')
if [[ "$BACKGROUND_PC" != "low-priority" ]]; then
    echo "ERROR: background-job should use priorityClassName 'low-priority', got '$BACKGROUND_PC'"
    exit 1
fi

# Check critical-app has Guaranteed QoS (requests = limits)
CRITICAL_MEM_REQ=$(kubectl get deployment critical-app -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].resources.requests.memory}')
CRITICAL_MEM_LIM=$(kubectl get deployment critical-app -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].resources.limits.memory}')
CRITICAL_CPU_REQ=$(kubectl get deployment critical-app -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].resources.requests.cpu}')
CRITICAL_CPU_LIM=$(kubectl get deployment critical-app -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].resources.limits.cpu}')

if [[ "$CRITICAL_MEM_REQ" != "$CRITICAL_MEM_LIM" ]] || [[ "$CRITICAL_CPU_REQ" != "$CRITICAL_CPU_LIM" ]]; then
    echo "ERROR: critical-app should have Guaranteed QoS (requests = limits)"
    echo "  Memory: req=$CRITICAL_MEM_REQ, lim=$CRITICAL_MEM_LIM"
    echo "  CPU: req=$CRITICAL_CPU_REQ, lim=$CRITICAL_CPU_LIM"
    exit 1
fi

# Verify specific resource values
if [[ "$CRITICAL_MEM_REQ" != "128Mi" ]]; then
    echo "ERROR: critical-app memory should be 128Mi, got '$CRITICAL_MEM_REQ'"
    exit 1
fi

if [[ "$CRITICAL_CPU_REQ" != "100m" ]]; then
    echo "ERROR: critical-app CPU should be 100m, got '$CRITICAL_CPU_REQ'"
    exit 1
fi

# Verify deployments are ready
if ! kubectl wait --for=condition=Available deployment/critical-app -n $NAMESPACE --timeout=$TIMEOUT; then
    echo "ERROR: critical-app deployment not available"
    exit 1
fi

# Check QoS class of running pods
POD_QOS=$(kubectl get pods -n $NAMESPACE -l app=critical-app -o jsonpath='{.items[0].status.qosClass}')
if [[ "$POD_QOS" != "Guaranteed" ]]; then
    echo "ERROR: critical-app pods should have Guaranteed QoS, got '$POD_QOS'"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Pod eviction prevention configured correctly."
exit 0
