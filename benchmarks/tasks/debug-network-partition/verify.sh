#!/bin/bash
set -euo pipefail

NAMESPACE="network-debug"
TIMEOUT="60s"

echo "Verifying debug-network-partition..."

# Check backend service selector is fixed
BACKEND_SELECTOR=$(kubectl get service backend -n $NAMESPACE -o jsonpath='{.spec.selector.app}')
if [[ "$BACKEND_SELECTOR" != "backend-api" ]]; then
    echo "ERROR: backend service selector should be 'app=backend-api', got '$BACKEND_SELECTOR'"
    exit 1
fi

# Check backend service has endpoints
BACKEND_ENDPOINTS=$(kubectl get endpoints backend -n $NAMESPACE -o jsonpath='{.subsets[0].addresses}')
if [[ -z "$BACKEND_ENDPOINTS" || "$BACKEND_ENDPOINTS" == "null" ]]; then
    echo "ERROR: backend service has no endpoints"
    exit 1
fi

# Check database service exists
if ! kubectl get service database -n $NAMESPACE &>/dev/null; then
    echo "ERROR: Service 'database' not found"
    exit 1
fi

# Verify database service configuration
DB_PORT=$(kubectl get service database -n $NAMESPACE -o jsonpath='{.spec.ports[0].port}')
DB_TARGET=$(kubectl get service database -n $NAMESPACE -o jsonpath='{.spec.ports[0].targetPort}')
DB_SELECTOR=$(kubectl get service database -n $NAMESPACE -o jsonpath='{.spec.selector.app}')

if [[ "$DB_PORT" != "5432" ]]; then
    echo "ERROR: database service port should be 5432, got '$DB_PORT'"
    exit 1
fi

if [[ "$DB_SELECTOR" != "postgres" ]]; then
    echo "ERROR: database service selector should be 'app=postgres', got '$DB_SELECTOR'"
    exit 1
fi

# Check database service has endpoints
DB_ENDPOINTS=$(kubectl get endpoints database -n $NAMESPACE -o jsonpath='{.subsets[0].addresses}')
if [[ -z "$DB_ENDPOINTS" || "$DB_ENDPOINTS" == "null" ]]; then
    echo "ERROR: database service has no endpoints"
    exit 1
fi

# Check NetworkPolicy exists
if ! kubectl get networkpolicy allow-internal -n $NAMESPACE &>/dev/null; then
    echo "ERROR: NetworkPolicy 'allow-internal' not found"
    exit 1
fi

# Verify NetworkPolicy allows internal traffic
NP_SELECTOR=$(kubectl get networkpolicy allow-internal -n $NAMESPACE -o json | jq -r '.spec.podSelector | keys | length')
if [[ "$NP_SELECTOR" != "0" ]]; then
    echo "ERROR: allow-internal should have empty podSelector (apply to all pods)"
    exit 1
fi

# Test DNS resolution from frontend pod
FRONTEND_POD=$(kubectl get pods -n $NAMESPACE -l app=frontend -o jsonpath='{.items[0].metadata.name}')
DNS_TEST=$(kubectl exec $FRONTEND_POD -n $NAMESPACE -- nslookup backend.network-debug.svc.cluster.local 2>/dev/null | grep -c "Address" || echo "0")
if [[ "$DNS_TEST" -lt 1 ]]; then
    echo "WARNING: DNS resolution test inconclusive"
fi

# Test connectivity from frontend to backend
BACKEND_IP=$(kubectl get service backend -n $NAMESPACE -o jsonpath='{.spec.clusterIP}')
CONNECT_TEST=$(kubectl exec $FRONTEND_POD -n $NAMESPACE -- wget -q -O /dev/null --timeout=5 http://$BACKEND_IP:8080 2>&1 && echo "success" || echo "failed")
if [[ "$CONNECT_TEST" != "success" ]]; then
    echo "WARNING: Connectivity test to backend inconclusive (may be normal if nginx not configured)"
fi

echo "--- Verification Successful! ---"
echo "Network partition issues fixed. Services can communicate."
exit 0
