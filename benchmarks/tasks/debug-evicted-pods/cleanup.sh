#!/bin/bash
kubectl delete namespace eviction-demo --ignore-not-found=true
kubectl delete priorityclass critical-priority low-priority --ignore-not-found=true
