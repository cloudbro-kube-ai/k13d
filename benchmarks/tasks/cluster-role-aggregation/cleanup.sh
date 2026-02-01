#!/bin/bash
kubectl delete namespace monitoring-system --ignore-not-found=true
kubectl delete clusterrole monitoring-view --ignore-not-found=true
kubectl delete clusterrole monitoring-logs --ignore-not-found=true
kubectl delete clusterrole monitoring-aggregate --ignore-not-found=true
kubectl delete clusterrolebinding monitoring-aggregate-binding --ignore-not-found=true
