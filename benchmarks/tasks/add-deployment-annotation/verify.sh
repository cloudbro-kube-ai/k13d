#!/bin/bash
set -e

# Check if deployment exists
if ! kubectl get deployment web-app -n benchmark &>/dev/null; then
    echo "FAIL: Deployment 'web-app' not found in namespace 'benchmark'"
    exit 1
fi

# Check for the description annotation
ANNOTATION_VALUE=$(kubectl get deployment web-app -n benchmark -o jsonpath='{.metadata.annotations.description}')
if [[ "$ANNOTATION_VALUE" != "Web application frontend" ]]; then
    echo "FAIL: Annotation 'description' is '$ANNOTATION_VALUE', expected 'Web application frontend'"
    exit 1
fi

echo "PASS: Annotation 'description=Web application frontend' successfully added to deployment 'web-app'"
exit 0
