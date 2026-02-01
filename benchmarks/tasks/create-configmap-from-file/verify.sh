#!/bin/bash
set -e

# Check if ConfigMap exists
if ! kubectl get configmap nginx-config -n benchmark &>/dev/null; then
    echo "FAIL: ConfigMap 'nginx-config' not found in namespace 'benchmark'"
    exit 1
fi

# Check if the key nginx.conf exists
NGINX_CONF=$(kubectl get configmap nginx-config -n benchmark -o jsonpath='{.data.nginx\.conf}')
if [[ -z "$NGINX_CONF" ]]; then
    echo "FAIL: Key 'nginx.conf' not found in ConfigMap"
    exit 1
fi

# Check if content contains expected nginx configuration
if [[ ! "$NGINX_CONF" =~ "listen 80" ]] || [[ ! "$NGINX_CONF" =~ "server_name localhost" ]]; then
    echo "FAIL: ConfigMap content does not match expected nginx configuration"
    exit 1
fi

echo "PASS: ConfigMap 'nginx-config' created correctly from file"
exit 0
