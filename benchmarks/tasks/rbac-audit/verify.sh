#!/bin/bash
set -euo pipefail

NAMESPACE="dev-team"

echo "Verifying rbac-audit..."

# Check Role exists
if ! kubectl get role developer-role -n $NAMESPACE &>/dev/null; then
    echo "ERROR: Role 'developer-role' not found"
    exit 1
fi

# Check RoleBinding exists
if ! kubectl get rolebinding developer-binding -n $NAMESPACE &>/dev/null; then
    echo "ERROR: RoleBinding 'developer-binding' not found"
    exit 1
fi

# Get role rules as JSON
ROLE_JSON=$(kubectl get role developer-role -n $NAMESPACE -o json)

# Check NO wildcard verbs on secrets
SECRETS_WILDCARD=$(echo "$ROLE_JSON" | jq -r '.rules[] | select(.resources | contains(["secrets"])) | .verbs | contains(["*"])')
if [[ "$SECRETS_WILDCARD" == "true" ]]; then
    echo "ERROR: Secrets still have wildcard (*) verbs"
    exit 1
fi

# Check secrets only have get, list (not delete, create, update)
SECRETS_DELETE=$(kubectl auth can-i delete secrets -n $NAMESPACE --as=system:serviceaccount:$NAMESPACE:dev-sa 2>/dev/null || echo "no")
if [[ "$SECRETS_DELETE" == "yes" ]]; then
    echo "ERROR: dev-sa can still delete secrets"
    exit 1
fi

# Check NO delete on deployments
DEPLOY_DELETE=$(kubectl auth can-i delete deployments -n $NAMESPACE --as=system:serviceaccount:$NAMESPACE:dev-sa 2>/dev/null || echo "no")
if [[ "$DEPLOY_DELETE" == "yes" ]]; then
    echo "ERROR: dev-sa can still delete deployments"
    exit 1
fi

# Check can still do allowed operations
DEPLOY_GET=$(kubectl auth can-i get deployments -n $NAMESPACE --as=system:serviceaccount:$NAMESPACE:dev-sa 2>/dev/null || echo "no")
if [[ "$DEPLOY_GET" != "yes" ]]; then
    echo "ERROR: dev-sa should be able to get deployments"
    exit 1
fi

DEPLOY_UPDATE=$(kubectl auth can-i update deployments -n $NAMESPACE --as=system:serviceaccount:$NAMESPACE:dev-sa 2>/dev/null || echo "no")
if [[ "$DEPLOY_UPDATE" != "yes" ]]; then
    echo "ERROR: dev-sa should be able to update deployments"
    exit 1
fi

# Check NO wildcard resources
WILDCARD_RESOURCES=$(echo "$ROLE_JSON" | jq -r '.rules[] | .resources | contains(["*"])')
if echo "$WILDCARD_RESOURCES" | grep -q "true"; then
    echo "ERROR: Role still has wildcard (*) resources"
    exit 1
fi

# Verify can access allowed resources: pods, services, configmaps
PODS_GET=$(kubectl auth can-i get pods -n $NAMESPACE --as=system:serviceaccount:$NAMESPACE:dev-sa 2>/dev/null || echo "no")
SVC_GET=$(kubectl auth can-i get services -n $NAMESPACE --as=system:serviceaccount:$NAMESPACE:dev-sa 2>/dev/null || echo "no")
CM_GET=$(kubectl auth can-i get configmaps -n $NAMESPACE --as=system:serviceaccount:$NAMESPACE:dev-sa 2>/dev/null || echo "no")

if [[ "$PODS_GET" != "yes" ]] || [[ "$SVC_GET" != "yes" ]] || [[ "$CM_GET" != "yes" ]]; then
    echo "ERROR: dev-sa should be able to get pods, services, and configmaps"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "RBAC audit fixes applied correctly. Role is now properly scoped."
exit 0
