#!/bin/bash
set -euo pipefail

NAMESPACE="leader-election"

echo "Verifying leader-election..."

# Check Lease exists
if ! kubectl get lease controller-leader-lock -n $NAMESPACE &>/dev/null; then
    echo "ERROR: Lease 'controller-leader-lock' not found"
    exit 1
fi

# Verify Lease spec
LEASE_DURATION=$(kubectl get lease controller-leader-lock -n $NAMESPACE -o jsonpath='{.spec.leaseDurationSeconds}')
if [[ "$LEASE_DURATION" != "15" ]]; then
    echo "ERROR: Lease leaseDurationSeconds should be 15, got '$LEASE_DURATION'"
    exit 1
fi

# Check Role exists
if ! kubectl get role leader-election-role -n $NAMESPACE &>/dev/null; then
    echo "ERROR: Role 'leader-election-role' not found"
    exit 1
fi

# Verify Role has lease permissions
LEASE_PERMS=$(kubectl get role leader-election-role -n $NAMESPACE -o json | jq -r '.rules[] | select(.resources | contains(["leases"])) | .verbs | contains(["get", "create", "update"])')
if [[ "$LEASE_PERMS" != "true" ]]; then
    echo "ERROR: Role should have get, create, update permissions on leases"
    exit 1
fi

# Check ServiceAccount exists
if ! kubectl get serviceaccount controller-sa -n $NAMESPACE &>/dev/null; then
    echo "ERROR: ServiceAccount 'controller-sa' not found"
    exit 1
fi

# Check RoleBinding exists
BINDING_EXISTS=$(kubectl get rolebinding -n $NAMESPACE -o name 2>/dev/null | grep -c "leader-election" || echo "0")
if [[ "$BINDING_EXISTS" -lt 1 ]]; then
    echo "ERROR: RoleBinding for leader-election-role not found"
    exit 1
fi

# Check deployment configuration
DEPLOY_SA=$(kubectl get deployment controller-manager -n $NAMESPACE -o jsonpath='{.spec.template.spec.serviceAccountName}')
if [[ "$DEPLOY_SA" != "controller-sa" ]]; then
    echo "ERROR: Deployment should use ServiceAccount 'controller-sa', got '$DEPLOY_SA'"
    exit 1
fi

DEPLOY_REPLICAS=$(kubectl get deployment controller-manager -n $NAMESPACE -o jsonpath='{.spec.replicas}')
if [[ "$DEPLOY_REPLICAS" != "3" ]]; then
    echo "ERROR: Deployment should have 3 replicas for HA, got '$DEPLOY_REPLICAS'"
    exit 1
fi

# Check environment variables
LE_ENABLED=$(kubectl get deployment controller-manager -n $NAMESPACE -o json | jq -r '.spec.template.spec.containers[0].env[] | select(.name == "LEADER_ELECTION_ENABLED") | .value // empty')
if [[ "$LE_ENABLED" != "true" ]]; then
    echo "ERROR: LEADER_ELECTION_ENABLED env var should be 'true'"
    exit 1
fi

LE_LEASE=$(kubectl get deployment controller-manager -n $NAMESPACE -o json | jq -r '.spec.template.spec.containers[0].env[] | select(.name == "LEADER_ELECTION_LEASE_NAME") | .value // empty')
if [[ "$LE_LEASE" != "controller-leader-lock" ]]; then
    echo "ERROR: LEADER_ELECTION_LEASE_NAME env var should be 'controller-leader-lock'"
    exit 1
fi

# Check annotation
LE_ANNOTATION=$(kubectl get deployment controller-manager -n $NAMESPACE -o jsonpath='{.metadata.annotations.leader-election\.k13d\.io/enabled}')
if [[ "$LE_ANNOTATION" != "true" ]]; then
    echo "ERROR: Deployment should have leader-election.k13d.io/enabled annotation"
    exit 1
fi

# Check ConfigMap
if ! kubectl get configmap leader-election-config -n $NAMESPACE &>/dev/null; then
    echo "ERROR: ConfigMap 'leader-election-config' not found"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Leader election configured correctly."
exit 0
