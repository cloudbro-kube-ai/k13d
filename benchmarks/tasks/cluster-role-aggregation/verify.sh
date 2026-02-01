#!/bin/bash
set -euo pipefail

NAMESPACE="monitoring-system"
TIMEOUT="60s"

echo "Verifying cluster-role-aggregation..."

# Check if monitoring-view ClusterRole exists with correct label
if ! kubectl get clusterrole monitoring-view -o jsonpath='{.metadata.labels.rbac\.k13d\.io/aggregate-to-monitoring}' | grep -q "true"; then
    echo "ERROR: ClusterRole 'monitoring-view' doesn't have the correct aggregation label"
    exit 1
fi

# Check monitoring-view has correct rules
VIEW_RULES=$(kubectl get clusterrole monitoring-view -o json | jq -r '.rules[] | select(.resources | contains(["pods", "services", "endpoints"])) | .verbs | contains(["get", "list", "watch"])')
if [[ "$VIEW_RULES" != "true" ]]; then
    echo "ERROR: ClusterRole 'monitoring-view' doesn't have correct permissions for pods, services, endpoints"
    exit 1
fi

# Check if monitoring-logs ClusterRole exists with correct label
if ! kubectl get clusterrole monitoring-logs -o jsonpath='{.metadata.labels.rbac\.k13d\.io/aggregate-to-monitoring}' | grep -q "true"; then
    echo "ERROR: ClusterRole 'monitoring-logs' doesn't have the correct aggregation label"
    exit 1
fi

# Check monitoring-logs has correct rules for pods/log
LOGS_RULES=$(kubectl get clusterrole monitoring-logs -o json | jq -r '.rules[] | select(.resources | contains(["pods/log"])) | .verbs | contains(["get", "list", "watch"])')
if [[ "$LOGS_RULES" != "true" ]]; then
    echo "ERROR: ClusterRole 'monitoring-logs' doesn't have correct permissions for pods/log"
    exit 1
fi

# Check if monitoring-aggregate ClusterRole exists with aggregationRule
AGG_RULE=$(kubectl get clusterrole monitoring-aggregate -o json | jq -r '.aggregationRule.clusterRoleSelectors[0].matchLabels."rbac.k13d.io/aggregate-to-monitoring"')
if [[ "$AGG_RULE" != "true" ]]; then
    echo "ERROR: ClusterRole 'monitoring-aggregate' doesn't have correct aggregationRule"
    exit 1
fi

# Check that aggregated role has inherited rules (K8s controller should have aggregated them)
AGG_RULES_COUNT=$(kubectl get clusterrole monitoring-aggregate -o json | jq '.rules | length')
if [[ "$AGG_RULES_COUNT" -lt 2 ]]; then
    echo "ERROR: Aggregated ClusterRole doesn't have inherited rules (found $AGG_RULES_COUNT rules)"
    exit 1
fi

# Check ServiceAccount exists
if ! kubectl get serviceaccount monitoring-sa -n $NAMESPACE &>/dev/null; then
    echo "ERROR: ServiceAccount 'monitoring-sa' not found in namespace '$NAMESPACE'"
    exit 1
fi

# Check ClusterRoleBinding exists
if ! kubectl get clusterrolebinding monitoring-aggregate-binding &>/dev/null; then
    echo "ERROR: ClusterRoleBinding 'monitoring-aggregate-binding' not found"
    exit 1
fi

# Verify binding connects SA to aggregated role
BINDING_ROLE=$(kubectl get clusterrolebinding monitoring-aggregate-binding -o jsonpath='{.roleRef.name}')
BINDING_SA=$(kubectl get clusterrolebinding monitoring-aggregate-binding -o jsonpath='{.subjects[0].name}')
if [[ "$BINDING_ROLE" != "monitoring-aggregate" ]] || [[ "$BINDING_SA" != "monitoring-sa" ]]; then
    echo "ERROR: ClusterRoleBinding not correctly configured"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Aggregated ClusterRole structure is correctly configured."
exit 0
