#!/bin/bash
# Verifier script for cronjob-suspend task

set -e

echo "Verifying cronjob-suspend task..."

NAMESPACE="job-test"

# Check if cronjob exists
if ! kubectl get cronjob cleanup-job --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: CronJob 'cleanup-job' not found"
    exit 1
fi

# Check if suspend is true
SUSPENDED=$(kubectl get cronjob cleanup-job --namespace="$NAMESPACE" -o jsonpath='{.spec.suspend}')
if [ "$SUSPENDED" != "true" ]; then
    echo "ERROR: CronJob should be suspended (spec.suspend=true), got '$SUSPENDED'"
    exit 1
fi

echo "Verification PASSED: CronJob 'cleanup-job' is suspended"
exit 0
