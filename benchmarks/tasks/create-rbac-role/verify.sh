#!/bin/bash
# Verify script for create-rbac-role task

set -e

echo "Verifying RBAC resources..."

# Check ServiceAccount exists
if ! kubectl get serviceaccount app-reader --namespace="${NAMESPACE}" &>/dev/null; then
    echo "FAIL: ServiceAccount 'app-reader' not found"
    exit 1
fi
echo "✓ ServiceAccount 'app-reader' exists"

# Check Role exists
if ! kubectl get role pod-reader --namespace="${NAMESPACE}" &>/dev/null; then
    echo "FAIL: Role 'pod-reader' not found"
    exit 1
fi
echo "✓ Role 'pod-reader' exists"

# Check Role has correct permissions
ROLE_RULES=$(kubectl get role pod-reader --namespace="${NAMESPACE}" -o jsonpath='{.rules}')
if [[ ! "$ROLE_RULES" =~ "pods" ]] || [[ ! "$ROLE_RULES" =~ "get" ]]; then
    echo "FAIL: Role 'pod-reader' does not have correct pod permissions"
    exit 1
fi
echo "✓ Role has correct pod permissions"

# Check RoleBinding exists
if ! kubectl get rolebinding app-reader-binding --namespace="${NAMESPACE}" &>/dev/null; then
    echo "FAIL: RoleBinding 'app-reader-binding' not found"
    exit 1
fi
echo "✓ RoleBinding 'app-reader-binding' exists"

# Verify binding connects role to serviceaccount
BINDING_SA=$(kubectl get rolebinding app-reader-binding --namespace="${NAMESPACE}" -o jsonpath='{.subjects[0].name}')
BINDING_ROLE=$(kubectl get rolebinding app-reader-binding --namespace="${NAMESPACE}" -o jsonpath='{.roleRef.name}')

if [[ "$BINDING_SA" != "app-reader" ]]; then
    echo "FAIL: RoleBinding does not reference ServiceAccount 'app-reader'"
    exit 1
fi

if [[ "$BINDING_ROLE" != "pod-reader" ]]; then
    echo "FAIL: RoleBinding does not reference Role 'pod-reader'"
    exit 1
fi
echo "✓ RoleBinding correctly binds role to serviceaccount"

echo ""
echo "SUCCESS: All RBAC resources verified successfully"
exit 0
