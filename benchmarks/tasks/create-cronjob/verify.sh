#!/bin/bash
# Verifier script for create-cronjob task

set -e

echo "Verifying create-cronjob task..."

# Check if CronJob exists
if ! kubectl get cronjob backup-job --namespace="${NAMESPACE}" &>/dev/null; then
    echo "ERROR: CronJob 'backup-job' not found"
    exit 1
fi

# Check schedule
SCHEDULE=$(kubectl get cronjob backup-job --namespace="${NAMESPACE}" -o jsonpath='{.spec.schedule}')
if [ "$SCHEDULE" != "0 2 * * *" ]; then
    echo "WARNING: Schedule is '$SCHEDULE', expected '0 2 * * *'"
fi

# Check concurrency policy
CONCURRENCY=$(kubectl get cronjob backup-job --namespace="${NAMESPACE}" -o jsonpath='{.spec.concurrencyPolicy}')
if [ "$CONCURRENCY" != "Forbid" ]; then
    echo "WARNING: Concurrency policy is '$CONCURRENCY', expected 'Forbid'"
fi

# Check container image
IMAGE=$(kubectl get cronjob backup-job --namespace="${NAMESPACE}" -o jsonpath='{.spec.jobTemplate.spec.template.spec.containers[0].image}')
if [[ "$IMAGE" != *"busybox"* ]]; then
    echo "ERROR: Container image is '$IMAGE', expected busybox"
    exit 1
fi

# Check successful jobs history limit
SUCCESS_LIMIT=$(kubectl get cronjob backup-job --namespace="${NAMESPACE}" -o jsonpath='{.spec.successfulJobsHistoryLimit}')
if [ "$SUCCESS_LIMIT" != "3" ]; then
    echo "WARNING: Successful jobs history limit is '$SUCCESS_LIMIT', expected '3'"
fi

# Check failed jobs history limit
FAILED_LIMIT=$(kubectl get cronjob backup-job --namespace="${NAMESPACE}" -o jsonpath='{.spec.failedJobsHistoryLimit}')
if [ "$FAILED_LIMIT" != "1" ]; then
    echo "WARNING: Failed jobs history limit is '$FAILED_LIMIT', expected '1'"
fi

echo "Verification PASSED: CronJob 'backup-job' created successfully"
exit 0
