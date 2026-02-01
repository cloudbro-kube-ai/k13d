#!/bin/bash
set -e

# Check if pod exists
if ! kubectl get pod args-pod -n benchmark &>/dev/null; then
    echo "FAIL: Pod 'args-pod' not found in namespace 'benchmark'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get pod args-pod -n benchmark -o jsonpath='{.spec.containers[0].image}')
if [[ "$IMAGE" != "busybox:1.35" ]]; then
    echo "FAIL: Pod image is '$IMAGE', expected 'busybox:1.35'"
    exit 1
fi

# Check command and args
COMMAND=$(kubectl get pod args-pod -n benchmark -o jsonpath='{.spec.containers[0].command}')
ARGS=$(kubectl get pod args-pod -n benchmark -o jsonpath='{.spec.containers[0].args}')

# Check if echo command is present and args contain the expected text
COMBINED="$COMMAND $ARGS"
if [[ "$COMBINED" =~ "echo" ]] && [[ "$COMBINED" =~ "Hello" ]] && [[ "$COMBINED" =~ "Kubernetes" ]] && [[ "$COMBINED" =~ "Benchmark" ]]; then
    echo "PASS: Pod 'args-pod' created correctly with echo command and arguments"
    exit 0
fi

echo "FAIL: Pod command/args do not match expected. Command: $COMMAND, Args: $ARGS"
exit 1
