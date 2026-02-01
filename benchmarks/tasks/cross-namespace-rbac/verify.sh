#!/bin/bash
set -euo pipefail

echo "Verifying cross-namespace-rbac..."

# Check namespaces exist
for NS in monitoring app-frontend app-backend; do
    if ! kubectl get namespace $NS &>/dev/null; then
        echo "ERROR: Namespace '$NS' not found"
        exit 1
    fi
done

# Check ServiceAccount exists
if ! kubectl get serviceaccount metrics-collector -n monitoring &>/dev/null; then
    echo "ERROR: ServiceAccount 'metrics-collector' not found in 'monitoring' namespace"
    exit 1
fi

# Check ClusterRole exists
if ! kubectl get clusterrole pod-reader &>/dev/null; then
    echo "ERROR: ClusterRole 'pod-reader' not found"
    exit 1
fi

# Verify ClusterRole has correct permissions
POD_PERMS=$(kubectl get clusterrole pod-reader -o json | jq -r '.rules[] | select(.resources | contains(["pods"])) | .verbs | contains(["get", "list", "watch"])')
if [[ "$POD_PERMS" != "true" ]]; then
    echo "ERROR: ClusterRole 'pod-reader' doesn't have get, list, watch permissions on pods"
    exit 1
fi

# Check RoleBindings exist in target namespaces
if ! kubectl get rolebinding metrics-collector-frontend -n app-frontend &>/dev/null; then
    echo "ERROR: RoleBinding 'metrics-collector-frontend' not found in 'app-frontend'"
    exit 1
fi

if ! kubectl get rolebinding metrics-collector-backend -n app-backend &>/dev/null; then
    echo "ERROR: RoleBinding 'metrics-collector-backend' not found in 'app-backend'"
    exit 1
fi

# Verify RoleBindings reference correct ClusterRole
FRONTEND_ROLE=$(kubectl get rolebinding metrics-collector-frontend -n app-frontend -o jsonpath='{.roleRef.name}')
BACKEND_ROLE=$(kubectl get rolebinding metrics-collector-backend -n app-backend -o jsonpath='{.roleRef.name}')

if [[ "$FRONTEND_ROLE" != "pod-reader" ]] || [[ "$BACKEND_ROLE" != "pod-reader" ]]; then
    echo "ERROR: RoleBindings don't reference 'pod-reader' ClusterRole"
    exit 1
fi

# Verify RoleBindings reference correct ServiceAccount
FRONTEND_SA=$(kubectl get rolebinding metrics-collector-frontend -n app-frontend -o json | jq -r '.subjects[] | select(.name == "metrics-collector") | select(.namespace == "monitoring") | .name')
BACKEND_SA=$(kubectl get rolebinding metrics-collector-backend -n app-backend -o json | jq -r '.subjects[] | select(.name == "metrics-collector") | select(.namespace == "monitoring") | .name')

if [[ "$FRONTEND_SA" != "metrics-collector" ]] || [[ "$BACKEND_SA" != "metrics-collector" ]]; then
    echo "ERROR: RoleBindings don't reference 'metrics-collector' ServiceAccount from 'monitoring' namespace"
    exit 1
fi

# Verify NO ClusterRoleBinding exists
if kubectl get clusterrolebinding -l app=metrics-collector &>/dev/null 2>&1; then
    CRB_COUNT=$(kubectl get clusterrolebinding -o name 2>/dev/null | grep -c "metrics-collector" || echo "0")
    if [[ "$CRB_COUNT" -gt 0 ]]; then
        echo "ERROR: ClusterRoleBinding should NOT exist - use RoleBindings for namespace-scoped access"
        exit 1
    fi
fi

# Test actual permissions
CAN_LIST_FRONTEND=$(kubectl auth can-i list pods -n app-frontend --as=system:serviceaccount:monitoring:metrics-collector 2>/dev/null || echo "no")
CAN_LIST_BACKEND=$(kubectl auth can-i list pods -n app-backend --as=system:serviceaccount:monitoring:metrics-collector 2>/dev/null || echo "no")
CAN_LIST_KUBESYSTEM=$(kubectl auth can-i list pods -n kube-system --as=system:serviceaccount:monitoring:metrics-collector 2>/dev/null || echo "no")

if [[ "$CAN_LIST_FRONTEND" != "yes" ]]; then
    echo "ERROR: metrics-collector cannot list pods in app-frontend"
    exit 1
fi

if [[ "$CAN_LIST_BACKEND" != "yes" ]]; then
    echo "ERROR: metrics-collector cannot list pods in app-backend"
    exit 1
fi

if [[ "$CAN_LIST_KUBESYSTEM" == "yes" ]]; then
    echo "ERROR: metrics-collector should NOT be able to list pods in kube-system"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Cross-namespace RBAC is correctly configured."
exit 0
