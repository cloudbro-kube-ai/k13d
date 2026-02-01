#!/bin/bash
set -e

NAMESPACE="sts-recreate"

echo "Setting up recreate-statefulset task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

# Create StatefulSet with original selector
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: v1
kind: Service
metadata:
  name: database
spec:
  clusterIP: None
  selector:
    app: database
  ports:
  - port: 6379
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: database
spec:
  serviceName: database
  replicas: 2
  podManagementPolicy: OrderedReady
  selector:
    matchLabels:
      app: database
  template:
    metadata:
      labels:
        app: database
    spec:
      containers:
      - name: db
        image: redis:alpine
        ports:
        - containerPort: 6379
        volumeMounts:
        - name: data
          mountPath: /data
        resources:
          requests:
            memory: "32Mi"
            cpu: "25m"
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 100Mi
EOF

echo "Waiting for StatefulSet pods..."
kubectl wait --for=condition=Ready pod database-0 -n $NAMESPACE --timeout=120s || true
kubectl wait --for=condition=Ready pod database-1 -n $NAMESPACE --timeout=120s || true

# Write test data to verify persistence
kubectl exec database-0 -n $NAMESPACE -- sh -c "echo 'important-data-0' > /data/test.txt" || true
kubectl exec database-1 -n $NAMESPACE -- sh -c "echo 'important-data-1' > /data/test.txt" || true

echo "Setup complete. The StatefulSet has data in PVCs. Recreate it with new selector while preserving data."
