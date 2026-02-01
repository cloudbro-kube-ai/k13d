#!/bin/bash
set -e

NAMESPACE="dev-team"

echo "Setting up rbac-audit task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create ServiceAccount
kubectl create serviceaccount dev-sa -n $NAMESPACE

# Create overly permissive Role (the problem to fix)
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: developer-role
  namespace: $NAMESPACE
rules:
- apiGroups: [""]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "list", "watch", "update", "delete"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["*"]
EOF

# Create RoleBinding
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: developer-binding
  namespace: $NAMESPACE
subjects:
- kind: ServiceAccount
  name: dev-sa
  namespace: $NAMESPACE
roleRef:
  kind: Role
  name: developer-role
  apiGroup: rbac.authorization.k8s.io
EOF

echo "Setup complete. The 'developer-role' has overly permissive access. Fix it!"
