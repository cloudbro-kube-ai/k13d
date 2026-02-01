#!/bin/bash
kubectl delete namespace storage-tiers --ignore-not-found=true
kubectl delete storageclass ssd-immediate --ignore-not-found=true
kubectl delete storageclass hdd-topology --ignore-not-found=true
kubectl delete storageclass encrypted-storage --ignore-not-found=true
