#!/bin/bash
set -e

# Check if pod exists
if ! kubectl get pod env-pod -n benchmark &>/dev/null; then
    echo "FAIL: Pod 'env-pod' not found in namespace 'benchmark'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get pod env-pod -n benchmark -o jsonpath='{.spec.containers[0].image}')
if [[ "$IMAGE" != "nginx:alpine" ]]; then
    echo "FAIL: Pod image is '$IMAGE', expected 'nginx:alpine'"
    exit 1
fi

# Check environment variables
ENV_VARS=$(kubectl get pod env-pod -n benchmark -o jsonpath='{.spec.containers[0].env[*].name}')
if [[ ! "$ENV_VARS" =~ "APP_ENV" ]] || [[ ! "$ENV_VARS" =~ "APP_DEBUG" ]] || [[ ! "$ENV_VARS" =~ "APP_PORT" ]]; then
    echo "FAIL: Missing required environment variables (APP_ENV, APP_DEBUG, APP_PORT)"
    exit 1
fi

# Check environment variable values
APP_ENV=$(kubectl get pod env-pod -n benchmark -o jsonpath='{.spec.containers[0].env[?(@.name=="APP_ENV")].value}')
APP_DEBUG=$(kubectl get pod env-pod -n benchmark -o jsonpath='{.spec.containers[0].env[?(@.name=="APP_DEBUG")].value}')
APP_PORT=$(kubectl get pod env-pod -n benchmark -o jsonpath='{.spec.containers[0].env[?(@.name=="APP_PORT")].value}')

if [[ "$APP_ENV" != "production" ]]; then
    echo "FAIL: APP_ENV is '$APP_ENV', expected 'production'"
    exit 1
fi

if [[ "$APP_DEBUG" != "false" ]]; then
    echo "FAIL: APP_DEBUG is '$APP_DEBUG', expected 'false'"
    exit 1
fi

if [[ "$APP_PORT" != "8080" ]]; then
    echo "FAIL: APP_PORT is '$APP_PORT', expected '8080'"
    exit 1
fi

echo "PASS: Pod 'env-pod' created correctly with all environment variables"
exit 0
