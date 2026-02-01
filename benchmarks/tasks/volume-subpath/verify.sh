#!/bin/bash
# Verifier script for volume-subpath task

set -e

echo "Verifying volume-subpath task..."

NAMESPACE="volume-test"

# Check if ConfigMap exists
if ! kubectl get configmap app-files --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: ConfigMap 'app-files' not found"
    exit 1
fi

# Check ConfigMap has required keys
NGINX_KEY=$(kubectl get configmap app-files --namespace="$NAMESPACE" -o jsonpath='{.data.nginx\.conf}')
if [ -z "$NGINX_KEY" ]; then
    echo "ERROR: ConfigMap missing 'nginx.conf' key"
    exit 1
fi

APP_KEY=$(kubectl get configmap app-files --namespace="$NAMESPACE" -o jsonpath='{.data.app\.properties}')
if [ -z "$APP_KEY" ]; then
    echo "ERROR: ConfigMap missing 'app.properties' key"
    exit 1
fi

# Check if pod exists
if ! kubectl get pod subpath-pod --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Pod 'subpath-pod' not found"
    exit 1
fi

# Check if pod is running
STATUS=$(kubectl get pod subpath-pod --namespace="$NAMESPACE" -o jsonpath='{.status.phase}')
if [ "$STATUS" != "Running" ]; then
    echo "ERROR: Pod is not running. Current status: $STATUS"
    exit 1
fi

# Check for volume mounts with subPath
VOLUME_MOUNTS=$(kubectl get pod subpath-pod --namespace="$NAMESPACE" -o json | grep -c "subPath" || echo "0")
if [ "$VOLUME_MOUNTS" -lt 2 ]; then
    echo "ERROR: Pod should have at least 2 volume mounts with subPath"
    exit 1
fi

# Verify subPath for nginx.conf
NGINX_SUBPATH=$(kubectl get pod subpath-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].volumeMounts[?(@.subPath=="nginx.conf")].mountPath}')
if [ -z "$NGINX_SUBPATH" ]; then
    echo "ERROR: Volume mount with subPath 'nginx.conf' not found"
    exit 1
fi

# Verify subPath for app.properties
APP_SUBPATH=$(kubectl get pod subpath-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[0].volumeMounts[?(@.subPath=="app.properties")].mountPath}')
if [ -z "$APP_SUBPATH" ]; then
    echo "ERROR: Volume mount with subPath 'app.properties' not found"
    exit 1
fi

echo "Verification PASSED: Pod 'subpath-pod' created with subPath volume mounts correctly configured"
exit 0
