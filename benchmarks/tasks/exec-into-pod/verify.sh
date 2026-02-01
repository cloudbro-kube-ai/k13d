#!/bin/bash
set -e

echo "Verifying exec-into-pod task..."

NAMESPACE="exec-test"

# Check Pod exists and is running
STATUS=$(kubectl get pod debug-target -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
if [ "$STATUS" != "Running" ]; then
    echo "FAILED: Pod not in Running state, got '$STATUS'"
    exit 1
fi

# Verify exec works - run a simple command
echo "Testing kubectl exec..."
HOSTNAME=$(kubectl exec debug-target -n "$NAMESPACE" -- hostname 2>/dev/null || echo "FAILED")
if [[ "$HOSTNAME" == "FAILED" ]]; then
    echo "FAILED: kubectl exec failed"
    exit 1
fi

# Verify we can check nginx process
echo "Testing process listing..."
PROCESSES=$(kubectl exec debug-target -n "$NAMESPACE" -- ps aux 2>/dev/null || kubectl exec debug-target -n "$NAMESPACE" -- cat /proc/1/cmdline 2>/dev/null || echo "")
if [[ -z "$PROCESSES" ]]; then
    echo "WARNING: Could not list processes"
fi

# Verify we can read nginx config
echo "Testing config file access..."
NGINX_CONF=$(kubectl exec debug-target -n "$NAMESPACE" -- cat /etc/nginx/nginx.conf 2>/dev/null || echo "")
if [[ -z "$NGINX_CONF" ]]; then
    echo "FAILED: Could not read nginx config"
    exit 1
fi

# Verify network connectivity
echo "Testing network connectivity..."
HTTP_CODE=$(kubectl exec debug-target -n "$NAMESPACE" -- curl -s -o /dev/null -w "%{http_code}" localhost:80 2>/dev/null || echo "000")
if [[ "$HTTP_CODE" != "200" ]]; then
    echo "WARNING: HTTP check returned $HTTP_CODE"
fi

echo "SUCCESS: Exec into pod commands verified"
exit 0
