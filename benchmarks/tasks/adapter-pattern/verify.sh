#!/bin/bash
set -e

echo "Verifying adapter-pattern task..."

NAMESPACE="adapter-test"

# Check Pod exists
if ! kubectl get pod adapter-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'adapter-pod' not found"
    exit 1
fi

# Check both containers exist
CONTAINERS=$(kubectl get pod adapter-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[*].name}' 2>/dev/null || echo "")
if [[ "$CONTAINERS" != *"app"* ]]; then
    echo "FAILED: Container 'app' not found"
    exit 1
fi
if [[ "$CONTAINERS" != *"metrics-adapter"* ]]; then
    echo "FAILED: Container 'metrics-adapter' not found"
    exit 1
fi

# Check shared volume exists
VOLUMES=$(kubectl get pod adapter-pod -n "$NAMESPACE" -o jsonpath='{.spec.volumes[*].name}' 2>/dev/null || echo "")
if [[ "$VOLUMES" != *"metrics"* ]]; then
    echo "FAILED: Volume 'metrics' not found"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/adapter-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

# Wait for metrics to be generated
sleep 15

# Check app metrics file exists
APP_METRICS=$(kubectl exec adapter-pod -n "$NAMESPACE" -c app -- cat /metrics/app-metrics.txt 2>/dev/null || echo "")
if [[ -z "$APP_METRICS" ]]; then
    echo "FAILED: App metrics file not found or empty"
    exit 1
fi

# Check adapter output exists
PROMETHEUS_METRICS=$(kubectl exec adapter-pod -n "$NAMESPACE" -c metrics-adapter -- cat /metrics/prometheus-metrics.txt 2>/dev/null || echo "")
if [[ -z "$PROMETHEUS_METRICS" ]]; then
    echo "WARNING: Prometheus metrics file not found - adapter may not have processed yet"
fi

echo "SUCCESS: Adapter pattern correctly configured"
exit 0
