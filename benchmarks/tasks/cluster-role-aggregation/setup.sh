#!/bin/bash
set -e

NAMESPACE="monitoring-system"

echo "Setting up cluster-role-aggregation task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true
kubectl delete clusterrole monitoring-view --ignore-not-found=true 2>/dev/null || true
kubectl delete clusterrole monitoring-logs --ignore-not-found=true 2>/dev/null || true
kubectl delete clusterrole monitoring-aggregate --ignore-not-found=true 2>/dev/null || true
kubectl delete clusterrolebinding monitoring-aggregate-binding --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

echo "Setup complete. Create the aggregated ClusterRole structure."
