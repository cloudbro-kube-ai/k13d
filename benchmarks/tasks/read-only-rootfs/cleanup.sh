#!/bin/bash
echo "Cleaning up read-only-rootfs task..."
kubectl delete namespace readonly-fs-test --ignore-not-found=true
echo "Cleanup complete."
