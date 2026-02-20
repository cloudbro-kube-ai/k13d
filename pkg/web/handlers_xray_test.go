package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/db"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// setupXRayTestServer creates a test server with resources that have owner references.
func setupXRayTestServer(t *testing.T) *Server {
	t.Helper()

	replicas := int32(2)
	completions := int32(1)

	fakeClientset := fake.NewSimpleClientset( //nolint:staticcheck
		// Deployment
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "web-app",
				Namespace: "default",
				UID:       "deploy-uid-1",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "web"},
				},
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     2,
				AvailableReplicas: 2,
			},
		},

		// ReplicaSet owned by the deployment
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "web-app-abc123",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{Name: "web-app", Kind: "Deployment"},
				},
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "web"},
				},
			},
			Status: appsv1.ReplicaSetStatus{
				Replicas:      2,
				ReadyReplicas: 2,
			},
		},

		// Pod owned by the replicaset
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "web-app-abc123-pod1",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{Name: "web-app-abc123", Kind: "ReplicaSet"},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},

		// StatefulSet
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "redis",
				Namespace: "default",
				UID:       "sts-uid-1",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "redis"},
				},
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas: 2,
			},
		},

		// Pod owned by statefulset
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "redis-0",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{Name: "redis", Kind: "StatefulSet"},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},

		// DaemonSet
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fluentd",
				Namespace: "default",
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 2,
				NumberReady:            2,
			},
		},

		// Pod owned by daemonset
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fluentd-node1",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{Name: "fluentd", Kind: "DaemonSet"},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},

		// Job
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "migrate-db",
				Namespace: "default",
			},
			Spec: batchv1.JobSpec{
				Completions: &completions,
			},
			Status: batchv1.JobStatus{
				Succeeded: 1,
			},
		},

		// Pod owned by job
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "migrate-db-pod1",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{Name: "migrate-db", Kind: "Job"},
				},
			},
			Status: corev1.PodStatus{Phase: "Succeeded"},
		},

		// CronJob
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "backup",
				Namespace: "default",
			},
			Spec: batchv1.CronJobSpec{
				Schedule: "0 2 * * *",
			},
		},

		// Job owned by cronjob
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "backup-1234",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{Name: "backup", Kind: "CronJob"},
				},
			},
			Spec: batchv1.JobSpec{
				Completions: &completions,
			},
			Status: batchv1.JobStatus{
				Active: 1,
			},
		},
	)

	cfg := &config.Config{Language: "en"}
	authConfig := &AuthConfig{
		Enabled:         false,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "admin123",
		Quiet:           true,
	}

	return &Server{
		cfg:              cfg,
		k8sClient:        &k8s.Client{Clientset: fakeClientset},
		authManager:      NewAuthManager(authConfig),
		pendingApprovals: make(map[string]*PendingToolApproval),
	}
}

func TestHandleXRayDeployments(t *testing.T) {
	dbPath := "test_xray_deploy.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	server := setupXRayTestServer(t)
	handler := http.HandlerFunc(server.handleXRay)

	req := httptest.NewRequest(http.MethodGet, "/api/xray?type=deploy", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp XRayResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Type != "deploy" {
		t.Errorf("Expected type 'deploy', got %q", resp.Type)
	}
	if len(resp.Nodes) != 1 {
		t.Fatalf("Expected 1 deployment node, got %d", len(resp.Nodes))
	}

	dep := resp.Nodes[0]
	if dep.Kind != "Deployment" {
		t.Errorf("Expected kind 'Deployment', got %q", dep.Kind)
	}
	if dep.Name != "web-app" {
		t.Errorf("Expected name 'web-app', got %q", dep.Name)
	}
	if dep.Status != "Ready" {
		t.Errorf("Expected status 'Ready', got %q", dep.Status)
	}

	// Should have 1 ReplicaSet child
	if len(dep.Children) != 1 {
		t.Fatalf("Expected 1 ReplicaSet child, got %d", len(dep.Children))
	}
	rs := dep.Children[0]
	if rs.Kind != "ReplicaSet" {
		t.Errorf("Expected kind 'ReplicaSet', got %q", rs.Kind)
	}

	// ReplicaSet should have 1 Pod child
	if len(rs.Children) != 1 {
		t.Fatalf("Expected 1 Pod child, got %d", len(rs.Children))
	}
	pod := rs.Children[0]
	if pod.Kind != "Pod" {
		t.Errorf("Expected kind 'Pod', got %q", pod.Kind)
	}
	if pod.Status != "Running" {
		t.Errorf("Expected pod status 'Running', got %q", pod.Status)
	}
}

func TestHandleXRayStatefulSets(t *testing.T) {
	dbPath := "test_xray_sts.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	server := setupXRayTestServer(t)
	handler := http.HandlerFunc(server.handleXRay)

	req := httptest.NewRequest(http.MethodGet, "/api/xray?type=sts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp XRayResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Type != "sts" {
		t.Errorf("Expected type 'sts', got %q", resp.Type)
	}
	if len(resp.Nodes) != 1 {
		t.Fatalf("Expected 1 StatefulSet node, got %d", len(resp.Nodes))
	}

	sts := resp.Nodes[0]
	if sts.Kind != "StatefulSet" {
		t.Errorf("Expected kind 'StatefulSet', got %q", sts.Kind)
	}
	if sts.Name != "redis" {
		t.Errorf("Expected name 'redis', got %q", sts.Name)
	}
	if len(sts.Children) != 1 {
		t.Errorf("Expected 1 pod child, got %d", len(sts.Children))
	}
}

func TestHandleXRayDaemonSets(t *testing.T) {
	dbPath := "test_xray_ds.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	server := setupXRayTestServer(t)
	handler := http.HandlerFunc(server.handleXRay)

	req := httptest.NewRequest(http.MethodGet, "/api/xray?type=ds", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp XRayResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Type != "ds" {
		t.Errorf("Expected type 'ds', got %q", resp.Type)
	}
	if len(resp.Nodes) != 1 {
		t.Fatalf("Expected 1 DaemonSet node, got %d", len(resp.Nodes))
	}
	if resp.Nodes[0].Kind != "DaemonSet" {
		t.Errorf("Expected kind 'DaemonSet', got %q", resp.Nodes[0].Kind)
	}
	if len(resp.Nodes[0].Children) != 1 {
		t.Errorf("Expected 1 pod child, got %d", len(resp.Nodes[0].Children))
	}
}

func TestHandleXRayJobs(t *testing.T) {
	dbPath := "test_xray_jobs.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	server := setupXRayTestServer(t)
	handler := http.HandlerFunc(server.handleXRay)

	req := httptest.NewRequest(http.MethodGet, "/api/xray?type=job", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp XRayResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Type != "job" {
		t.Errorf("Expected type 'job', got %q", resp.Type)
	}
	// 2 jobs: migrate-db + backup-1234 (both in default ns)
	if len(resp.Nodes) != 2 {
		t.Fatalf("Expected 2 job nodes, got %d", len(resp.Nodes))
	}

	// Find the migrate-db job and check its child
	for _, node := range resp.Nodes {
		if node.Name == "migrate-db" {
			if node.Status != "Complete" {
				t.Errorf("Expected migrate-db status 'Complete', got %q", node.Status)
			}
			if len(node.Children) != 1 {
				t.Errorf("Expected 1 pod child for migrate-db, got %d", len(node.Children))
			}
		}
		if node.Name == "backup-1234" {
			if node.Status != "Active" {
				t.Errorf("Expected backup-1234 status 'Active', got %q", node.Status)
			}
		}
	}
}

func TestHandleXRayCronJobs(t *testing.T) {
	dbPath := "test_xray_cj.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	server := setupXRayTestServer(t)
	handler := http.HandlerFunc(server.handleXRay)

	req := httptest.NewRequest(http.MethodGet, "/api/xray?type=cj", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp XRayResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Type != "cj" {
		t.Errorf("Expected type 'cj', got %q", resp.Type)
	}
	if len(resp.Nodes) != 1 {
		t.Fatalf("Expected 1 CronJob node, got %d", len(resp.Nodes))
	}

	cj := resp.Nodes[0]
	if cj.Kind != "CronJob" {
		t.Errorf("Expected kind 'CronJob', got %q", cj.Kind)
	}
	if cj.Name != "backup" {
		t.Errorf("Expected name 'backup', got %q", cj.Name)
	}
	if cj.Status != "0 2 * * *" {
		t.Errorf("Expected schedule '0 2 * * *', got %q", cj.Status)
	}

	// Should have 1 job child (backup-1234)
	if len(cj.Children) != 1 {
		t.Fatalf("Expected 1 job child, got %d", len(cj.Children))
	}
	if cj.Children[0].Name != "backup-1234" {
		t.Errorf("Expected child job name 'backup-1234', got %q", cj.Children[0].Name)
	}
}

func TestHandleXRayDefaultType(t *testing.T) {
	dbPath := "test_xray_default.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	server := setupXRayTestServer(t)
	handler := http.HandlerFunc(server.handleXRay)

	// No type parameter should default to deploy
	req := httptest.NewRequest(http.MethodGet, "/api/xray", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp XRayResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Type != "deploy" {
		t.Errorf("Expected default type 'deploy', got %q", resp.Type)
	}
}

func TestHandleXRayNamespaceFilter(t *testing.T) {
	dbPath := "test_xray_ns.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	server := setupXRayTestServer(t)
	handler := http.HandlerFunc(server.handleXRay)

	req := httptest.NewRequest(http.MethodGet, "/api/xray?type=deploy&namespace=nonexistent", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp XRayResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Namespace != "nonexistent" {
		t.Errorf("Expected namespace 'nonexistent', got %q", resp.Namespace)
	}
	if len(resp.Nodes) != 0 {
		t.Errorf("Expected 0 nodes for nonexistent namespace, got %d", len(resp.Nodes))
	}
}

func TestHandleXRayNilK8sClient(t *testing.T) {
	server := &Server{k8sClient: nil}
	handler := http.HandlerFunc(server.handleXRay)

	req := httptest.NewRequest(http.MethodGet, "/api/xray", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}
	if body["code"] != ErrCodeK8sError {
		t.Errorf("Expected K8S_ERROR code, got %v", body["code"])
	}
}

func TestNormalizeXRayType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"deploy", "deploy"},
		{"deployment", "deploy"},
		{"deployments", "deploy"},
		{"sts", "sts"},
		{"statefulset", "sts"},
		{"statefulsets", "sts"},
		{"job", "job"},
		{"jobs", "job"},
		{"cj", "cj"},
		{"cronjob", "cj"},
		{"cronjobs", "cj"},
		{"ds", "ds"},
		{"daemonset", "ds"},
		{"daemonsets", "ds"},
		{"unknown", "deploy"}, // defaults to deploy
		{"", "deploy"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeXRayType(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeXRayType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
