#!/bin/bash
set -euo pipefail

NAMESPACE="pressure-demo"
TIMEOUT="120s"

echo "Verifying debug-node-pressure..."

# Check LimitRange exists
if ! kubectl get limitrange default-limits -n $NAMESPACE &>/dev/null; then
    echo "ERROR: LimitRange 'default-limits' not found"
    exit 1
fi

# Verify LimitRange defaults
LR_MEM_DEFAULT=$(kubectl get limitrange default-limits -n $NAMESPACE -o json | jq -r '.spec.limits[] | select(.type == "Container") | .default.memory')
LR_CPU_DEFAULT=$(kubectl get limitrange default-limits -n $NAMESPACE -o json | jq -r '.spec.limits[] | select(.type == "Container") | .default.cpu')

if [[ "$LR_MEM_DEFAULT" != "128Mi" ]]; then
    echo "ERROR: LimitRange default memory should be 128Mi, got '$LR_MEM_DEFAULT'"
    exit 1
fi

# Check max limits
LR_MEM_MAX=$(kubectl get limitrange default-limits -n $NAMESPACE -o json | jq -r '.spec.limits[] | select(.type == "Container") | .max.memory')
LR_CPU_MAX=$(kubectl get limitrange default-limits -n $NAMESPACE -o json | jq -r '.spec.limits[] | select(.type == "Container") | .max.cpu')

if [[ "$LR_MEM_MAX" != "512Mi" ]]; then
    echo "ERROR: LimitRange max memory should be 512Mi, got '$LR_MEM_MAX'"
    exit 1
fi

# Check ResourceQuota exists
if ! kubectl get resourcequota namespace-quota -n $NAMESPACE &>/dev/null; then
    echo "ERROR: ResourceQuota 'namespace-quota' not found"
    exit 1
fi

# Verify ResourceQuota values
RQ_CPU_REQ=$(kubectl get resourcequota namespace-quota -n $NAMESPACE -o jsonpath='{.spec.hard.requests\.cpu}')
RQ_MEM_REQ=$(kubectl get resourcequota namespace-quota -n $NAMESPACE -o jsonpath='{.spec.hard.requests\.memory}')
RQ_PODS=$(kubectl get resourcequota namespace-quota -n $NAMESPACE -o jsonpath='{.spec.hard.pods}')

if [[ "$RQ_CPU_REQ" != "2" ]]; then
    echo "ERROR: ResourceQuota requests.cpu should be 2, got '$RQ_CPU_REQ'"
    exit 1
fi

if [[ "$RQ_MEM_REQ" != "1Gi" ]]; then
    echo "ERROR: ResourceQuota requests.memory should be 1Gi, got '$RQ_MEM_REQ'"
    exit 1
fi

if [[ "$RQ_PODS" != "10" ]]; then
    echo "ERROR: ResourceQuota pods should be 10, got '$RQ_PODS'"
    exit 1
fi

# Check resource-hog deployment is optimized
HOG_REPLICAS=$(kubectl get deployment resource-hog -n $NAMESPACE -o jsonpath='{.spec.replicas}')
if [[ "$HOG_REPLICAS" -gt 2 ]]; then
    echo "ERROR: resource-hog replicas should be reduced to 2, got '$HOG_REPLICAS'"
    exit 1
fi

HOG_MEM_REQ=$(kubectl get deployment resource-hog -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].resources.requests.memory}')
HOG_MEM_LIM=$(kubectl get deployment resource-hog -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].resources.limits.memory}')

if [[ "$HOG_MEM_REQ" != "128Mi" ]]; then
    echo "ERROR: resource-hog memory request should be 128Mi, got '$HOG_MEM_REQ'"
    exit 1
fi

if [[ "$HOG_MEM_LIM" != "256Mi" ]]; then
    echo "ERROR: resource-hog memory limit should be 256Mi, got '$HOG_MEM_LIM'"
    exit 1
fi

# Verify all pods are scheduled (no Pending)
kubectl wait --for=condition=Available deployment/resource-hog -n $NAMESPACE --timeout=$TIMEOUT || {
    echo "ERROR: resource-hog deployment not available"
    exit 1
}

kubectl wait --for=condition=Available deployment/normal-app -n $NAMESPACE --timeout=$TIMEOUT || {
    echo "ERROR: normal-app deployment not available"
    exit 1
}

PENDING_PODS=$(kubectl get pods -n $NAMESPACE --field-selector=status.phase=Pending -o name | wc -l | tr -d ' ')
if [[ "$PENDING_PODS" -gt 0 ]]; then
    echo "ERROR: Still have $PENDING_PODS pending pods"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Node pressure issues resolved with proper resource management."
exit 0
