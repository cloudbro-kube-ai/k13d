#!/bin/bash
kubectl delete namespace storage-demo --ignore-not-found=true
kubectl delete storageclass fast-storage --ignore-not-found=true
kubectl delete pv fast-pv-1 --ignore-not-found=true
rm -rf /tmp/k13d-storage 2>/dev/null || true
