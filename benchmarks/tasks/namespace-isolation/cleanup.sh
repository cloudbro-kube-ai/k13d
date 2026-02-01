#!/bin/bash
kubectl delete namespace tenant-alpha --ignore-not-found=true
kubectl delete namespace tenant-beta --ignore-not-found=true
kubectl delete namespace tenant-gamma --ignore-not-found=true
kubectl delete namespace shared-services --ignore-not-found=true
