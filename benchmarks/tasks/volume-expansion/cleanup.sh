#!/bin/bash
kubectl delete namespace expansion-demo --ignore-not-found=true
kubectl delete storageclass expandable-sc --ignore-not-found=true
kubectl delete pv expansion-pv expansion-pv-2 --ignore-not-found=true
rm -rf /tmp/expansion-pv /tmp/expansion-pv-2 2>/dev/null || true
