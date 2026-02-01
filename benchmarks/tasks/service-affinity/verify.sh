#!/bin/bash
# Verifier script for service-affinity task

set -e

echo "Verifying service-affinity task..."

NAMESPACE="service-test"

# Check if service exists
if ! kubectl get service sticky-svc --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Service 'sticky-svc' not found"
    exit 1
fi

# Check sessionAffinity
SESSION_AFFINITY=$(kubectl get service sticky-svc --namespace="$NAMESPACE" -o jsonpath='{.spec.sessionAffinity}')
if [ "$SESSION_AFFINITY" != "ClientIP" ]; then
    echo "ERROR: sessionAffinity should be 'ClientIP', got '$SESSION_AFFINITY'"
    exit 1
fi

# Check sessionAffinityConfig timeout
TIMEOUT=$(kubectl get service sticky-svc --namespace="$NAMESPACE" -o jsonpath='{.spec.sessionAffinityConfig.clientIP.timeoutSeconds}')
if [ "$TIMEOUT" != "3600" ]; then
    echo "ERROR: sessionAffinityConfig.clientIP.timeoutSeconds should be 3600, got '$TIMEOUT'"
    exit 1
fi

# Check selector
SELECTOR=$(kubectl get service sticky-svc --namespace="$NAMESPACE" -o jsonpath='{.spec.selector.app}')
if [ "$SELECTOR" != "sticky-app" ]; then
    echo "ERROR: Service selector 'app' should be 'sticky-app', got '$SELECTOR'"
    exit 1
fi

# Check deployment exists
if ! kubectl get deployment sticky-app --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Deployment 'sticky-app' not found"
    exit 1
fi

# Check deployment replicas
REPLICAS=$(kubectl get deployment sticky-app --namespace="$NAMESPACE" -o jsonpath='{.spec.replicas}')
if [ "$REPLICAS" != "3" ]; then
    echo "ERROR: Deployment should have 3 replicas, got $REPLICAS"
    exit 1
fi

echo "Verification PASSED: Service 'sticky-svc' created with sessionAffinity=ClientIP and timeout=3600s"
exit 0
