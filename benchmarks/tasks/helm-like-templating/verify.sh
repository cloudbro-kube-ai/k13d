#!/bin/bash
set -euo pipefail

NAMESPACE="templating-demo"
TIMEOUT="120s"

echo "Verifying helm-like-templating..."

# Check app-values ConfigMap
if ! kubectl get configmap app-values -n $NAMESPACE &>/dev/null; then
    echo "ERROR: ConfigMap 'app-values' not found"
    exit 1
fi

# Verify ConfigMap values
APP_NAME=$(kubectl get configmap app-values -n $NAMESPACE -o jsonpath='{.data.APP_NAME}')
if [[ "$APP_NAME" != "mywebapp" ]]; then
    echo "ERROR: APP_NAME should be 'mywebapp', got '$APP_NAME'"
    exit 1
fi

APP_VERSION=$(kubectl get configmap app-values -n $NAMESPACE -o jsonpath='{.data.APP_VERSION}')
if [[ "$APP_VERSION" != "2.1.0" ]]; then
    echo "ERROR: APP_VERSION should be '2.1.0', got '$APP_VERSION'"
    exit 1
fi

REPLICAS=$(kubectl get configmap app-values -n $NAMESPACE -o jsonpath='{.data.REPLICAS}')
if [[ "$REPLICAS" != "3" ]]; then
    echo "ERROR: REPLICAS should be '3', got '$REPLICAS'"
    exit 1
fi

IMAGE=$(kubectl get configmap app-values -n $NAMESPACE -o jsonpath='{.data.IMAGE}')
if [[ "$IMAGE" != "nginx:1.25-alpine" ]]; then
    echo "ERROR: IMAGE should be 'nginx:1.25-alpine', got '$IMAGE'"
    exit 1
fi

# Check app-config ConfigMap
if ! kubectl get configmap app-config -n $NAMESPACE &>/dev/null; then
    echo "ERROR: ConfigMap 'app-config' not found"
    exit 1
fi

CONFIG_YAML=$(kubectl get configmap app-config -n $NAMESPACE -o jsonpath='{.data.config\.yaml}')
if [[ -z "$CONFIG_YAML" ]]; then
    echo "ERROR: app-config should have config.yaml"
    exit 1
fi

# Check Deployment
if ! kubectl get deployment templated-app -n $NAMESPACE &>/dev/null; then
    echo "ERROR: Deployment 'templated-app' not found"
    exit 1
fi

# Verify deployment uses ConfigMap values
DEPLOY_REPLICAS=$(kubectl get deployment templated-app -n $NAMESPACE -o jsonpath='{.spec.replicas}')
if [[ "$DEPLOY_REPLICAS" != "3" ]]; then
    echo "ERROR: Deployment replicas should be 3, got '$DEPLOY_REPLICAS'"
    exit 1
fi

DEPLOY_IMAGE=$(kubectl get deployment templated-app -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}')
if [[ "$DEPLOY_IMAGE" != "nginx:1.25-alpine" ]]; then
    echo "ERROR: Deployment image should be nginx:1.25-alpine, got '$DEPLOY_IMAGE'"
    exit 1
fi

# Check envFrom or env with configMapKeyRef
ENV_FROM=$(kubectl get deployment templated-app -n $NAMESPACE -o json | jq -r '.spec.template.spec.containers[0].envFrom[].configMapRef.name // empty' 2>/dev/null | head -1)
ENV_REF=$(kubectl get deployment templated-app -n $NAMESPACE -o json | jq -r '.spec.template.spec.containers[0].env[].valueFrom.configMapKeyRef.name // empty' 2>/dev/null | head -1)

if [[ -z "$ENV_FROM" ]] && [[ -z "$ENV_REF" ]]; then
    echo "ERROR: Deployment should use envFrom or valueFrom with app-values ConfigMap"
    exit 1
fi

# Check volume mount for app-config
VOLUME_MOUNT=$(kubectl get deployment templated-app -n $NAMESPACE -o json | jq -r '.spec.template.spec.containers[0].volumeMounts[] | select(.mountPath == "/etc/config") | .name // empty')
if [[ -z "$VOLUME_MOUNT" ]]; then
    echo "ERROR: Deployment should mount app-config at /etc/config"
    exit 1
fi

# Check labels
LABEL_NAME=$(kubectl get deployment templated-app -n $NAMESPACE -o jsonpath='{.spec.template.metadata.labels.app\.kubernetes\.io/name}')
if [[ "$LABEL_NAME" != "mywebapp" ]]; then
    echo "ERROR: Pod label app.kubernetes.io/name should be 'mywebapp'"
    exit 1
fi

LABEL_VERSION=$(kubectl get deployment templated-app -n $NAMESPACE -o jsonpath='{.spec.template.metadata.labels.app\.kubernetes\.io/version}')
if [[ "$LABEL_VERSION" != "2.1.0" ]]; then
    echo "ERROR: Pod label app.kubernetes.io/version should be '2.1.0'"
    exit 1
fi

LABEL_ENV=$(kubectl get deployment templated-app -n $NAMESPACE -o jsonpath='{.spec.template.metadata.labels.app\.kubernetes\.io/env}')
if [[ "$LABEL_ENV" != "production" ]]; then
    echo "ERROR: Pod label app.kubernetes.io/env should be 'production'"
    exit 1
fi

# Check resource limits
MEM_LIMIT=$(kubectl get deployment templated-app -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].resources.limits.memory}')
if [[ "$MEM_LIMIT" != "128Mi" ]]; then
    echo "ERROR: Memory limit should be 128Mi, got '$MEM_LIMIT'"
    exit 1
fi

CPU_LIMIT=$(kubectl get deployment templated-app -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].resources.limits.cpu}')
if [[ "$CPU_LIMIT" != "200m" ]]; then
    echo "ERROR: CPU limit should be 200m, got '$CPU_LIMIT'"
    exit 1
fi

# Check Service
if ! kubectl get service templated-app -n $NAMESPACE &>/dev/null; then
    echo "ERROR: Service 'templated-app' not found"
    exit 1
fi

SVC_PORT=$(kubectl get service templated-app -n $NAMESPACE -o jsonpath='{.spec.ports[0].port}')
if [[ "$SVC_PORT" != "8080" ]]; then
    echo "ERROR: Service port should be 8080, got '$SVC_PORT'"
    exit 1
fi

# Verify deployment is available
kubectl wait --for=condition=Available deployment/templated-app -n $NAMESPACE --timeout=$TIMEOUT || {
    echo "ERROR: Deployment not available"
    exit 1
}

echo "--- Verification Successful! ---"
echo "Helm-like templating system configured correctly."
exit 0
