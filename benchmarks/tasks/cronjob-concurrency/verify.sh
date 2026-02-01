#!/bin/bash
# Verifier script for cronjob-concurrency task

set -e

echo "Verifying cronjob-concurrency task..."

NAMESPACE="job-test"

# Check if cronjob exists
if ! kubectl get cronjob report-job --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: CronJob 'report-job' not found"
    exit 1
fi

# Check concurrencyPolicy
CONCURRENCY=$(kubectl get cronjob report-job --namespace="$NAMESPACE" -o jsonpath='{.spec.concurrencyPolicy}')
if [ "$CONCURRENCY" != "Forbid" ]; then
    echo "ERROR: concurrencyPolicy should be 'Forbid', got '$CONCURRENCY'"
    exit 1
fi

# Check successfulJobsHistoryLimit
SUCCESS_LIMIT=$(kubectl get cronjob report-job --namespace="$NAMESPACE" -o jsonpath='{.spec.successfulJobsHistoryLimit}')
if [ "$SUCCESS_LIMIT" != "3" ]; then
    echo "ERROR: successfulJobsHistoryLimit should be 3, got '$SUCCESS_LIMIT'"
    exit 1
fi

# Check failedJobsHistoryLimit
FAILED_LIMIT=$(kubectl get cronjob report-job --namespace="$NAMESPACE" -o jsonpath='{.spec.failedJobsHistoryLimit}')
if [ "$FAILED_LIMIT" != "1" ]; then
    echo "ERROR: failedJobsHistoryLimit should be 1, got '$FAILED_LIMIT'"
    exit 1
fi

# Check schedule
SCHEDULE=$(kubectl get cronjob report-job --namespace="$NAMESPACE" -o jsonpath='{.spec.schedule}')
if [ "$SCHEDULE" != "*/10 * * * *" ]; then
    echo "ERROR: schedule should be '*/10 * * * *', got '$SCHEDULE'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get cronjob report-job --namespace="$NAMESPACE" -o jsonpath='{.spec.jobTemplate.spec.template.spec.containers[0].image}')
if [[ "$IMAGE" != *"busybox"* ]]; then
    echo "ERROR: CronJob should use busybox image, got $IMAGE"
    exit 1
fi

echo "Verification PASSED: CronJob 'report-job' created with concurrencyPolicy=Forbid"
exit 0
