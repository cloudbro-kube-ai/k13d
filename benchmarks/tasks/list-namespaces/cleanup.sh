#!/bin/bash
set -e

# Delete test namespaces
kubectl delete namespace test-ns-alpha test-ns-beta --ignore-not-found=true

# Remove output file
rm -f /tmp/namespaces.txt

echo "Cleanup complete"
