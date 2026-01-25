#!/bin/bash
echo "Cleaning up rolling-update task..."
kubectl delete namespace rolling --ignore-not-found=true
echo "Cleanup complete."
