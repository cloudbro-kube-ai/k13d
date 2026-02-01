#!/bin/bash
# Verify script for fix-rbac-permission task

set -e

echo "Verifying RBAC fix..."

# Check if a Role or ClusterRole was created for the ServiceAccount
ROLE_EXISTS=false
if kubectl get role -n "${NAMESPACE}" -o name 2>/dev/null | grep -q .; then
    ROLE_EXISTS=true
fi
if kubectl get clusterrole -o name 2>/dev/null | grep -qi "api-client"; then
    ROLE_EXISTS=true
fi

if [ "$ROLE_EXISTS" = false ]; then
    echo "FAIL: No Role or ClusterRole found for api-client"
    exit 1
fi
echo "✓ Role/ClusterRole exists"

# Check if RoleBinding or ClusterRoleBinding exists
BINDING_EXISTS=false
if kubectl get rolebinding -n "${NAMESPACE}" -o jsonpath='{.items[*].subjects[*].name}' 2>/dev/null | grep -q "api-client-sa"; then
    BINDING_EXISTS=true
fi
if kubectl get clusterrolebinding -o jsonpath='{.items[*].subjects[*].name}' 2>/dev/null | grep -q "api-client-sa"; then
    BINDING_EXISTS=true
fi

if [ "$BINDING_EXISTS" = false ]; then
    echo "FAIL: No RoleBinding/ClusterRoleBinding found for api-client-sa"
    exit 1
fi
echo "✓ RoleBinding/ClusterRoleBinding exists"

# Verify the pod can now list pods
echo "Testing if pod can list pods..."
RESULT=$(kubectl exec api-client --namespace="${NAMESPACE}" -- kubectl get pods --namespace="${NAMESPACE}" 2>&1) || true

if echo "$RESULT" | grep -qi "forbidden"; then
    echo "FAIL: Pod still cannot access API - Forbidden"
    exit 1
fi

if echo "$RESULT" | grep -qi "error"; then
    echo "FAIL: Pod encountered error accessing API: $RESULT"
    exit 1
fi

echo "✓ Pod can successfully list pods"
echo ""
echo "SUCCESS: RBAC permissions fixed successfully"
exit 0
