#!/bin/bash
set -e

# Check if ConfigMap exists
if ! kubectl get configmap app-config -n benchmark &>/dev/null; then
    echo "FAIL: ConfigMap 'app-config' not found in namespace 'benchmark'"
    exit 1
fi

# Check DATABASE_HOST
DB_HOST=$(kubectl get configmap app-config -n benchmark -o jsonpath='{.data.DATABASE_HOST}')
if [[ "$DB_HOST" != "mysql.default.svc.cluster.local" ]]; then
    echo "FAIL: DATABASE_HOST is '$DB_HOST', expected 'mysql.default.svc.cluster.local'"
    exit 1
fi

# Check DATABASE_PORT
DB_PORT=$(kubectl get configmap app-config -n benchmark -o jsonpath='{.data.DATABASE_PORT}')
if [[ "$DB_PORT" != "3306" ]]; then
    echo "FAIL: DATABASE_PORT is '$DB_PORT', expected '3306'"
    exit 1
fi

# Check LOG_LEVEL
LOG_LEVEL=$(kubectl get configmap app-config -n benchmark -o jsonpath='{.data.LOG_LEVEL}')
if [[ "$LOG_LEVEL" != "info" ]]; then
    echo "FAIL: LOG_LEVEL is '$LOG_LEVEL', expected 'info'"
    exit 1
fi

echo "PASS: ConfigMap 'app-config' created correctly with all key-value pairs"
exit 0
