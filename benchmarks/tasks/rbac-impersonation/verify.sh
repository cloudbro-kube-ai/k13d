#!/bin/bash
set -euo pipefail

NAMESPACE="tenant-platform"

echo "Verifying rbac-impersonation..."

# Check if platform-admin ServiceAccount exists
if ! kubectl get serviceaccount platform-admin -n $NAMESPACE &>/dev/null; then
    echo "ERROR: ServiceAccount 'platform-admin' not found in namespace '$NAMESPACE'"
    exit 1
fi

# Check if tenant-operator ServiceAccount exists
if ! kubectl get serviceaccount tenant-operator -n $NAMESPACE &>/dev/null; then
    echo "ERROR: ServiceAccount 'tenant-operator' not found in namespace '$NAMESPACE'"
    exit 1
fi

# Check if impersonation-role ClusterRole exists
if ! kubectl get clusterrole impersonation-role &>/dev/null; then
    echo "ERROR: ClusterRole 'impersonation-role' not found"
    exit 1
fi

# Verify impersonation-role has impersonate verb
IMPERSONATE_USERS=$(kubectl get clusterrole impersonation-role -o json | jq -r '.rules[] | select(.verbs | contains(["impersonate"])) | select(.resources | contains(["users"])) | .verbs[0]')
if [[ -z "$IMPERSONATE_USERS" ]]; then
    echo "ERROR: ClusterRole doesn't have impersonate permission on users"
    exit 1
fi

IMPERSONATE_GROUPS=$(kubectl get clusterrole impersonation-role -o json | jq -r '.rules[] | select(.verbs | contains(["impersonate"])) | select(.resources | contains(["groups"])) | .verbs[0]')
if [[ -z "$IMPERSONATE_GROUPS" ]]; then
    echo "ERROR: ClusterRole doesn't have impersonate permission on groups"
    exit 1
fi

IMPERSONATE_SA=$(kubectl get clusterrole impersonation-role -o json | jq -r '.rules[] | select(.verbs | contains(["impersonate"])) | select(.resources | contains(["serviceaccounts"])) | .verbs[0]')
if [[ -z "$IMPERSONATE_SA" ]]; then
    echo "ERROR: ClusterRole doesn't have impersonate permission on serviceaccounts"
    exit 1
fi

# Check ClusterRoleBinding exists and is correct
if ! kubectl get clusterrolebinding platform-impersonation &>/dev/null; then
    echo "ERROR: ClusterRoleBinding 'platform-impersonation' not found"
    exit 1
fi

BINDING_ROLE=$(kubectl get clusterrolebinding platform-impersonation -o jsonpath='{.roleRef.name}')
if [[ "$BINDING_ROLE" != "impersonation-role" ]]; then
    echo "ERROR: ClusterRoleBinding not bound to 'impersonation-role', found '$BINDING_ROLE'"
    exit 1
fi

# Check binding subject is platform-admin
BINDING_SA=$(kubectl get clusterrolebinding platform-impersonation -o json | jq -r '.subjects[] | select(.kind == "ServiceAccount") | select(.name == "platform-admin") | .name')
if [[ "$BINDING_SA" != "platform-admin" ]]; then
    echo "ERROR: ClusterRoleBinding subject is not 'platform-admin'"
    exit 1
fi

# Verify resourceNames restrictions exist (should restrict who can be impersonated)
RESOURCE_NAMES=$(kubectl get clusterrole impersonation-role -o json | jq -r '.rules[] | select(.verbs | contains(["impersonate"])) | .resourceNames // empty | length')
if [[ -n "$RESOURCE_NAMES" ]] && [[ "$RESOURCE_NAMES" -gt 0 ]]; then
    echo "INFO: Role has resourceNames restrictions - good security practice"
fi

echo "--- Verification Successful! ---"
echo "RBAC impersonation is correctly configured."
exit 0
