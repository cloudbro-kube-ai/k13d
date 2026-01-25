package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// ============================================================================
// Test Helpers for Operations E2E Tests
// ============================================================================

// setupOperationsTestServer creates a test server with deployment/statefulset/node resources
func setupOperationsTestServer(t *testing.T) (*Server, *AuthManager) {
	t.Helper()

	cfg := &config.Config{
		Language:     "en",
		BeginnerMode: false,
		EnableAudit:  true,
		LogLevel:     "debug",
	}

	authConfig := &AuthConfig{
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "admin123",
		Quiet:           true,
	}
	authManager := NewAuthManager(authConfig)

	replicas := int32(3)
	fakeClientset := fake.NewSimpleClientset(
		// Namespace
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "production"}},

		// Deployment
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-deployment",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "nginx"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": "nginx"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "nginx", Image: "nginx:1.19"},
						},
					},
				},
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          3,
				ReadyReplicas:     3,
				AvailableReplicas: 3,
			},
		},

		// StatefulSet
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mysql-statefulset",
				Namespace: "default",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas:    &replicas,
				ServiceName: "mysql",
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "mysql"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": "mysql"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "mysql", Image: "mysql:8.0"},
						},
					},
				},
			},
			Status: appsv1.StatefulSetStatus{
				Replicas:      3,
				ReadyReplicas: 3,
			},
		},

		// DaemonSet
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fluentd-daemonset",
				Namespace: "default",
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "fluentd"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": "fluentd"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "fluentd", Image: "fluentd:v1.14"},
						},
					},
				},
			},
			Status: appsv1.DaemonSetStatus{
				NumberReady:            3,
				DesiredNumberScheduled: 3,
			},
		},

		// CronJob
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "backup-cronjob",
				Namespace: "default",
			},
			Spec: batchv1.CronJobSpec{
				Schedule: "0 2 * * *",
				Suspend:  boolPtr(false),
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{Name: "backup", Image: "backup:latest"},
								},
								RestartPolicy: corev1.RestartPolicyOnFailure,
							},
						},
					},
				},
			},
		},

		// Nodes
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "worker-node-1",
				Labels: map[string]string{
					"node-role.kubernetes.io/worker": "",
				},
			},
			Spec: corev1.NodeSpec{
				Unschedulable: false,
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "worker-node-2",
				Labels: map[string]string{
					"node-role.kubernetes.io/worker": "",
				},
			},
			Spec: corev1.NodeSpec{
				Unschedulable: false,
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},

		// Pods on nodes
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-pod-1",
				Namespace: "default",
				Labels:    map[string]string{"app": "nginx"},
			},
			Spec: corev1.PodSpec{
				NodeName: "worker-node-1",
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-pod-2",
				Namespace: "default",
				Labels:    map[string]string{"app": "nginx"},
			},
			Spec: corev1.PodSpec{
				NodeName: "worker-node-2",
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
	)

	k8sClient := &k8s.Client{
		Clientset: fakeClientset,
	}

	server := &Server{
		cfg:         cfg,
		aiClient:    nil,
		k8sClient:   k8sClient,
		authManager: authManager,
		port:        8080,
	}

	return server, authManager
}

func boolPtr(b bool) *bool {
	return &b
}

// ============================================================================
// Deployment Operations E2E Tests
// ============================================================================

// TestE2E_DeploymentScale tests scaling a deployment
func TestE2E_DeploymentScale(t *testing.T) {
	server, authManager := setupOperationsTestServer(t)
	session, _ := authManager.Authenticate("admin", "admin123")

	tests := []struct {
		name           string
		request        DeploymentScaleRequest
		expectedStatus int
		checkResult    bool
	}{
		{
			name: "scale up deployment",
			request: DeploymentScaleRequest{
				Namespace: "default",
				Name:      "nginx-deployment",
				Replicas:  5,
			},
			expectedStatus: http.StatusOK,
			checkResult:    true,
		},
		{
			name: "scale down deployment",
			request: DeploymentScaleRequest{
				Namespace: "default",
				Name:      "nginx-deployment",
				Replicas:  1,
			},
			expectedStatus: http.StatusOK,
			checkResult:    true,
		},
		{
			name: "scale to zero",
			request: DeploymentScaleRequest{
				Namespace: "default",
				Name:      "nginx-deployment",
				Replicas:  0,
			},
			expectedStatus: http.StatusOK,
			checkResult:    true,
		},
		{
			name: "non-existent deployment",
			request: DeploymentScaleRequest{
				Namespace: "default",
				Name:      "non-existent",
				Replicas:  3,
			},
			expectedStatus: http.StatusNotFound,
			checkResult:    false,
		},
		{
			name: "missing namespace",
			request: DeploymentScaleRequest{
				Name:     "nginx-deployment",
				Replicas: 3,
			},
			expectedStatus: http.StatusBadRequest,
			checkResult:    false,
		},
		{
			name: "missing name",
			request: DeploymentScaleRequest{
				Namespace: "default",
				Replicas:  3,
			},
			expectedStatus: http.StatusBadRequest,
			checkResult:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/deployment/scale", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+session.ID)
			w := httptest.NewRecorder()

			authManager.AuthMiddleware(http.HandlerFunc(server.handleDeploymentScale)).ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkResult && w.Code == http.StatusOK {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}

				if resp["success"] != true {
					t.Error("expected success: true")
				}

				newReplicas, ok := resp["newReplicas"].(float64)
				if !ok || int32(newReplicas) != tt.request.Replicas {
					t.Errorf("expected newReplicas %d, got %v", tt.request.Replicas, resp["newReplicas"])
				}
			}
		})
	}
}

// TestE2E_DeploymentRestart tests restarting a deployment
func TestE2E_DeploymentRestart(t *testing.T) {
	server, authManager := setupOperationsTestServer(t)
	session, _ := authManager.Authenticate("admin", "admin123")

	tests := []struct {
		name           string
		reqBody        map[string]string
		expectedStatus int
	}{
		{
			name: "restart deployment",
			reqBody: map[string]string{
				"namespace": "default",
				"name":      "nginx-deployment",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "restart non-existent deployment",
			reqBody: map[string]string{
				"namespace": "default",
				"name":      "non-existent",
			},
			expectedStatus: http.StatusInternalServerError, // Server returns 500 for not found in restart
		},
		{
			name: "missing fields",
			reqBody: map[string]string{
				"namespace": "default",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/deployment/restart", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+session.ID)
			w := httptest.NewRecorder()

			authManager.AuthMiddleware(http.HandlerFunc(server.handleDeploymentRestart)).ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestE2E_DeploymentMethodNotAllowed tests method validation
func TestE2E_DeploymentMethodNotAllowed(t *testing.T) {
	server, authManager := setupOperationsTestServer(t)
	session, _ := authManager.Authenticate("admin", "admin123")

	endpoints := []string{
		"/api/deployment/scale",
		"/api/deployment/restart",
	}

	for _, endpoint := range endpoints {
		t.Run("GET "+endpoint, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			req.Header.Set("Authorization", "Bearer "+session.ID)
			w := httptest.NewRecorder()

			var handler http.HandlerFunc
			switch endpoint {
			case "/api/deployment/scale":
				handler = server.handleDeploymentScale
			case "/api/deployment/restart":
				handler = server.handleDeploymentRestart
			}

			authManager.AuthMiddleware(handler).ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected 405, got %d", w.Code)
			}
		})
	}
}

// ============================================================================
// StatefulSet Operations E2E Tests
// ============================================================================

// TestE2E_StatefulSetScale tests scaling a statefulset
func TestE2E_StatefulSetScale(t *testing.T) {
	server, authManager := setupOperationsTestServer(t)
	session, _ := authManager.Authenticate("admin", "admin123")

	tests := []struct {
		name           string
		reqBody        map[string]interface{}
		expectedStatus int
	}{
		{
			name: "scale statefulset",
			reqBody: map[string]interface{}{
				"namespace": "default",
				"name":      "mysql-statefulset",
				"replicas":  5,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "non-existent statefulset",
			reqBody: map[string]interface{}{
				"namespace": "default",
				"name":      "non-existent",
				"replicas":  3,
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/statefulset/scale", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+session.ID)
			w := httptest.NewRecorder()

			authManager.AuthMiddleware(http.HandlerFunc(server.handleStatefulSetScale)).ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestE2E_StatefulSetRestart tests restarting a statefulset
func TestE2E_StatefulSetRestart(t *testing.T) {
	server, authManager := setupOperationsTestServer(t)
	session, _ := authManager.Authenticate("admin", "admin123")

	body, _ := json.Marshal(map[string]string{
		"namespace": "default",
		"name":      "mysql-statefulset",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/statefulset/restart", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+session.ID)
	w := httptest.NewRecorder()

	authManager.AuthMiddleware(http.HandlerFunc(server.handleStatefulSetRestart)).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ============================================================================
// DaemonSet Operations E2E Tests
// ============================================================================

// TestE2E_DaemonSetRestart tests restarting a daemonset
func TestE2E_DaemonSetRestart(t *testing.T) {
	server, authManager := setupOperationsTestServer(t)
	session, _ := authManager.Authenticate("admin", "admin123")

	tests := []struct {
		name           string
		reqBody        map[string]string
		expectedStatus int
	}{
		{
			name: "restart daemonset",
			reqBody: map[string]string{
				"namespace": "default",
				"name":      "fluentd-daemonset",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "non-existent daemonset",
			reqBody: map[string]string{
				"namespace": "default",
				"name":      "non-existent",
			},
			expectedStatus: http.StatusInternalServerError, // Server returns 500 for not found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/daemonset/restart", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+session.ID)
			w := httptest.NewRecorder()

			authManager.AuthMiddleware(http.HandlerFunc(server.handleDaemonSetRestart)).ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// ============================================================================
// CronJob Operations E2E Tests
// ============================================================================

// TestE2E_CronJobSuspend tests suspending a cronjob
func TestE2E_CronJobSuspend(t *testing.T) {
	server, authManager := setupOperationsTestServer(t)
	session, _ := authManager.Authenticate("admin", "admin123")

	tests := []struct {
		name           string
		reqBody        map[string]interface{}
		expectedStatus int
	}{
		{
			name: "suspend cronjob",
			reqBody: map[string]interface{}{
				"namespace": "default",
				"name":      "backup-cronjob",
				"suspend":   true,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "resume cronjob",
			reqBody: map[string]interface{}{
				"namespace": "default",
				"name":      "backup-cronjob",
				"suspend":   false,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "non-existent cronjob",
			reqBody: map[string]interface{}{
				"namespace": "default",
				"name":      "non-existent",
				"suspend":   true,
			},
			expectedStatus: http.StatusInternalServerError, // Server returns 500 for not found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/cronjob/suspend", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+session.ID)
			w := httptest.NewRecorder()

			authManager.AuthMiddleware(http.HandlerFunc(server.handleCronJobSuspend)).ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// ============================================================================
// Node Operations E2E Tests
// ============================================================================

// TestE2E_NodeCordon tests cordoning a node
func TestE2E_NodeCordon(t *testing.T) {
	server, authManager := setupOperationsTestServer(t)
	session, _ := authManager.Authenticate("admin", "admin123")

	tests := []struct {
		name           string
		reqBody        map[string]interface{}
		expectedStatus int
	}{
		{
			name: "cordon node",
			reqBody: map[string]interface{}{
				"name":   "worker-node-1",
				"cordon": true,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "uncordon node",
			reqBody: map[string]interface{}{
				"name":   "worker-node-1",
				"cordon": false,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "non-existent node",
			reqBody: map[string]interface{}{
				"name":   "non-existent-node",
				"cordon": true,
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/node/cordon", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+session.ID)
			w := httptest.NewRecorder()

			authManager.AuthMiddleware(http.HandlerFunc(server.handleNodeCordon)).ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestE2E_NodePods tests getting pods on a node
func TestE2E_NodePods(t *testing.T) {
	server, authManager := setupOperationsTestServer(t)
	session, _ := authManager.Authenticate("admin", "admin123")

	tests := []struct {
		name           string
		nodeName       string
		expectedStatus int
		minPods        int // Minimum expected pods (may include system pods)
	}{
		{
			name:           "get pods on worker-node-1",
			nodeName:       "worker-node-1",
			expectedStatus: http.StatusOK,
			minPods:        1,
		},
		{
			name:           "get pods on worker-node-2",
			nodeName:       "worker-node-2",
			expectedStatus: http.StatusOK,
			minPods:        1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/node/pods?name="+tt.nodeName, nil)
			req.Header.Set("Authorization", "Bearer "+session.ID)
			w := httptest.NewRecorder()

			authManager.AuthMiddleware(http.HandlerFunc(server.handleNodePods)).ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if w.Code == http.StatusOK {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}

				pods, ok := resp["pods"].([]interface{})
				if !ok {
					t.Fatal("expected pods array in response")
				}

				if len(pods) < tt.minPods {
					t.Errorf("expected at least %d pods, got %d", tt.minPods, len(pods))
				}
			}
		})
	}
}

// ============================================================================
// Unauthorized Access Tests
// ============================================================================

// TestE2E_OperationsUnauthorized tests that operations require authentication
func TestE2E_OperationsUnauthorized(t *testing.T) {
	server, authManager := setupOperationsTestServer(t)

	endpoints := []struct {
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{http.MethodPost, "/api/deployment/scale", server.handleDeploymentScale},
		{http.MethodPost, "/api/deployment/restart", server.handleDeploymentRestart},
		{http.MethodPost, "/api/statefulset/scale", server.handleStatefulSetScale},
		{http.MethodPost, "/api/statefulset/restart", server.handleStatefulSetRestart},
		{http.MethodPost, "/api/daemonset/restart", server.handleDaemonSetRestart},
		{http.MethodPost, "/api/cronjob/suspend", server.handleCronJobSuspend},
		{http.MethodPost, "/api/node/cordon", server.handleNodeCordon},
		{http.MethodGet, "/api/node/pods", server.handleNodePods},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			var body *bytes.Buffer
			if ep.method == http.MethodPost {
				body = bytes.NewBufferString(`{}`)
			} else {
				body = nil
			}

			var req *http.Request
			if body != nil {
				req = httptest.NewRequest(ep.method, ep.path, body)
			} else {
				req = httptest.NewRequest(ep.method, ep.path, nil)
			}
			w := httptest.NewRecorder()

			// Call without authentication
			authManager.AuthMiddleware(ep.handler).ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", w.Code)
			}
		})
	}
}

// ============================================================================
// Concurrent Operations Tests
// ============================================================================

// TestE2E_ConcurrentDeploymentScaling tests concurrent scaling operations
func TestE2E_ConcurrentDeploymentScaling(t *testing.T) {
	server, authManager := setupOperationsTestServer(t)
	session, _ := authManager.Authenticate("admin", "admin123")

	// Run multiple concurrent scale operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(replicas int) {
			body, _ := json.Marshal(DeploymentScaleRequest{
				Namespace: "default",
				Name:      "nginx-deployment",
				Replicas:  int32(replicas),
			})
			req := httptest.NewRequest(http.MethodPost, "/api/deployment/scale", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+session.ID)
			w := httptest.NewRecorder()

			authManager.AuthMiddleware(http.HandlerFunc(server.handleDeploymentScale)).ServeHTTP(w, req)

			// All should succeed or fail gracefully (no panics)
			if w.Code != http.StatusOK && w.Code != http.StatusConflict {
				t.Logf("concurrent scale returned %d (may be expected)", w.Code)
			}
			done <- true
		}(i + 1)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// ============================================================================
// Integration Flow Tests
// ============================================================================

// TestE2E_DeploymentLifecycle tests a full deployment lifecycle
func TestE2E_DeploymentLifecycle(t *testing.T) {
	server, authManager := setupOperationsTestServer(t)
	session, _ := authManager.Authenticate("admin", "admin123")

	// Step 1: Scale up
	t.Run("scale up", func(t *testing.T) {
		body, _ := json.Marshal(DeploymentScaleRequest{
			Namespace: "default",
			Name:      "nginx-deployment",
			Replicas:  5,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/deployment/scale", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+session.ID)
		w := httptest.NewRecorder()

		authManager.AuthMiddleware(http.HandlerFunc(server.handleDeploymentScale)).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("scale up failed: %d - %s", w.Code, w.Body.String())
		}
	})

	// Step 2: Restart (rolling update)
	t.Run("restart", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"namespace": "default",
			"name":      "nginx-deployment",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/deployment/restart", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+session.ID)
		w := httptest.NewRecorder()

		authManager.AuthMiddleware(http.HandlerFunc(server.handleDeploymentRestart)).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("restart failed: %d - %s", w.Code, w.Body.String())
		}
	})

	// Step 3: Scale down
	t.Run("scale down", func(t *testing.T) {
		body, _ := json.Marshal(DeploymentScaleRequest{
			Namespace: "default",
			Name:      "nginx-deployment",
			Replicas:  2,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/deployment/scale", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+session.ID)
		w := httptest.NewRecorder()

		authManager.AuthMiddleware(http.HandlerFunc(server.handleDeploymentScale)).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("scale down failed: %d - %s", w.Code, w.Body.String())
		}
	})
}

// TestE2E_NodeMaintenanceFlow tests a node maintenance workflow
func TestE2E_NodeMaintenanceFlow(t *testing.T) {
	server, authManager := setupOperationsTestServer(t)
	session, _ := authManager.Authenticate("admin", "admin123")

	nodeName := "worker-node-1"

	// Step 1: Get pods on node before maintenance
	t.Run("get pods before", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/node/pods?name="+nodeName, nil)
		req.Header.Set("Authorization", "Bearer "+session.ID)
		w := httptest.NewRecorder()

		authManager.AuthMiddleware(http.HandlerFunc(server.handleNodePods)).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("get pods failed: %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		t.Logf("Pods on node before maintenance: %v", resp["pods"])
	})

	// Step 2: Cordon node
	t.Run("cordon node", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"name":   nodeName,
			"cordon": true,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/node/cordon", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+session.ID)
		w := httptest.NewRecorder()

		authManager.AuthMiddleware(http.HandlerFunc(server.handleNodeCordon)).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("cordon failed: %d - %s", w.Code, w.Body.String())
		}
	})

	// Step 3: Uncordon node (restore)
	t.Run("uncordon node", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"name":   nodeName,
			"cordon": false,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/node/cordon", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+session.ID)
		w := httptest.NewRecorder()

		authManager.AuthMiddleware(http.HandlerFunc(server.handleNodeCordon)).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("uncordon failed: %d - %s", w.Code, w.Body.String())
		}
	})
}
