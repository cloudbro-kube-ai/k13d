#!/bin/bash
# Setup script for cronjob-suspend task

set -e

echo "Setting up cronjob-suspend task..."

# Create namespace if not exists
kubectl create namespace job-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing cronjob
kubectl delete cronjob cleanup-job --namespace=job-test --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create an active CronJob
cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cleanup-job
  namespace: job-test
spec:
  schedule: "*/5 * * * *"
  suspend: false
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: cleanup
            image: busybox
            command: ["sh", "-c", "echo Cleaning up..."]
          restartPolicy: OnFailure
EOF

echo "Setup complete. CronJob 'cleanup-job' is active (not suspended)."
