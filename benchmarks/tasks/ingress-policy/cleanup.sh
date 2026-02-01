#!/bin/bash
kubectl delete namespace microservices --ignore-not-found=true
kubectl delete namespace external-service --ignore-not-found=true
