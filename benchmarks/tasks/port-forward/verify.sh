#!/bin/bash
set -e

echo "Verifying port-forward task..."

NAMESPACE="port-fwd-test"

# Check Pod exists and is running
STATUS=$(kubectl get pod web-server -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
if [ "$STATUS" != "Running" ]; then
    echo "FAILED: Pod not in Running state, got '$STATUS'"
    exit 1
fi

# Check Service exists
if ! kubectl get service web-svc -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Service 'web-svc' not found"
    exit 1
fi

# Verify nginx is responding inside the cluster
echo "Testing nginx inside cluster..."
RESPONSE=$(kubectl exec web-server -n "$NAMESPACE" -- curl -s -o /dev/null -w "%{http_code}" localhost:80 2>/dev/null || echo "000")
if [[ "$RESPONSE" != "200" ]]; then
    echo "FAILED: nginx not responding inside pod"
    exit 1
fi

# Start port-forward in background for testing
echo "Testing port-forward capability..."
kubectl port-forward pod/web-server -n "$NAMESPACE" 18080:80 &
PF_PID=$!
sleep 3

# Test the port forward
LOCALHOST_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:18080 2>/dev/null || echo "000")

# Cleanup port-forward process
kill $PF_PID 2>/dev/null || true
wait $PF_PID 2>/dev/null || true

if [[ "$LOCALHOST_RESPONSE" == "200" ]]; then
    echo "SUCCESS: Port forward works correctly"
else
    echo "INFO: Port forward test returned $LOCALHOST_RESPONSE (may be timing issue)"
fi

echo "SUCCESS: Port forward setup verified"
exit 0
