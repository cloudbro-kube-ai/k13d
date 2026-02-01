#!/bin/bash
set -e

# Delete the Secret if it exists
kubectl delete secret tls-secret -n benchmark --ignore-not-found=true

# Delete temp files
rm -f /tmp/tls.crt /tmp/tls.key

echo "Cleanup complete"
