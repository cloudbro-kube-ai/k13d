#!/bin/bash
set -e

echo "Verifying horizontal-pod-autoscaler task..."

# Check if HPA exists
if ! kubectl get hpa -n autoscale-ns 2>/dev/null | grep -q web-app; then
    echo "FAILED: HPA for web-app not found"
    kubectl get hpa -n autoscale-ns 2>/dev/null || true
    exit 1
fi

# Check min replicas
MIN=$(kubectl get hpa -n autoscale-ns -o jsonpath='{.items[0].spec.minReplicas}')
if [ "$MIN" != "2" ]; then
    echo "FAILED: minReplicas should be 2, got '$MIN'"
    exit 1
fi

# Check max replicas
MAX=$(kubectl get hpa -n autoscale-ns -o jsonpath='{.items[0].spec.maxReplicas}')
if [ "$MAX" != "10" ]; then
    echo "FAILED: maxReplicas should be 10, got '$MAX'"
    exit 1
fi

# Check CPU target (either in metrics or targetCPUUtilizationPercentage)
CPU_TARGET=$(kubectl get hpa -n autoscale-ns -o json | grep -c "50" || echo "0")
if [ "$CPU_TARGET" -eq "0" ]; then
    echo "FAILED: CPU target 50% not found"
    exit 1
fi

echo "SUCCESS: HPA correctly configured"
exit 0
