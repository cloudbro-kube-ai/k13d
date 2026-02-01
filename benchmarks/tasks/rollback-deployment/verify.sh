#!/bin/bash
set -euo pipefail

NAMESPACE="rollback-demo"
TIMEOUT="120s"

echo "Verifying rollback-deployment..."

# Check deployment is available
if ! kubectl wait --for=condition=Available deployment/webapp -n $NAMESPACE --timeout=$TIMEOUT; then
    echo "ERROR: Deployment 'webapp' is not available"
    exit 1
fi

# Check the image is the rolled-back version
CURRENT_IMAGE=$(kubectl get deployment webapp -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}')
if [[ "$CURRENT_IMAGE" != "nginx:1.25-alpine" ]]; then
    echo "ERROR: Deployment should be rolled back to nginx:1.25-alpine, got '$CURRENT_IMAGE'"
    exit 1
fi

# Check rollback annotations
CHANGE_CAUSE=$(kubectl get deployment webapp -n $NAMESPACE -o jsonpath='{.metadata.annotations.kubernetes\.io/change-cause}')
if [[ ! "$CHANGE_CAUSE" =~ "Rolled back" ]] && [[ ! "$CHANGE_CAUSE" =~ "rollback" ]]; then
    echo "ERROR: Deployment should have change-cause annotation about rollback"
    exit 1
fi

FROM_REV=$(kubectl get deployment webapp -n $NAMESPACE -o jsonpath='{.metadata.annotations.rollback\.k13d\.io/from-revision}')
if [[ -z "$FROM_REV" ]]; then
    echo "ERROR: Deployment should have rollback.k13d.io/from-revision annotation"
    exit 1
fi

TO_REV=$(kubectl get deployment webapp -n $NAMESPACE -o jsonpath='{.metadata.annotations.rollback\.k13d\.io/to-revision}')
if [[ -z "$TO_REV" ]]; then
    echo "ERROR: Deployment should have rollback.k13d.io/to-revision annotation"
    exit 1
fi

REASON=$(kubectl get deployment webapp -n $NAMESPACE -o jsonpath='{.metadata.annotations.rollback\.k13d\.io/reason}')
if [[ -z "$REASON" ]]; then
    echo "ERROR: Deployment should have rollback.k13d.io/reason annotation"
    exit 1
fi

# Check revisionHistoryLimit
HIST_LIMIT=$(kubectl get deployment webapp -n $NAMESPACE -o jsonpath='{.spec.revisionHistoryLimit}')
if [[ "$HIST_LIMIT" != "5" ]]; then
    echo "ERROR: revisionHistoryLimit should be 5, got '$HIST_LIMIT'"
    exit 1
fi

# Check ConfigMap exists
if ! kubectl get configmap rollback-log -n $NAMESPACE &>/dev/null; then
    echo "ERROR: ConfigMap 'rollback-log' not found"
    exit 1
fi

FAILED_IMAGE=$(kubectl get configmap rollback-log -n $NAMESPACE -o jsonpath='{.data.FAILED_IMAGE}')
if [[ -z "$FAILED_IMAGE" ]]; then
    echo "ERROR: rollback-log should have FAILED_IMAGE"
    exit 1
fi

ROLLBACK_IMAGE=$(kubectl get configmap rollback-log -n $NAMESPACE -o jsonpath='{.data.ROLLBACK_IMAGE}')
if [[ "$ROLLBACK_IMAGE" != "nginx:1.25-alpine" ]]; then
    echo "ERROR: rollback-log ROLLBACK_IMAGE should be nginx:1.25-alpine"
    exit 1
fi

# Verify pods are running
READY_PODS=$(kubectl get deployment webapp -n $NAMESPACE -o jsonpath='{.status.readyReplicas}')
if [[ "$READY_PODS" -lt 2 ]]; then
    echo "ERROR: Deployment should have at least 2 ready replicas"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Deployment rollback completed successfully."
exit 0
