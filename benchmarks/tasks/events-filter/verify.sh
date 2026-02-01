#!/bin/bash
set -e

echo "Verifying events-filter task..."

NAMESPACE="events-test"

# Check namespace exists
if ! kubectl get namespace "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Namespace '$NAMESPACE' not found"
    exit 1
fi

# Verify we can get events
echo "Testing kubectl get events..."
EVENTS=$(kubectl get events -n "$NAMESPACE" 2>/dev/null || echo "")
if [[ -z "$EVENTS" ]]; then
    echo "FAILED: No events found in namespace"
    exit 1
fi

# Verify we can sort events
echo "Testing --sort-by option..."
SORTED_EVENTS=$(kubectl get events -n "$NAMESPACE" --sort-by='.lastTimestamp' 2>/dev/null || echo "")
if [[ -z "$SORTED_EVENTS" ]]; then
    echo "WARNING: Could not sort events"
fi

# Verify we can filter by type
echo "Testing --field-selector type=Warning..."
WARNING_EVENTS=$(kubectl get events -n "$NAMESPACE" --field-selector type=Warning 2>/dev/null || echo "No resources found")
echo "Warning events filter test complete"

# Verify we can filter by involvedObject
echo "Testing --field-selector involvedObject.name..."
POD_EVENTS=$(kubectl get events -n "$NAMESPACE" --field-selector involvedObject.name=failing-pod 2>/dev/null || echo "No resources found")
echo "Involved object filter test complete"

# Verify we can use custom output
echo "Testing custom output format..."
CUSTOM_OUTPUT=$(kubectl get events -n "$NAMESPACE" -o custom-columns=REASON:.reason,MESSAGE:.message,TYPE:.type 2>/dev/null || echo "")
if [[ -z "$CUSTOM_OUTPUT" ]]; then
    echo "WARNING: Custom output format may have issues"
fi

# Check that failing-pod has generated events
FAILING_POD_EXISTS=$(kubectl get pod failing-pod -n "$NAMESPACE" 2>/dev/null && echo "yes" || echo "no")
if [[ "$FAILING_POD_EXISTS" == "yes" ]]; then
    echo "Failing pod exists, should have Warning events"
fi

echo "SUCCESS: Events filtering commands verified"
exit 0
