package web

import (
	"encoding/json"
	"net/http"
)

// ResourceTemplate represents a built-in Kubernetes resource template
type ResourceTemplate struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	YAML        string `json:"yaml"`
}

var builtinTemplates = []ResourceTemplate{
	{
		Name:        "Nginx Deployment",
		Category:    "Web Server",
		Description: "A basic Nginx web server deployment with 2 replicas",
		YAML: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.27
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 250m
            memory: 256Mi`,
	},
	{
		Name:        "Redis StatefulSet",
		Category:    "Database",
		Description: "Redis StatefulSet with persistent storage",
		YAML: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis
spec:
  serviceName: redis
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 250m
            memory: 256Mi
        volumeMounts:
        - name: redis-data
          mountPath: /data
  volumeClaimTemplates:
  - metadata:
      name: redis-data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 1Gi`,
	},
	{
		Name:        "PostgreSQL Deployment",
		Category:    "Database",
		Description: "PostgreSQL database with persistent volume claim",
		YAML: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  labels:
    app: postgres
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:16-alpine
        ports:
        - containerPort: 5432
        env:
        - name: POSTGRES_DB
          value: mydb
        - name: POSTGRES_USER
          value: admin
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        resources:
          requests:
            cpu: 250m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
        volumeMounts:
        - name: postgres-data
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: postgres-data
        persistentVolumeClaim:
          claimName: postgres-pvc`,
	},
	{
		Name:        "CronJob Example",
		Category:    "Batch",
		Description: "A CronJob that runs every hour",
		YAML: `apiVersion: batch/v1
kind: CronJob
metadata:
  name: hourly-job
spec:
  schedule: "0 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: job
            image: busybox:1.36
            command:
            - /bin/sh
            - -c
            - echo "Running scheduled task at $(date)"
            resources:
              requests:
                cpu: 50m
                memory: 64Mi
              limits:
                cpu: 100m
                memory: 128Mi
          restartPolicy: OnFailure
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1`,
	},
	{
		Name:        "Service + Ingress",
		Category:    "Networking",
		Description: "ClusterIP Service with Ingress for external access",
		YAML: `apiVersion: v1
kind: Service
metadata:
  name: web-service
spec:
  type: ClusterIP
  selector:
    app: web
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: web-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
  - host: app.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: web-service
            port:
              number: 80`,
	},
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": builtinTemplates,
	})
}

func (s *Server) handleTemplateApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TemplateName string `json:"templateName"`
		Namespace    string `json:"namespace"`
		YAML         string `json:"yaml"` // Allow custom YAML override
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	yamlContent := req.YAML

	// If no custom YAML, find the template by name
	if yamlContent == "" {
		for _, t := range builtinTemplates {
			if t.Name == req.TemplateName {
				yamlContent = t.YAML
				break
			}
		}
	}

	if yamlContent == "" {
		http.Error(w, "Template not found: "+req.TemplateName, http.StatusNotFound)
		return
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	result, err := s.k8sClient.ApplyYAML(r.Context(), yamlContent, namespace, false)
	if err != nil {
		http.Error(w, "Failed to apply template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result":  result,
	})
}
