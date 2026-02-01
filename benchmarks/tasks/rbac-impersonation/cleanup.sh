#!/bin/bash
kubectl delete namespace tenant-platform --ignore-not-found=true
kubectl delete clusterrole impersonation-role --ignore-not-found=true
kubectl delete clusterrolebinding platform-impersonation --ignore-not-found=true
