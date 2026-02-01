#!/bin/bash
# Verifier script for priority-class task

set -e

echo "Verifying priority-class task..."

NAMESPACE="schedule-test"

# Check if PriorityClass exists
if ! kubectl get priorityclass high-priority &>/dev/null; then
    echo "ERROR: PriorityClass 'high-priority' not found"
    exit 1
fi

# Check priority value
PRIORITY_VALUE=$(kubectl get priorityclass high-priority -o jsonpath='{.value}')
if [ "$PRIORITY_VALUE" != "1000000" ]; then
    echo "ERROR: PriorityClass value should be 1000000, got '$PRIORITY_VALUE'"
    exit 1
fi

# Check globalDefault
GLOBAL_DEFAULT=$(kubectl get priorityclass high-priority -o jsonpath='{.globalDefault}')
if [ "$GLOBAL_DEFAULT" == "true" ]; then
    echo "ERROR: PriorityClass globalDefault should be false"
    exit 1
fi

# Check preemptionPolicy
PREEMPTION=$(kubectl get priorityclass high-priority -o jsonpath='{.preemptionPolicy}')
if [ "$PREEMPTION" != "PreemptLowerPriority" ]; then
    echo "ERROR: preemptionPolicy should be 'PreemptLowerPriority', got '$PREEMPTION'"
    exit 1
fi

# Check if pod exists
if ! kubectl get pod critical-pod --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Pod 'critical-pod' not found"
    exit 1
fi

# Check pod priorityClassName
POD_PRIORITY=$(kubectl get pod critical-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.priorityClassName}')
if [ "$POD_PRIORITY" != "high-priority" ]; then
    echo "ERROR: Pod priorityClassName should be 'high-priority', got '$POD_PRIORITY'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get pod critical-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].image}')
if [[ "$IMAGE" != *"nginx"* ]]; then
    echo "ERROR: Pod should use nginx image, got $IMAGE"
    exit 1
fi

echo "Verification PASSED: PriorityClass 'high-priority' and pod 'critical-pod' configured correctly"
exit 0
