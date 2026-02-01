#!/bin/bash
# Verifier script for create-ingress task

set -e

echo "Verifying create-ingress task..."

# Check if Ingress exists
if ! kubectl get ingress app-ingress --namespace="${NAMESPACE}" &>/dev/null; then
    echo "ERROR: Ingress 'app-ingress' not found"
    exit 1
fi

# Check host
HOST=$(kubectl get ingress app-ingress --namespace="${NAMESPACE}" -o jsonpath='{.spec.rules[0].host}')
if [ "$HOST" != "app.example.com" ]; then
    echo "ERROR: Host is '$HOST', expected 'app.example.com'"
    exit 1
fi

# Check for path rules (at least 2 paths expected)
PATH_COUNT=$(kubectl get ingress app-ingress --namespace="${NAMESPACE}" -o jsonpath='{.spec.rules[0].http.paths}' | grep -o '"path"' | wc -l)
if [ "$PATH_COUNT" -lt 2 ]; then
    echo "WARNING: Expected at least 2 path rules, found $PATH_COUNT"
fi

# Check if api-svc backend exists in paths
API_BACKEND=$(kubectl get ingress app-ingress --namespace="${NAMESPACE}" -o json | grep -o '"api-svc"' || true)
if [ -z "$API_BACKEND" ]; then
    echo "ERROR: Backend 'api-svc' not found in ingress paths"
    exit 1
fi

# Check if web-svc backend exists in paths
WEB_BACKEND=$(kubectl get ingress app-ingress --namespace="${NAMESPACE}" -o json | grep -o '"web-svc"' || true)
if [ -z "$WEB_BACKEND" ]; then
    echo "ERROR: Backend 'web-svc' not found in ingress paths"
    exit 1
fi

echo "Verification PASSED: Ingress 'app-ingress' created with correct routing configuration"
exit 0
