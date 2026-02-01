#!/bin/bash
# Verifier script for job-parallelism task

set -e

echo "Verifying job-parallelism task..."

NAMESPACE="job-test"

# Check if job exists
if ! kubectl get job parallel-job --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Job 'parallel-job' not found"
    exit 1
fi

# Check completions
COMPLETIONS=$(kubectl get job parallel-job --namespace="$NAMESPACE" -o jsonpath='{.spec.completions}')
if [ "$COMPLETIONS" != "6" ]; then
    echo "ERROR: Job completions should be 6, got '$COMPLETIONS'"
    exit 1
fi

# Check parallelism
PARALLELISM=$(kubectl get job parallel-job --namespace="$NAMESPACE" -o jsonpath='{.spec.parallelism}')
if [ "$PARALLELISM" != "3" ]; then
    echo "ERROR: Job parallelism should be 3, got '$PARALLELISM'"
    exit 1
fi

# Check completionMode
COMPLETION_MODE=$(kubectl get job parallel-job --namespace="$NAMESPACE" -o jsonpath='{.spec.completionMode}')
if [ "$COMPLETION_MODE" != "Indexed" ]; then
    echo "ERROR: completionMode should be 'Indexed', got '$COMPLETION_MODE'"
    exit 1
fi

# Check restartPolicy
RESTART_POLICY=$(kubectl get job parallel-job --namespace="$NAMESPACE" -o jsonpath='{.spec.template.spec.restartPolicy}')
if [ "$RESTART_POLICY" != "Never" ]; then
    echo "ERROR: restartPolicy should be 'Never', got '$RESTART_POLICY'"
    exit 1
fi

echo "Verification PASSED: Job 'parallel-job' created with parallelism=3 and completions=6"
exit 0
