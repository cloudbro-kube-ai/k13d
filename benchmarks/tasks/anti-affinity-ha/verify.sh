#!/bin/bash
set -euo pipefail

NAMESPACE="affinity-ha"

echo "Verifying anti-affinity-ha..."

# Check critical-api has required anti-affinity
CRITICAL_REQUIRED=$(kubectl get deployment critical-api -n $NAMESPACE -o json | jq -r '.spec.template.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].labelSelector.matchLabels.app // empty')
if [[ "$CRITICAL_REQUIRED" != "critical-api" ]]; then
    echo "ERROR: critical-api should have required podAntiAffinity for app=critical-api"
    exit 1
fi

CRITICAL_TOPO=$(kubectl get deployment critical-api -n $NAMESPACE -o json | jq -r '.spec.template.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey // empty')
if [[ "$CRITICAL_TOPO" != "kubernetes.io/hostname" ]]; then
    echo "ERROR: critical-api anti-affinity should use topologyKey kubernetes.io/hostname"
    exit 1
fi

# Check worker has preferred anti-affinity
WORKER_PREFERRED=$(kubectl get deployment worker -n $NAMESPACE -o json | jq -r '.spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchLabels.app // empty')
if [[ "$WORKER_PREFERRED" != "worker" ]]; then
    echo "ERROR: worker should have preferred podAntiAffinity for app=worker"
    exit 1
fi

WORKER_WEIGHT=$(kubectl get deployment worker -n $NAMESPACE -o json | jq -r '.spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].weight // empty')
if [[ "$WORKER_WEIGHT" != "100" ]]; then
    echo "ERROR: worker anti-affinity weight should be 100, got '$WORKER_WEIGHT'"
    exit 1
fi

WORKER_TOPO=$(kubectl get deployment worker -n $NAMESPACE -o json | jq -r '.spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.topologyKey // empty')
if [[ "$WORKER_TOPO" != "kubernetes.io/hostname" ]]; then
    echo "ERROR: worker anti-affinity should use topologyKey kubernetes.io/hostname"
    exit 1
fi

# Check cache has required anti-affinity against critical-api
CACHE_REQUIRED=$(kubectl get statefulset cache -n $NAMESPACE -o json | jq -r '.spec.template.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].labelSelector.matchLabels.app // empty')
if [[ "$CACHE_REQUIRED" != "critical-api" ]]; then
    echo "ERROR: cache should have required podAntiAffinity against app=critical-api"
    exit 1
fi

CACHE_TOPO=$(kubectl get statefulset cache -n $NAMESPACE -o json | jq -r '.spec.template.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey // empty')
if [[ "$CACHE_TOPO" != "kubernetes.io/hostname" ]]; then
    echo "ERROR: cache anti-affinity should use topologyKey kubernetes.io/hostname"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Pod anti-affinity rules configured correctly for high availability."
exit 0
