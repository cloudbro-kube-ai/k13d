#!/bin/bash
set -euo pipefail

NAMESPACE="graceful-demo"

echo "Verifying graceful-shutdown..."

# Check web-server terminationGracePeriodSeconds
WEB_GRACE=$(kubectl get deployment web-server -n $NAMESPACE -o jsonpath='{.spec.template.spec.terminationGracePeriodSeconds}')
if [[ "$WEB_GRACE" != "60" ]]; then
    echo "ERROR: web-server terminationGracePeriodSeconds should be 60, got '$WEB_GRACE'"
    exit 1
fi

# Check web-server preStop hook
WEB_PRESTOP=$(kubectl get deployment web-server -n $NAMESPACE -o json | jq -r '.spec.template.spec.containers[0].lifecycle.preStop.exec.command | join(" ") // empty')
if [[ ! "$WEB_PRESTOP" =~ "sleep" ]]; then
    echo "ERROR: web-server should have preStop hook with sleep"
    exit 1
fi

# Check web-server readinessProbe
WEB_READINESS_PATH=$(kubectl get deployment web-server -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].readinessProbe.httpGet.path}')
if [[ "$WEB_READINESS_PATH" != "/health" ]]; then
    echo "ERROR: web-server readinessProbe should check /health, got '$WEB_READINESS_PATH'"
    exit 1
fi

WEB_READINESS_PERIOD=$(kubectl get deployment web-server -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].readinessProbe.periodSeconds}')
if [[ "$WEB_READINESS_PERIOD" != "5" ]]; then
    echo "ERROR: web-server readinessProbe periodSeconds should be 5"
    exit 1
fi

# Check api-server terminationGracePeriodSeconds
API_GRACE=$(kubectl get deployment api-server -n $NAMESPACE -o jsonpath='{.spec.template.spec.terminationGracePeriodSeconds}')
if [[ "$API_GRACE" != "90" ]]; then
    echo "ERROR: api-server terminationGracePeriodSeconds should be 90, got '$API_GRACE'"
    exit 1
fi

# Check api-server preStop hook
API_PRESTOP=$(kubectl get deployment api-server -n $NAMESPACE -o json | jq -r '.spec.template.spec.containers[0].lifecycle.preStop.exec.command | join(" ") // empty')
if [[ -z "$API_PRESTOP" ]]; then
    echo "ERROR: api-server should have preStop hook"
    exit 1
fi

# Check api-server readinessProbe (TCP)
API_READINESS_PORT=$(kubectl get deployment api-server -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].readinessProbe.tcpSocket.port}')
if [[ "$API_READINESS_PORT" != "8080" ]]; then
    echo "ERROR: api-server readinessProbe should check TCP port 8080"
    exit 1
fi

# Check api-server livenessProbe
API_LIVENESS_PORT=$(kubectl get deployment api-server -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].livenessProbe.tcpSocket.port}')
if [[ "$API_LIVENESS_PORT" != "8080" ]]; then
    echo "ERROR: api-server livenessProbe should check TCP port 8080"
    exit 1
fi

API_LIVENESS_DELAY=$(kubectl get deployment api-server -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].livenessProbe.initialDelaySeconds}')
if [[ "$API_LIVENESS_DELAY" != "30" ]]; then
    echo "ERROR: api-server livenessProbe initialDelaySeconds should be 30"
    exit 1
fi

# Check web-server Service
if ! kubectl get service web-server -n $NAMESPACE &>/dev/null; then
    echo "ERROR: Service 'web-server' not found"
    exit 1
fi

WEB_SVC_AFFINITY=$(kubectl get service web-server -n $NAMESPACE -o jsonpath='{.spec.sessionAffinity}')
if [[ "$WEB_SVC_AFFINITY" != "ClientIP" ]]; then
    echo "ERROR: web-server service sessionAffinity should be ClientIP"
    exit 1
fi

# Check api-server Service
if ! kubectl get service api-server -n $NAMESPACE &>/dev/null; then
    echo "ERROR: Service 'api-server' not found"
    exit 1
fi

API_SVC_PORT=$(kubectl get service api-server -n $NAMESPACE -o jsonpath='{.spec.ports[0].port}')
if [[ "$API_SVC_PORT" != "8080" ]]; then
    echo "ERROR: api-server service port should be 8080"
    exit 1
fi

# Check shutdown annotations
WEB_SHUTDOWN_ANN=$(kubectl get deployment web-server -n $NAMESPACE -o jsonpath='{.metadata.annotations.shutdown\.k13d\.io/grace-period}')
if [[ -z "$WEB_SHUTDOWN_ANN" ]]; then
    echo "ERROR: web-server should have shutdown.k13d.io/grace-period annotation"
    exit 1
fi

API_SHUTDOWN_ANN=$(kubectl get deployment api-server -n $NAMESPACE -o jsonpath='{.metadata.annotations.shutdown\.k13d\.io/grace-period}')
if [[ -z "$API_SHUTDOWN_ANN" ]]; then
    echo "ERROR: api-server should have shutdown.k13d.io/grace-period annotation"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Graceful shutdown configured correctly for zero-downtime deployments."
exit 0
