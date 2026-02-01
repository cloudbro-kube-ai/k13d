#!/bin/bash
set -e

# Check if namespace exists
if ! kubectl get namespace production &>/dev/null; then
    echo "FAIL: Namespace 'production' not found"
    exit 1
fi

# Check environment label
ENV_LABEL=$(kubectl get namespace production -o jsonpath='{.metadata.labels.environment}')
if [[ "$ENV_LABEL" != "production" ]]; then
    echo "FAIL: Label 'environment' is '$ENV_LABEL', expected 'production'"
    exit 1
fi

# Check team label
TEAM_LABEL=$(kubectl get namespace production -o jsonpath='{.metadata.labels.team}')
if [[ "$TEAM_LABEL" != "platform" ]]; then
    echo "FAIL: Label 'team' is '$TEAM_LABEL', expected 'platform'"
    exit 1
fi

echo "PASS: Namespace 'production' created with correct labels"
exit 0
