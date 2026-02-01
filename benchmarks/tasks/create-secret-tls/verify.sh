#!/bin/bash
set -e

# Check if Secret exists
if ! kubectl get secret tls-secret -n benchmark &>/dev/null; then
    echo "FAIL: Secret 'tls-secret' not found in namespace 'benchmark'"
    exit 1
fi

# Check Secret type
SECRET_TYPE=$(kubectl get secret tls-secret -n benchmark -o jsonpath='{.type}')
if [[ "$SECRET_TYPE" != "kubernetes.io/tls" ]]; then
    echo "FAIL: Secret type is '$SECRET_TYPE', expected 'kubernetes.io/tls'"
    exit 1
fi

# Check that tls.crt and tls.key exist in the secret
TLS_CRT=$(kubectl get secret tls-secret -n benchmark -o jsonpath='{.data.tls\.crt}')
TLS_KEY=$(kubectl get secret tls-secret -n benchmark -o jsonpath='{.data.tls\.key}')

if [[ -z "$TLS_CRT" ]]; then
    echo "FAIL: 'tls.crt' not found in secret"
    exit 1
fi

if [[ -z "$TLS_KEY" ]]; then
    echo "FAIL: 'tls.key' not found in secret"
    exit 1
fi

echo "PASS: TLS Secret 'tls-secret' created correctly with tls.crt and tls.key"
exit 0
