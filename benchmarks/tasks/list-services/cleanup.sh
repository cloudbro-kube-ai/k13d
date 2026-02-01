#!/bin/bash
set -e

# Delete Services
kubectl delete service frontend-svc backend-svc database-svc -n benchmark --ignore-not-found=true

# Remove output file
rm -f /tmp/services.txt

echo "Cleanup complete"
