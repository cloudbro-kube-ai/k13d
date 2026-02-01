#!/bin/bash
echo "Cleaning up port-forward task..."
# Kill any lingering port-forward processes
pkill -f "kubectl port-forward.*port-fwd-test" 2>/dev/null || true
kubectl delete namespace port-fwd-test --ignore-not-found=true
echo "Cleanup complete."
