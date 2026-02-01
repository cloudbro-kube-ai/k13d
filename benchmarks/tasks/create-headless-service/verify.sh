#!/bin/bash
# Verifier script for create-headless-service task

set -e

echo "Verifying create-headless-service task..."

NAMESPACE="service-test"

# Check if service exists
if ! kubectl get service db-headless --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Service 'db-headless' not found"
    exit 1
fi

# Check clusterIP is None (headless)
CLUSTER_IP=$(kubectl get service db-headless --namespace="$NAMESPACE" -o jsonpath='{.spec.clusterIP}')
if [ "$CLUSTER_IP" != "None" ]; then
    echo "ERROR: Service clusterIP should be 'None' for headless service, got '$CLUSTER_IP'"
    exit 1
fi

# Check selector
SELECTOR=$(kubectl get service db-headless --namespace="$NAMESPACE" -o jsonpath='{.spec.selector.app}')
if [ "$SELECTOR" != "database" ]; then
    echo "ERROR: Service selector 'app' should be 'database', got '$SELECTOR'"
    exit 1
fi

# Check port
PORT=$(kubectl get service db-headless --namespace="$NAMESPACE" -o jsonpath='{.spec.ports[0].port}')
if [ "$PORT" != "3306" ]; then
    echo "ERROR: Service port should be 3306, got '$PORT'"
    exit 1
fi

# Check deployment exists
if ! kubectl get deployment database --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Deployment 'database' not found"
    exit 1
fi

# Check deployment has correct labels
DEPLOY_LABEL=$(kubectl get deployment database --namespace="$NAMESPACE" -o jsonpath='{.spec.template.metadata.labels.app}')
if [ "$DEPLOY_LABEL" != "database" ]; then
    echo "ERROR: Deployment pod label 'app' should be 'database', got '$DEPLOY_LABEL'"
    exit 1
fi

echo "Verification PASSED: Headless service 'db-headless' created with clusterIP=None"
exit 0
