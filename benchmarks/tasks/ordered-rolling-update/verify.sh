#!/bin/bash
set -euo pipefail

NAMESPACE="ordered-update"
TIMEOUT="180s"

echo "Verifying ordered-rolling-update..."

# Check StatefulSet exists
if ! kubectl get statefulset app-cluster -n $NAMESPACE &>/dev/null; then
    echo "ERROR: StatefulSet 'app-cluster' not found"
    exit 1
fi

# Check updateStrategy is RollingUpdate
UPDATE_TYPE=$(kubectl get statefulset app-cluster -n $NAMESPACE -o jsonpath='{.spec.updateStrategy.type}')
if [[ "$UPDATE_TYPE" != "RollingUpdate" ]]; then
    echo "ERROR: updateStrategy.type should be RollingUpdate, got '$UPDATE_TYPE'"
    exit 1
fi

# Check partition is 0 (final state after full rollout)
PARTITION=$(kubectl get statefulset app-cluster -n $NAMESPACE -o jsonpath='{.spec.updateStrategy.rollingUpdate.partition}')
if [[ "$PARTITION" != "0" ]] && [[ -n "$PARTITION" ]]; then
    echo "ERROR: Final partition should be 0, got '$PARTITION'"
    exit 1
fi

# Check all pods are running the new image
for i in {0..4}; do
    POD_IMAGE=$(kubectl get pod app-cluster-$i -n $NAMESPACE -o jsonpath='{.spec.containers[0].image}' 2>/dev/null || echo "")
    if [[ "$POD_IMAGE" != "nginx:1.25-alpine" ]]; then
        echo "ERROR: Pod app-cluster-$i should have nginx:1.25-alpine, got '$POD_IMAGE'"
        exit 1
    fi
done

# Check phase annotations
PHASE=$(kubectl get statefulset app-cluster -n $NAMESPACE -o jsonpath='{.metadata.annotations.update\.k13d\.io/phase}')
if [[ "$PHASE" != "3" ]]; then
    echo "ERROR: Final phase should be 3, got '$PHASE'"
    exit 1
fi

PARTITION_ANN=$(kubectl get statefulset app-cluster -n $NAMESPACE -o jsonpath='{.metadata.annotations.update\.k13d\.io/partition}')
if [[ "$PARTITION_ANN" != "0" ]]; then
    echo "ERROR: Final partition annotation should be 0, got '$PARTITION_ANN'"
    exit 1
fi

# Check ConfigMap
if ! kubectl get configmap rollout-status -n $NAMESPACE &>/dev/null; then
    echo "ERROR: ConfigMap 'rollout-status' not found"
    exit 1
fi

INITIAL=$(kubectl get configmap rollout-status -n $NAMESPACE -o jsonpath='{.data.INITIAL_IMAGE}')
if [[ "$INITIAL" != "nginx:1.24-alpine" ]]; then
    echo "ERROR: INITIAL_IMAGE should be nginx:1.24-alpine"
    exit 1
fi

FINAL=$(kubectl get configmap rollout-status -n $NAMESPACE -o jsonpath='{.data.FINAL_IMAGE}')
if [[ "$FINAL" != "nginx:1.25-alpine" ]]; then
    echo "ERROR: FINAL_IMAGE should be nginx:1.25-alpine"
    exit 1
fi

STRATEGY=$(kubectl get configmap rollout-status -n $NAMESPACE -o jsonpath='{.data.STRATEGY}')
if [[ "$STRATEGY" != "partitioned-rollout" ]]; then
    echo "ERROR: STRATEGY should be partitioned-rollout"
    exit 1
fi

PHASES=$(kubectl get configmap rollout-status -n $NAMESPACE -o jsonpath='{.data.PHASES}')
if [[ "$PHASES" != "3" ]]; then
    echo "ERROR: PHASES should be 3"
    exit 1
fi

# Verify all pods are ready
kubectl wait --for=condition=Ready pod -l app=app-cluster -n $NAMESPACE --timeout=$TIMEOUT || {
    echo "ERROR: Not all pods are ready"
    exit 1
}

echo "--- Verification Successful! ---"
echo "Ordered rolling update with partitions completed."
exit 0
