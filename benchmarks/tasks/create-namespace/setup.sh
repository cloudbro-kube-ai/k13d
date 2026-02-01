#!/bin/bash
set -e

# Ensure namespace doesn't exist
kubectl delete namespace production --ignore-not-found=true 2>/dev/null || true

echo "Setup complete: ready for namespace creation"
