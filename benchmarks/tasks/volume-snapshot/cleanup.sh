#!/bin/bash
kubectl delete namespace snapshot-demo --ignore-not-found=true
kubectl delete volumesnapshotclass csi-snapshot-class --ignore-not-found=true 2>/dev/null || true
