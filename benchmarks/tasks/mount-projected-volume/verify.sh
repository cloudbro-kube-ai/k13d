#!/bin/bash
# Verifier script for mount-projected-volume task

set -e

echo "Verifying mount-projected-volume task..."

NAMESPACE="volume-test"

# Check if ConfigMap exists
if ! kubectl get configmap app-config --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: ConfigMap 'app-config' not found"
    exit 1
fi

# Check if Secret exists
if ! kubectl get secret app-secret --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Secret 'app-secret' not found"
    exit 1
fi

# Check if pod exists
if ! kubectl get pod projected-pod --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Pod 'projected-pod' not found"
    exit 1
fi

# Check if pod is running
STATUS=$(kubectl get pod projected-pod --namespace="$NAMESPACE" -o jsonpath='{.status.phase}')
if [ "$STATUS" != "Running" ]; then
    echo "ERROR: Pod is not running. Current status: $STATUS"
    exit 1
fi

# Check for projected volume
PROJECTED=$(kubectl get pod projected-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.volumes[?(@.name=="combined-volume")].projected}')
if [ -z "$PROJECTED" ]; then
    echo "ERROR: Projected volume 'combined-volume' not found"
    exit 1
fi

# Check that projected volume contains configMap source
CONFIGMAP_SOURCE=$(kubectl get pod projected-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.volumes[?(@.name=="combined-volume")].projected.sources[?(@.configMap)].configMap.name}')
if [ "$CONFIGMAP_SOURCE" != "app-config" ]; then
    echo "ERROR: Projected volume should include configMap 'app-config'"
    exit 1
fi

# Check that projected volume contains secret source
SECRET_SOURCE=$(kubectl get pod projected-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.volumes[?(@.name=="combined-volume")].projected.sources[?(@.secret)].secret.name}')
if [ "$SECRET_SOURCE" != "app-secret" ]; then
    echo "ERROR: Projected volume should include secret 'app-secret'"
    exit 1
fi

# Check volume mount
MOUNT_PATH=$(kubectl get pod projected-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].volumeMounts[?(@.name=="combined-volume")].mountPath}')
if [ "$MOUNT_PATH" != "/etc/config" ]; then
    echo "ERROR: Volume mount path should be /etc/config, got $MOUNT_PATH"
    exit 1
fi

echo "Verification PASSED: Pod 'projected-pod' created with projected volume containing ConfigMap and Secret"
exit 0
