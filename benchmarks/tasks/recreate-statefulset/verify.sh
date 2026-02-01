#!/bin/bash
set -euo pipefail

NAMESPACE="sts-recreate"
TIMEOUT="120s"

echo "Verifying recreate-statefulset..."

# Check StatefulSet exists
if ! kubectl get statefulset database -n $NAMESPACE &>/dev/null; then
    echo "ERROR: StatefulSet 'database' not found"
    exit 1
fi

# Check new selector
SELECTOR=$(kubectl get statefulset database -n $NAMESPACE -o jsonpath='{.spec.selector.matchLabels.app}')
if [[ "$SELECTOR" != "database-new" ]]; then
    echo "ERROR: StatefulSet selector should be app=database-new, got app=$SELECTOR"
    exit 1
fi

# Check pod labels
POD_LABEL=$(kubectl get statefulset database -n $NAMESPACE -o jsonpath='{.spec.template.metadata.labels.app}')
if [[ "$POD_LABEL" != "database-new" ]]; then
    echo "ERROR: StatefulSet pod template should have app=database-new label"
    exit 1
fi

# Check podManagementPolicy
POLICY=$(kubectl get statefulset database -n $NAMESPACE -o jsonpath='{.spec.podManagementPolicy}')
if [[ "$POLICY" != "Parallel" ]]; then
    echo "ERROR: podManagementPolicy should be Parallel, got '$POLICY'"
    exit 1
fi

# Check annotations
REASON=$(kubectl get statefulset database -n $NAMESPACE -o jsonpath='{.metadata.annotations.recreate\.k13d\.io/reason}')
if [[ "$REASON" != "selector-change" ]]; then
    echo "ERROR: StatefulSet should have recreate.k13d.io/reason annotation"
    exit 1
fi

PRESERVED=$(kubectl get statefulset database -n $NAMESPACE -o jsonpath='{.metadata.annotations.recreate\.k13d\.io/preserved-pvcs}')
if [[ "$PRESERVED" != "true" ]]; then
    echo "ERROR: StatefulSet should have recreate.k13d.io/preserved-pvcs annotation"
    exit 1
fi

# Wait for pods to be ready
kubectl wait --for=condition=Ready pod -l app=database-new -n $NAMESPACE --timeout=$TIMEOUT || {
    echo "ERROR: Pods not ready"
    exit 1
}

# Verify data was preserved in PVCs
DATA_0=$(kubectl exec database-0 -n $NAMESPACE -- cat /data/test.txt 2>/dev/null || echo "")
if [[ "$DATA_0" != "important-data-0" ]]; then
    echo "ERROR: Data in database-0 was not preserved, got '$DATA_0'"
    exit 1
fi

DATA_1=$(kubectl exec database-1 -n $NAMESPACE -- cat /data/test.txt 2>/dev/null || echo "")
if [[ "$DATA_1" != "important-data-1" ]]; then
    echo "ERROR: Data in database-1 was not preserved, got '$DATA_1'"
    exit 1
fi

# Check ConfigMap exists
if ! kubectl get configmap migration-record -n $NAMESPACE &>/dev/null; then
    echo "ERROR: ConfigMap 'migration-record' not found"
    exit 1
fi

OLD_SEL=$(kubectl get configmap migration-record -n $NAMESPACE -o jsonpath='{.data.OLD_SELECTOR}')
if [[ -z "$OLD_SEL" ]]; then
    echo "ERROR: migration-record should have OLD_SELECTOR"
    exit 1
fi

NEW_SEL=$(kubectl get configmap migration-record -n $NAMESPACE -o jsonpath='{.data.NEW_SELECTOR}')
if [[ "$NEW_SEL" != "database-new" ]]; then
    echo "ERROR: migration-record NEW_SELECTOR should be 'database-new'"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "StatefulSet recreated with data preserved."
exit 0
