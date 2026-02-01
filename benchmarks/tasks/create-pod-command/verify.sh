#!/bin/bash
set -e

# Check if pod exists
if ! kubectl get pod command-pod -n benchmark &>/dev/null; then
    echo "FAIL: Pod 'command-pod' not found in namespace 'benchmark'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get pod command-pod -n benchmark -o jsonpath='{.spec.containers[0].image}')
if [[ "$IMAGE" != "busybox:1.35" ]]; then
    echo "FAIL: Pod image is '$IMAGE', expected 'busybox:1.35'"
    exit 1
fi

# Check command contains sleep 3600
COMMAND=$(kubectl get pod command-pod -n benchmark -o jsonpath='{.spec.containers[0].command}')
ARGS=$(kubectl get pod command-pod -n benchmark -o jsonpath='{.spec.containers[0].args}')

if [[ "$COMMAND" =~ "sleep" ]] && [[ "$COMMAND" =~ "3600" || "$ARGS" =~ "3600" ]]; then
    echo "PASS: Pod 'command-pod' created correctly with sleep 3600 command"
    exit 0
fi

if [[ "$ARGS" =~ "sleep" ]] && [[ "$ARGS" =~ "3600" ]]; then
    echo "PASS: Pod 'command-pod' created correctly with sleep 3600 command"
    exit 0
fi

# Alternative: check if command is ["sleep"] and args is ["3600"]
if [[ "$COMMAND" =~ "sleep" ]] && [[ "$ARGS" =~ "3600" ]]; then
    echo "PASS: Pod 'command-pod' created correctly with sleep 3600 command"
    exit 0
fi

# Alternative: check using sh -c
if [[ "$COMMAND" =~ "sh" ]] && [[ "$ARGS" =~ "sleep" ]] && [[ "$ARGS" =~ "3600" ]]; then
    echo "PASS: Pod 'command-pod' created correctly with sleep 3600 command"
    exit 0
fi

echo "FAIL: Pod command does not include 'sleep 3600'. Command: $COMMAND, Args: $ARGS"
exit 1
