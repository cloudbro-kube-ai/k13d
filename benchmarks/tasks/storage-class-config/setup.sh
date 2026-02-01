#!/bin/bash
set -e

NAMESPACE="storage-tiers"

echo "Setting up storage-class-config task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true
kubectl delete storageclass ssd-immediate --ignore-not-found=true 2>/dev/null || true
kubectl delete storageclass hdd-topology --ignore-not-found=true 2>/dev/null || true
kubectl delete storageclass encrypted-storage --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete. Create the StorageClasses with different configurations."
