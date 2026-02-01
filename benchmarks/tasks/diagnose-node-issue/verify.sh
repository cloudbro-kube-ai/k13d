#!/bin/bash
# Verifier script for diagnose-node-issue task
# This is an analysis task - verification checks for completeness of analysis

set -e

echo "Verifying diagnose-node-issue task..."

# For analysis tasks, we verify that the AI has gathered key information
# This verification is more lenient since it depends on AI output analysis

# Check that nodes exist (basic sanity check)
NODE_COUNT=$(kubectl get nodes --no-headers 2>/dev/null | wc -l)

if [ "$NODE_COUNT" -eq 0 ]; then
    echo "ERROR: No nodes found in cluster"
    exit 1
fi

# Since this is an analysis task, success is determined by whether
# the AI agent performed the diagnostic steps and provided analysis.
# The eval framework will check the AI's output against expect patterns.

echo "Verification PASSED: Cluster has $NODE_COUNT node(s)"
echo "Note: This is an analysis task. The AI agent should have:"
echo "  - Checked node conditions (kubectl get nodes -o wide)"
echo "  - Reviewed node details (kubectl describe node)"
echo "  - Analyzed events and resources"
echo "  - Provided recommendations"
exit 0
