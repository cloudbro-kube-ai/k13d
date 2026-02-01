#!/bin/bash
# Setup script for diagnose-node-issue task
# This task is an analysis task - no specific broken state to set up

set -e

echo "Setting up diagnose-node-issue task..."

# This is an analysis task that works with whatever nodes exist
# in the cluster. Just verify we have at least one node.

NODE_COUNT=$(kubectl get nodes --no-headers 2>/dev/null | wc -l)

if [ "$NODE_COUNT" -eq 0 ]; then
    echo "WARNING: No nodes found in cluster. This task requires at least one node."
    exit 1
fi

echo "Setup complete. Found $NODE_COUNT node(s) in the cluster."
echo "The AI agent should analyze node health and provide recommendations."
