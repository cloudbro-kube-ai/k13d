package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/drain"
)

// ==========================================
// Deployment Operations
// ==========================================

// DeploymentScaleRequest represents a scale request
type DeploymentScaleRequest struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Replicas  int32  `json:"replicas"`
}

// DeploymentRollbackRequest represents a rollback request
type DeploymentRollbackRequest struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Revision  int64  `json:"revision,omitempty"` // 0 means rollback to previous
}

// handleDeploymentScale handles POST /api/deployment/scale
func (s *Server) handleDeploymentScale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeploymentScaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Namespace == "" || req.Name == "" {
		http.Error(w, "namespace and name are required", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	ctx := r.Context()
	clientset := s.k8sClient.Clientset

	// Get current deployment
	deployment, err := clientset.AppsV1().Deployments(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get deployment: %v", err), http.StatusNotFound)
		return
	}

	oldReplicas := int32(0)
	if deployment.Spec.Replicas != nil {
		oldReplicas = *deployment.Spec.Replicas
	}

	// Scale the deployment
	deployment.Spec.Replicas = &req.Replicas
	_, err = clientset.AppsV1().Deployments(req.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to scale deployment: %v", err), http.StatusInternalServerError)
		return
	}

	// Record audit log
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "scale",
		Resource: fmt.Sprintf("deployment/%s", req.Name),
		Details:  fmt.Sprintf("Scaled from %d to %d replicas in namespace %s", oldReplicas, req.Replicas, req.Namespace),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     fmt.Sprintf("Deployment %s scaled to %d replicas", req.Name, req.Replicas),
		"oldReplicas": oldReplicas,
		"newReplicas": req.Replicas,
	})
}

// handleDeploymentRestart handles POST /api/deployment/restart
func (s *Server) handleDeploymentRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Namespace == "" || req.Name == "" {
		http.Error(w, "namespace and name are required", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	ctx := r.Context()
	clientset := s.k8sClient.Clientset

	// Patch deployment to trigger rollout restart
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err := clientset.AppsV1().Deployments(req.Namespace).Patch(ctx, req.Name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to restart deployment: %v", err), http.StatusInternalServerError)
		return
	}

	// Record audit log
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "restart",
		Resource: fmt.Sprintf("deployment/%s", req.Name),
		Details:  fmt.Sprintf("Triggered rollout restart in namespace %s", req.Namespace),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Deployment %s restart initiated", req.Name),
	})
}

// handleDeploymentPause handles POST /api/deployment/pause
func (s *Server) handleDeploymentPause(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Namespace == "" || req.Name == "" {
		http.Error(w, "namespace and name are required", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	ctx := r.Context()
	clientset := s.k8sClient.Clientset

	// Patch deployment to pause
	patch := `{"spec":{"paused":true}}`
	_, err := clientset.AppsV1().Deployments(req.Namespace).Patch(ctx, req.Name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to pause deployment: %v", err), http.StatusInternalServerError)
		return
	}

	// Record audit log
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "pause",
		Resource: fmt.Sprintf("deployment/%s", req.Name),
		Details:  fmt.Sprintf("Paused deployment in namespace %s", req.Namespace),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Deployment %s paused", req.Name),
	})
}

// handleDeploymentResume handles POST /api/deployment/resume
func (s *Server) handleDeploymentResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Namespace == "" || req.Name == "" {
		http.Error(w, "namespace and name are required", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	ctx := r.Context()
	clientset := s.k8sClient.Clientset

	// Patch deployment to resume
	patch := `{"spec":{"paused":false}}`
	_, err := clientset.AppsV1().Deployments(req.Namespace).Patch(ctx, req.Name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resume deployment: %v", err), http.StatusInternalServerError)
		return
	}

	// Record audit log
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "resume",
		Resource: fmt.Sprintf("deployment/%s", req.Name),
		Details:  fmt.Sprintf("Resumed deployment in namespace %s", req.Namespace),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Deployment %s resumed", req.Name),
	})
}

// handleDeploymentRollback handles POST /api/deployment/rollback
func (s *Server) handleDeploymentRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeploymentRollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Namespace == "" || req.Name == "" {
		http.Error(w, "namespace and name are required", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	ctx := r.Context()
	clientset := s.k8sClient.Clientset

	// Get deployment
	deployment, err := clientset.AppsV1().Deployments(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get deployment: %v", err), http.StatusNotFound)
		return
	}

	// Get ReplicaSets for this deployment
	rsList, err := clientset.AppsV1().ReplicaSets(req.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(deployment.Spec.Selector),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list ReplicaSets: %v", err), http.StatusInternalServerError)
		return
	}

	// Find the ReplicaSet to rollback to
	var targetRS *appsv1.ReplicaSet
	if req.Revision == 0 {
		// Rollback to previous revision
		var latestRevision, previousRevision int64
		var latestRS, previousRS *appsv1.ReplicaSet

		for i := range rsList.Items {
			rs := &rsList.Items[i]
			revision := getRevision(rs)
			if revision > latestRevision {
				previousRevision = latestRevision
				previousRS = latestRS
				latestRevision = revision
				latestRS = rs
			} else if revision > previousRevision && revision < latestRevision {
				previousRevision = revision
				previousRS = rs
			}
		}

		if previousRS == nil {
			http.Error(w, "No previous revision found", http.StatusBadRequest)
			return
		}
		targetRS = previousRS
	} else {
		// Find specific revision
		for i := range rsList.Items {
			rs := &rsList.Items[i]
			if getRevision(rs) == req.Revision {
				targetRS = rs
				break
			}
		}
		if targetRS == nil {
			http.Error(w, fmt.Sprintf("Revision %d not found", req.Revision), http.StatusNotFound)
			return
		}
	}

	// Patch deployment with the target template
	patch, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"template": targetRS.Spec.Template,
		},
	})
	if err != nil {
		http.Error(w, "Failed to create patch", http.StatusInternalServerError)
		return
	}

	_, err = clientset.AppsV1().Deployments(req.Namespace).Patch(ctx, req.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to rollback deployment: %v", err), http.StatusInternalServerError)
		return
	}

	targetRevision := getRevision(targetRS)

	// Record audit log
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "rollback",
		Resource: fmt.Sprintf("deployment/%s", req.Name),
		Details:  fmt.Sprintf("Rolled back to revision %d in namespace %s", targetRevision, req.Namespace),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"message":  fmt.Sprintf("Deployment %s rolled back to revision %d", req.Name, targetRevision),
		"revision": targetRevision,
	})
}

// getRevision extracts the revision number from a ReplicaSet
func getRevision(rs *appsv1.ReplicaSet) int64 {
	if rs.Annotations == nil {
		return 0
	}
	revisionStr, ok := rs.Annotations["deployment.kubernetes.io/revision"]
	if !ok {
		return 0
	}
	var revision int64
	fmt.Sscanf(revisionStr, "%d", &revision)
	return revision
}

// handleDeploymentHistory handles GET /api/deployment/history
func (s *Server) handleDeploymentHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	namespace := r.URL.Query().Get("namespace")
	name := r.URL.Query().Get("name")

	if namespace == "" || name == "" {
		http.Error(w, "namespace and name are required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	clientset := s.k8sClient.Clientset

	// Get deployment
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get deployment: %v", err), http.StatusNotFound)
		return
	}

	// Get ReplicaSets
	rsList, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(deployment.Spec.Selector),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list ReplicaSets: %v", err), http.StatusInternalServerError)
		return
	}

	type HistoryEntry struct {
		Revision  int64     `json:"revision"`
		CreatedAt time.Time `json:"createdAt"`
		Replicas  int32     `json:"replicas"`
		Image     string    `json:"image"`
		Current   bool      `json:"current"`
	}

	history := make([]HistoryEntry, 0)
	currentRevision := getRevision(&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Annotations: deployment.Annotations}})

	for _, rs := range rsList.Items {
		revision := getRevision(&rs)
		image := ""
		if len(rs.Spec.Template.Spec.Containers) > 0 {
			image = rs.Spec.Template.Spec.Containers[0].Image
		}
		replicas := int32(0)
		if rs.Spec.Replicas != nil {
			replicas = *rs.Spec.Replicas
		}

		history = append(history, HistoryEntry{
			Revision:  revision,
			CreatedAt: rs.CreationTimestamp.Time,
			Replicas:  replicas,
			Image:     image,
			Current:   revision == currentRevision,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"deployment": name,
		"namespace":  namespace,
		"history":    history,
	})
}

// ==========================================
// CronJob Operations
// ==========================================

// handleCronJobTrigger handles POST /api/cronjob/trigger
func (s *Server) handleCronJobTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Namespace == "" || req.Name == "" {
		http.Error(w, "namespace and name are required", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	ctx := r.Context()
	clientset := s.k8sClient.Clientset

	// Get the CronJob
	cronJob, err := clientset.BatchV1().CronJobs(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get CronJob: %v", err), http.StatusNotFound)
		return
	}

	// Create a Job from the CronJob template
	jobName := fmt.Sprintf("%s-manual-%d", req.Name, time.Now().Unix())
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: req.Namespace,
			Labels: map[string]string{
				"job-name":                             jobName,
				"cronjob.kubernetes.io/manual-trigger": "true",
			},
			Annotations: map[string]string{
				"cronjob.kubernetes.io/instantiate": "manual",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "batch/v1",
					Kind:       "CronJob",
					Name:       cronJob.Name,
					UID:        cronJob.UID,
				},
			},
		},
		Spec: cronJob.Spec.JobTemplate.Spec,
	}

	createdJob, err := clientset.BatchV1().Jobs(req.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create Job: %v", err), http.StatusInternalServerError)
		return
	}

	// Record audit log
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "trigger",
		Resource: fmt.Sprintf("cronjob/%s", req.Name),
		Details:  fmt.Sprintf("Manually triggered job %s in namespace %s", jobName, req.Namespace),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("CronJob %s triggered, created Job %s", req.Name, jobName),
		"jobName": createdJob.Name,
	})
}

// handleCronJobSuspend handles POST /api/cronjob/suspend
func (s *Server) handleCronJobSuspend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
		Suspend   bool   `json:"suspend"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Namespace == "" || req.Name == "" {
		http.Error(w, "namespace and name are required", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	ctx := r.Context()
	clientset := s.k8sClient.Clientset

	// Patch CronJob
	patch := fmt.Sprintf(`{"spec":{"suspend":%t}}`, req.Suspend)
	_, err := clientset.BatchV1().CronJobs(req.Namespace).Patch(ctx, req.Name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update CronJob: %v", err), http.StatusInternalServerError)
		return
	}

	action := "suspended"
	if !req.Suspend {
		action = "resumed"
	}

	// Record audit log
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   action,
		Resource: fmt.Sprintf("cronjob/%s", req.Name),
		Details:  fmt.Sprintf("CronJob %s in namespace %s", action, req.Namespace),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("CronJob %s %s", req.Name, action),
	})
}

// ==========================================
// Node Operations
// ==========================================

// handleNodeCordon handles POST /api/node/cordon
func (s *Server) handleNodeCordon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name     string `json:"name"`
		Uncordon bool   `json:"uncordon"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "node name is required", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	ctx := r.Context()
	clientset := s.k8sClient.Clientset

	// Get node
	node, err := clientset.CoreV1().Nodes().Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get node: %v", err), http.StatusNotFound)
		return
	}

	// Update unschedulable field
	node.Spec.Unschedulable = !req.Uncordon
	_, err = clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update node: %v", err), http.StatusInternalServerError)
		return
	}

	action := "cordoned"
	if req.Uncordon {
		action = "uncordoned"
	}

	// Record audit log
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   action,
		Resource: fmt.Sprintf("node/%s", req.Name),
		Details:  fmt.Sprintf("Node %s %s", req.Name, action),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Node %s %s", req.Name, action),
	})
}

// handleNodeDrain handles POST /api/node/drain
func (s *Server) handleNodeDrain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name               string `json:"name"`
		Force              bool   `json:"force"`
		IgnoreDaemonSets   bool   `json:"ignoreDaemonSets"`
		DeleteEmptyDirData bool   `json:"deleteEmptyDirData"`
		GracePeriod        int    `json:"gracePeriod"` // seconds, -1 for default
		Timeout            int    `json:"timeout"`     // seconds
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "node name is required", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	ctx := r.Context()
	clientset := s.k8sClient.Clientset

	// First cordon the node
	node, err := clientset.CoreV1().Nodes().Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get node: %v", err), http.StatusNotFound)
		return
	}

	if !node.Spec.Unschedulable {
		node.Spec.Unschedulable = true
		_, err = clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to cordon node: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Set up drain helper
	drainer := &drain.Helper{
		Ctx:                 ctx,
		Client:              clientset,
		Force:               req.Force,
		IgnoreAllDaemonSets: req.IgnoreDaemonSets,
		DeleteEmptyDirData:  req.DeleteEmptyDirData,
		GracePeriodSeconds:  req.GracePeriod,
		Out:                 &strings.Builder{},
		ErrOut:              &strings.Builder{},
	}

	if req.Timeout > 0 {
		drainer.Timeout = time.Duration(req.Timeout) * time.Second
	}

	// Get pods to evict
	podList, errs := drainer.GetPodsForDeletion(req.Name)
	if len(errs) > 0 {
		errMsgs := make([]string, len(errs))
		for i, e := range errs {
			errMsgs[i] = e.Error()
		}
		http.Error(w, fmt.Sprintf("Failed to get pods: %s", strings.Join(errMsgs, "; ")), http.StatusInternalServerError)
		return
	}

	// Evict pods
	err = drainer.DeleteOrEvictPods(podList.Pods())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to drain node: %v", err), http.StatusInternalServerError)
		return
	}

	// Record audit log
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "drain",
		Resource: fmt.Sprintf("node/%s", req.Name),
		Details:  fmt.Sprintf("Drained node %s, evicted %d pods", req.Name, len(podList.Pods())),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     fmt.Sprintf("Node %s drained successfully", req.Name),
		"evictedPods": len(podList.Pods()),
	})
}

// handleNodePods handles GET /api/node/pods
func (s *Server) handleNodePods(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nodeName := r.URL.Query().Get("name")
	if nodeName == "" {
		http.Error(w, "node name is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	clientset := s.k8sClient.Clientset

	// List pods on the node
	podList, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list pods: %v", err), http.StatusInternalServerError)
		return
	}

	type PodInfo struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Status    string `json:"status"`
		Ready     string `json:"ready"`
		Restarts  int32  `json:"restarts"`
		Age       string `json:"age"`
	}

	pods := make([]PodInfo, 0)
	for _, pod := range podList.Items {
		pods = append(pods, PodInfo{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
			Ready:     getPodReadyCount(&pod),
			Restarts:  getPodRestarts(&pod),
			Age:       formatAge(pod.CreationTimestamp.Time),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"node":  nodeName,
		"pods":  pods,
		"count": len(pods),
	})
}

// ==========================================
// StatefulSet Operations
// ==========================================

// handleStatefulSetScale handles POST /api/statefulset/scale
func (s *Server) handleStatefulSetScale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
		Replicas  int32  `json:"replicas"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Namespace == "" || req.Name == "" {
		http.Error(w, "namespace and name are required", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	ctx := r.Context()
	clientset := s.k8sClient.Clientset

	// Get current StatefulSet
	sts, err := clientset.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get StatefulSet: %v", err), http.StatusNotFound)
		return
	}

	oldReplicas := int32(0)
	if sts.Spec.Replicas != nil {
		oldReplicas = *sts.Spec.Replicas
	}

	// Scale the StatefulSet
	sts.Spec.Replicas = &req.Replicas
	_, err = clientset.AppsV1().StatefulSets(req.Namespace).Update(ctx, sts, metav1.UpdateOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to scale StatefulSet: %v", err), http.StatusInternalServerError)
		return
	}

	// Record audit log
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "scale",
		Resource: fmt.Sprintf("statefulset/%s", req.Name),
		Details:  fmt.Sprintf("Scaled from %d to %d replicas in namespace %s", oldReplicas, req.Replicas, req.Namespace),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     fmt.Sprintf("StatefulSet %s scaled to %d replicas", req.Name, req.Replicas),
		"oldReplicas": oldReplicas,
		"newReplicas": req.Replicas,
	})
}

// handleStatefulSetRestart handles POST /api/statefulset/restart
func (s *Server) handleStatefulSetRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Namespace == "" || req.Name == "" {
		http.Error(w, "namespace and name are required", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	ctx := r.Context()
	clientset := s.k8sClient.Clientset

	// Patch StatefulSet to trigger rollout restart
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err := clientset.AppsV1().StatefulSets(req.Namespace).Patch(ctx, req.Name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to restart StatefulSet: %v", err), http.StatusInternalServerError)
		return
	}

	// Record audit log
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "restart",
		Resource: fmt.Sprintf("statefulset/%s", req.Name),
		Details:  fmt.Sprintf("Triggered rollout restart in namespace %s", req.Namespace),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("StatefulSet %s restart initiated", req.Name),
	})
}

// ==========================================
// DaemonSet Operations
// ==========================================

// handleDaemonSetRestart handles POST /api/daemonset/restart
func (s *Server) handleDaemonSetRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Namespace == "" || req.Name == "" {
		http.Error(w, "namespace and name are required", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	ctx := r.Context()
	clientset := s.k8sClient.Clientset

	// Patch DaemonSet to trigger rollout restart
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err := clientset.AppsV1().DaemonSets(req.Namespace).Patch(ctx, req.Name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to restart DaemonSet: %v", err), http.StatusInternalServerError)
		return
	}

	// Record audit log
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "restart",
		Resource: fmt.Sprintf("daemonset/%s", req.Name),
		Details:  fmt.Sprintf("Triggered rollout restart in namespace %s", req.Namespace),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("DaemonSet %s restart initiated", req.Name),
	})
}

// formatAge formats a time as a human-readable age string.
// For ages >= 24h, shows days and hours (e.g., "21d9h").
// For smaller ages, shows hours/minutes/seconds.
func formatAge(t time.Time) string {
	dur := time.Since(t).Round(time.Second)
	days := int(dur.Hours() / 24)
	hours := int(dur.Hours()) % 24
	minutes := int(dur.Minutes()) % 60
	seconds := int(dur.Seconds()) % 60

	if days >= 365 {
		years := days / 365
		months := (days % 365) / 30
		if months > 0 {
			return fmt.Sprintf("%dy%dM", years, months)
		}
		return fmt.Sprintf("%dy", years)
	}
	if days >= 30 {
		months := days / 30
		remainDays := days % 30
		if remainDays > 0 {
			return fmt.Sprintf("%dM%dd", months, remainDays)
		}
		return fmt.Sprintf("%dM", months)
	}
	if days > 0 {
		if hours > 0 {
			return fmt.Sprintf("%dd%dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	}
	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	if minutes > 0 {
		if seconds > 0 {
			return fmt.Sprintf("%dm%ds", minutes, seconds)
		}
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", seconds)
}
