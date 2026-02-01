#!/bin/bash
set -euo pipefail

NAMESPACE="zone-spread"

echo "Verifying zone-spread-ha..."

# Check web-app zone spread constraint
WEB_ZONE_SKEW=$(kubectl get deployment web-app -n $NAMESPACE -o json | jq -r '.spec.template.spec.topologySpreadConstraints[] | select(.topologyKey == "topology.kubernetes.io/zone") | .maxSkew // empty')
if [[ "$WEB_ZONE_SKEW" != "1" ]]; then
    echo "ERROR: web-app zone spread maxSkew should be 1"
    exit 1
fi

WEB_ZONE_UNSATISFIABLE=$(kubectl get deployment web-app -n $NAMESPACE -o json | jq -r '.spec.template.spec.topologySpreadConstraints[] | select(.topologyKey == "topology.kubernetes.io/zone") | .whenUnsatisfiable // empty')
if [[ "$WEB_ZONE_UNSATISFIABLE" != "DoNotSchedule" ]]; then
    echo "ERROR: web-app zone spread whenUnsatisfiable should be DoNotSchedule"
    exit 1
fi

WEB_ZONE_SELECTOR=$(kubectl get deployment web-app -n $NAMESPACE -o json | jq -r '.spec.template.spec.topologySpreadConstraints[] | select(.topologyKey == "topology.kubernetes.io/zone") | .labelSelector.matchLabels.app // empty')
if [[ "$WEB_ZONE_SELECTOR" != "web-app" ]]; then
    echo "ERROR: web-app zone spread should select app=web-app"
    exit 1
fi

# Check web-app hostname spread constraint
WEB_HOST_SKEW=$(kubectl get deployment web-app -n $NAMESPACE -o json | jq -r '.spec.template.spec.topologySpreadConstraints[] | select(.topologyKey == "kubernetes.io/hostname") | .maxSkew // empty')
if [[ "$WEB_HOST_SKEW" != "1" ]]; then
    echo "ERROR: web-app hostname spread maxSkew should be 1"
    exit 1
fi

WEB_HOST_UNSATISFIABLE=$(kubectl get deployment web-app -n $NAMESPACE -o json | jq -r '.spec.template.spec.topologySpreadConstraints[] | select(.topologyKey == "kubernetes.io/hostname") | .whenUnsatisfiable // empty')
if [[ "$WEB_HOST_UNSATISFIABLE" != "ScheduleAnyway" ]]; then
    echo "ERROR: web-app hostname spread whenUnsatisfiable should be ScheduleAnyway"
    exit 1
fi

# Check database zone spread
DB_ZONE_SKEW=$(kubectl get statefulset database -n $NAMESPACE -o json | jq -r '.spec.template.spec.topologySpreadConstraints[] | select(.topologyKey == "topology.kubernetes.io/zone") | .maxSkew // empty')
if [[ "$DB_ZONE_SKEW" != "1" ]]; then
    echo "ERROR: database zone spread maxSkew should be 1"
    exit 1
fi

DB_ZONE_UNSATISFIABLE=$(kubectl get statefulset database -n $NAMESPACE -o json | jq -r '.spec.template.spec.topologySpreadConstraints[] | select(.topologyKey == "topology.kubernetes.io/zone") | .whenUnsatisfiable // empty')
if [[ "$DB_ZONE_UNSATISFIABLE" != "DoNotSchedule" ]]; then
    echo "ERROR: database zone spread whenUnsatisfiable should be DoNotSchedule"
    exit 1
fi

DB_ZONE_SELECTOR=$(kubectl get statefulset database -n $NAMESPACE -o json | jq -r '.spec.template.spec.topologySpreadConstraints[] | select(.topologyKey == "topology.kubernetes.io/zone") | .labelSelector.matchLabels.app // empty')
if [[ "$DB_ZONE_SELECTOR" != "database" ]]; then
    echo "ERROR: database zone spread should select app=database"
    exit 1
fi

# Check background zone spread
BG_ZONE_SKEW=$(kubectl get deployment background -n $NAMESPACE -o json | jq -r '.spec.template.spec.topologySpreadConstraints[] | select(.topologyKey == "topology.kubernetes.io/zone") | .maxSkew // empty')
if [[ "$BG_ZONE_SKEW" != "2" ]]; then
    echo "ERROR: background zone spread maxSkew should be 2"
    exit 1
fi

BG_ZONE_UNSATISFIABLE=$(kubectl get deployment background -n $NAMESPACE -o json | jq -r '.spec.template.spec.topologySpreadConstraints[] | select(.topologyKey == "topology.kubernetes.io/zone") | .whenUnsatisfiable // empty')
if [[ "$BG_ZONE_UNSATISFIABLE" != "ScheduleAnyway" ]]; then
    echo "ERROR: background zone spread whenUnsatisfiable should be ScheduleAnyway"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Topology spread constraints configured correctly for zone distribution."
exit 0
