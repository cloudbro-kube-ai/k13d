#!/bin/bash
set -e

echo "Verifying fix-service-routing task..."

# Check if service has endpoints
ENDPOINTS=$(kubectl get endpoints nginx-service -n web -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null)

if [ -n "$ENDPOINTS" ]; then
    echo "SUCCESS: Service has endpoints: $ENDPOINTS"
    exit 0
else
    echo "FAILED: Service has no endpoints"
    kubectl describe svc nginx-service -n web 2>/dev/null || true
    exit 1
fi
