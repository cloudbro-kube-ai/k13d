#!/bin/bash
set -euo pipefail

NAMESPACE="token-demo"
TIMEOUT="120s"

echo "Verifying service-account-token..."

# Check ServiceAccount exists
if ! kubectl get serviceaccount api-client -n $NAMESPACE &>/dev/null; then
    echo "ERROR: ServiceAccount 'api-client' not found"
    exit 1
fi

# Check Pod exists
if ! kubectl get pod token-test -n $NAMESPACE &>/dev/null; then
    echo "ERROR: Pod 'token-test' not found"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/token-test -n $NAMESPACE --timeout=$TIMEOUT; then
    echo "ERROR: Pod 'token-test' is not ready"
    exit 1
fi

# Verify automountServiceAccountToken is false
AUTOMOUNT=$(kubectl get pod token-test -n $NAMESPACE -o jsonpath='{.spec.automountServiceAccountToken}')
if [[ "$AUTOMOUNT" != "false" ]]; then
    echo "ERROR: automountServiceAccountToken should be false, got '$AUTOMOUNT'"
    exit 1
fi

# Check pod uses api-client ServiceAccount
POD_SA=$(kubectl get pod token-test -n $NAMESPACE -o jsonpath='{.spec.serviceAccountName}')
if [[ "$POD_SA" != "api-client" ]]; then
    echo "ERROR: Pod should use ServiceAccount 'api-client', got '$POD_SA'"
    exit 1
fi

# Verify projected volume exists with serviceAccountToken
PROJECTED_VOLUME=$(kubectl get pod token-test -n $NAMESPACE -o json | jq -r '.spec.volumes[] | select(.projected != null) | select(.projected.sources[].serviceAccountToken != null) | .name')
if [[ -z "$PROJECTED_VOLUME" ]]; then
    echo "ERROR: Pod doesn't have a projected volume with serviceAccountToken"
    exit 1
fi

# Check audience
AUDIENCE=$(kubectl get pod token-test -n $NAMESPACE -o json | jq -r '.spec.volumes[] | select(.projected != null) | .projected.sources[] | select(.serviceAccountToken != null) | .serviceAccountToken.audience')
if [[ "$AUDIENCE" != "api.k13d.io" ]]; then
    echo "ERROR: Token audience should be 'api.k13d.io', got '$AUDIENCE'"
    exit 1
fi

# Check expiration
EXPIRATION=$(kubectl get pod token-test -n $NAMESPACE -o json | jq -r '.spec.volumes[] | select(.projected != null) | .projected.sources[] | select(.serviceAccountToken != null) | .serviceAccountToken.expirationSeconds')
if [[ "$EXPIRATION" != "3600" ]]; then
    echo "ERROR: Token expiration should be 3600, got '$EXPIRATION'"
    exit 1
fi

# Check mount path
MOUNT_PATH=$(kubectl get pod token-test -n $NAMESPACE -o json | jq -r '.spec.containers[0].volumeMounts[] | select(.mountPath == "/var/run/secrets/tokens") | .mountPath')
if [[ "$MOUNT_PATH" != "/var/run/secrets/tokens" ]]; then
    echo "ERROR: Volume should be mounted at '/var/run/secrets/tokens'"
    exit 1
fi

# Verify token is readable in the pod
TOKEN_EXISTS=$(kubectl exec token-test -n $NAMESPACE -- cat /var/run/secrets/tokens/token 2>/dev/null | head -c 10)
if [[ -z "$TOKEN_EXISTS" ]]; then
    echo "ERROR: Token file is not readable at expected path"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Bound service account token is correctly configured."
exit 0
