#!/bin/bash
# Package Helm chart for goreleaser release
# Usage: scripts/package-helm.sh <version>
# Creates k13d-<version>.tgz in .helm-out/

set -e

VERSION="${1:?Usage: package-helm.sh <version>}"

CHART_DIR="deploy/helm/k13d"
CHART_YAML="${CHART_DIR}/Chart.yaml"

# Update Chart.yaml with release version
sed -i.bak "s/^version:.*/version: ${VERSION}/" "${CHART_YAML}"
sed -i.bak "s/^appVersion:.*/appVersion: \"${VERSION}\"/" "${CHART_YAML}"
rm -f "${CHART_YAML}.bak"

mkdir -p .helm-out

# Use helm if available, otherwise fallback to tar
if command -v helm &> /dev/null; then
    helm package "${CHART_DIR}" --destination .helm-out/
else
    # Helm chart tgz is just a tarball of the chart directory
    tar -czf ".helm-out/k13d-${VERSION}.tgz" -C deploy/helm k13d
fi

echo "Helm chart packaged: .helm-out/k13d-${VERSION}.tgz"
