#!/bin/bash
set -e

NAMESPACE="expansion-demo"

echo "Setting up volume-expansion task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true
kubectl delete storageclass expandable-sc --ignore-not-found=true 2>/dev/null || true
kubectl delete pv expansion-pv --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create StorageClass WITHOUT expansion enabled (problem to fix)
cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: expandable-sc
provisioner: kubernetes.io/no-provisioner
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Delete
allowVolumeExpansion: false
EOF

# Create PV
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolume
metadata:
  name: expansion-pv
spec:
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Delete
  storageClassName: expandable-sc
  hostPath:
    path: /tmp/expansion-pv
  nodeAffinity:
    required:
      nodeSelectorTerms:
      - matchExpressions:
        - key: kubernetes.io/hostname
          operator: Exists
EOF

mkdir -p /tmp/expansion-pv 2>/dev/null || true

# Create initial PVC (1Gi)
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: app-storage
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: expandable-sc
  resources:
    requests:
      storage: 1Gi
EOF

# Create StatefulSet
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: data-app
spec:
  serviceName: data-app
  replicas: 1
  selector:
    matchLabels:
      app: data-app
  template:
    metadata:
      labels:
        app: data-app
    spec:
      containers:
      - name: app
        image: busybox
        command: ["sleep", "infinity"]
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: app-storage
EOF

echo "Waiting for StatefulSet to be ready..."
kubectl wait --for=condition=Ready pod -l app=data-app -n $NAMESPACE --timeout=120s || true

echo "Setup complete. The StorageClass needs allowVolumeExpansion enabled, and PVC needs expansion."
