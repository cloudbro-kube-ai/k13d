#!/bin/bash
set -euo pipefail

NAMESPACE="api-stress"
TIMEOUT="60s"

echo "Verifying debug-api-throttling..."

# Check ResourceQuota exists
if ! kubectl get resourcequota api-limits -n $NAMESPACE &>/dev/null; then
    echo "ERROR: ResourceQuota 'api-limits' not found"
    exit 1
fi

# Verify ResourceQuota values
RQ_PODS=$(kubectl get resourcequota api-limits -n $NAMESPACE -o jsonpath='{.spec.hard.count/pods}')
RQ_SVCS=$(kubectl get resourcequota api-limits -n $NAMESPACE -o jsonpath='{.spec.hard.count/services}')
RQ_CMS=$(kubectl get resourcequota api-limits -n $NAMESPACE -o jsonpath='{.spec.hard.count/configmaps}')
RQ_SECRETS=$(kubectl get resourcequota api-limits -n $NAMESPACE -o jsonpath='{.spec.hard.count/secrets}')

if [[ "$RQ_PODS" != "10" ]]; then
    echo "ERROR: api-limits count/pods should be 10, got '$RQ_PODS'"
    exit 1
fi

if [[ "$RQ_SVCS" != "5" ]]; then
    echo "ERROR: api-limits count/services should be 5, got '$RQ_SVCS'"
    exit 1
fi

if [[ "$RQ_CMS" != "20" ]]; then
    echo "ERROR: api-limits count/configmaps should be 20, got '$RQ_CMS'"
    exit 1
fi

if [[ "$RQ_SECRETS" != "10" ]]; then
    echo "ERROR: api-limits count/secrets should be 10, got '$RQ_SECRETS'"
    exit 1
fi

# Check api-heavy-client deployment
CLIENT_REPLICAS=$(kubectl get deployment api-heavy-client -n $NAMESPACE -o jsonpath='{.spec.replicas}')
if [[ "$CLIENT_REPLICAS" != "1" ]]; then
    echo "ERROR: api-heavy-client replicas should be 1, got '$CLIENT_REPLICAS'"
    exit 1
fi

# Check rate limiter annotations
RL_ENABLED=$(kubectl get deployment api-heavy-client -n $NAMESPACE -o jsonpath='{.metadata.annotations.rate-limiter\.k13d\.io/enabled}')
RL_QPS=$(kubectl get deployment api-heavy-client -n $NAMESPACE -o jsonpath='{.metadata.annotations.rate-limiter\.k13d\.io/qps}')
RL_BURST=$(kubectl get deployment api-heavy-client -n $NAMESPACE -o jsonpath='{.metadata.annotations.rate-limiter\.k13d\.io/burst}')

if [[ "$RL_ENABLED" != "true" ]]; then
    echo "ERROR: api-heavy-client should have rate-limiter enabled annotation"
    exit 1
fi

if [[ "$RL_QPS" != "5" ]]; then
    echo "ERROR: api-heavy-client QPS annotation should be 5, got '$RL_QPS'"
    exit 1
fi

if [[ "$RL_BURST" != "10" ]]; then
    echo "ERROR: api-heavy-client burst annotation should be 10, got '$RL_BURST'"
    exit 1
fi

# Check LimitRange exists
if ! kubectl get limitrange api-protection -n $NAMESPACE &>/dev/null; then
    echo "ERROR: LimitRange 'api-protection' not found"
    exit 1
fi

# Check LimitRange defaults
LR_CPU=$(kubectl get limitrange api-protection -n $NAMESPACE -o json | jq -r '.spec.limits[] | select(.type == "Container") | .default.cpu')
LR_MEM=$(kubectl get limitrange api-protection -n $NAMESPACE -o json | jq -r '.spec.limits[] | select(.type == "Container") | .default.memory')

if [[ "$LR_CPU" != "100m" ]]; then
    echo "ERROR: LimitRange default CPU should be 100m, got '$LR_CPU'"
    exit 1
fi

if [[ "$LR_MEM" != "64Mi" ]]; then
    echo "ERROR: LimitRange default memory should be 64Mi, got '$LR_MEM'"
    exit 1
fi

# Check monitor-spam is scaled to 0
SPAM_REPLICAS=$(kubectl get deployment monitor-spam -n $NAMESPACE -o jsonpath='{.spec.replicas}')
if [[ "$SPAM_REPLICAS" != "0" ]]; then
    echo "ERROR: monitor-spam replicas should be 0, got '$SPAM_REPLICAS'"
    exit 1
fi

# Verify legitimate-app is still running
if ! kubectl wait --for=condition=Available deployment/legitimate-app -n $NAMESPACE --timeout=$TIMEOUT; then
    echo "ERROR: legitimate-app should still be available"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "API throttling issues mitigated."
exit 0
