#!/bin/bash
# Verifier script for maxsurge-config task

set -e

echo "Verifying maxsurge-config task..."

NAMESPACE="deploy-test"

# Check if deployment exists
if ! kubectl get deployment web-surge --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Deployment 'web-surge' not found"
    exit 1
fi

# Check strategy type is RollingUpdate
STRATEGY=$(kubectl get deployment web-surge --namespace="$NAMESPACE" -o jsonpath='{.spec.strategy.type}')
if [ "$STRATEGY" != "RollingUpdate" ]; then
    echo "ERROR: Deployment strategy should be 'RollingUpdate', got '$STRATEGY'"
    exit 1
fi

# Check maxSurge (can be number or percentage)
MAX_SURGE=$(kubectl get deployment web-surge --namespace="$NAMESPACE" -o jsonpath='{.spec.strategy.rollingUpdate.maxSurge}')
if [ -z "$MAX_SURGE" ]; then
    echo "ERROR: maxSurge not configured"
    exit 1
fi

# Accept both 2 and 50%
if [ "$MAX_SURGE" != "2" ] && [ "$MAX_SURGE" != "50%" ]; then
    echo "ERROR: maxSurge should be 2 or 50%, got '$MAX_SURGE'"
    exit 1
fi

# Check maxUnavailable is 0
MAX_UNAVAIL=$(kubectl get deployment web-surge --namespace="$NAMESPACE" -o jsonpath='{.spec.strategy.rollingUpdate.maxUnavailable}')
if [ "$MAX_UNAVAIL" != "0" ] && [ "$MAX_UNAVAIL" != "0%" ]; then
    echo "ERROR: maxUnavailable should be 0, got '$MAX_UNAVAIL'"
    exit 1
fi

# Check replicas
REPLICAS=$(kubectl get deployment web-surge --namespace="$NAMESPACE" -o jsonpath='{.spec.replicas}')
if [ "$REPLICAS" != "4" ]; then
    echo "ERROR: Deployment should have 4 replicas, got $REPLICAS"
    exit 1
fi

echo "Verification PASSED: Deployment 'web-surge' created with maxSurge=$MAX_SURGE and maxUnavailable=$MAX_UNAVAIL"
exit 0
