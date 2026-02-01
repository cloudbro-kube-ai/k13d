#!/bin/bash
set -e

# Delete the Secret if it exists
kubectl delete secret encoded-secret -n benchmark --ignore-not-found=true

# Remove output file
rm -f /tmp/decoded-password.txt

echo "Cleanup complete"
