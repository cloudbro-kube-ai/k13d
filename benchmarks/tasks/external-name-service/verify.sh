#!/bin/bash
# Verifier script for external-name-service task

set -e

echo "Verifying external-name-service task..."

NAMESPACE="service-test"

# Check if service exists
if ! kubectl get service external-db --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Service 'external-db' not found"
    exit 1
fi

# Check service type
SERVICE_TYPE=$(kubectl get service external-db --namespace="$NAMESPACE" -o jsonpath='{.spec.type}')
if [ "$SERVICE_TYPE" != "ExternalName" ]; then
    echo "ERROR: Service type should be 'ExternalName', got '$SERVICE_TYPE'"
    exit 1
fi

# Check externalName
EXTERNAL_NAME=$(kubectl get service external-db --namespace="$NAMESPACE" -o jsonpath='{.spec.externalName}')
if [ "$EXTERNAL_NAME" != "database.example.com" ]; then
    echo "ERROR: externalName should be 'database.example.com', got '$EXTERNAL_NAME'"
    exit 1
fi

echo "Verification PASSED: ExternalName service 'external-db' created pointing to database.example.com"
exit 0
