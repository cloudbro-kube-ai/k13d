#!/bin/bash
kubectl delete namespace monitoring --ignore-not-found=true
kubectl delete namespace app-frontend --ignore-not-found=true
kubectl delete namespace app-backend --ignore-not-found=true
kubectl delete clusterrole pod-reader --ignore-not-found=true
