#!/bin/bash
kubectl delete namespace reclaim-demo --ignore-not-found=true
kubectl delete pv pv-retain pv-delete pv-recycle --ignore-not-found=true
rm -rf /tmp/pv-retain /tmp/pv-delete /tmp/pv-recycle 2>/dev/null || true
