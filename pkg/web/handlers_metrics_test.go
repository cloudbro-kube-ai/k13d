package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	"github.com/cloudbro-kube-ai/k13d/pkg/mcp"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// setupMetricsTestServer creates a test server with a fake K8s client for metrics testing.
// The Metrics field is nil (simulating metrics-server not available).
func setupMetricsTestServer(t *testing.T) *Server {
	t.Helper()

	cfg := &config.Config{
		Language:    "en",
		EnableAudit: false,
		LLM: config.LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
	}

	authConfig := &AuthConfig{
		Enabled:         false,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "admin123",
		Quiet:           true,
	}
	authManager := NewAuthManager(authConfig)

	fakeClientset := fake.NewClientset( //nolint:staticcheck
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "default"},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
		},
	)

	k8sClient := &k8s.Client{
		Clientset: fakeClientset,
		// Metrics is nil - simulates metrics-server not installed
	}

	return &Server{
		cfg:              cfg,
		k8sClient:        k8sClient,
		mcpClient:        mcp.NewClient(),
		authManager:      authManager,
		port:             8080,
		pendingApprovals: make(map[string]*PendingToolApproval),
		// metricsCollector is nil - simulates collector not initialized
	}
}

// ==========================================
// handlePodMetrics Tests
// ==========================================

func TestPodMetrics_MethodNotAllowed(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/metrics/pods", nil)
	w := httptest.NewRecorder()

	s.handlePodMetrics(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/metrics/pods: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPodMetrics_NilMetricsClient_FallsBackToRequests(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/pods?namespace=default", nil)
	w := httptest.NewRecorder()

	s.handlePodMetrics(w, req)

	// With nil Metrics client, GetPodMetrics fails but handler falls back
	// to GetPodMetricsFromRequests which succeeds via the fake clientset.
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/metrics/pods (nil metrics): status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Fallback succeeds, so no error field expected
	if _, ok := resp["error"]; ok {
		t.Error("Did not expect 'error' field - fallback to pod requests should succeed")
	}

	items, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatal("Expected 'items' field to be an array")
	}
	// The test pod (Running) should appear with 0 CPU/Mem (no resource requests set)
	if len(items) != 1 {
		t.Errorf("Expected 1 item (fallback from pod requests), got %d", len(items))
	}
}

func TestPodMetrics_ContentType(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/pods?namespace=default", nil)
	w := httptest.NewRecorder()

	s.handlePodMetrics(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", ct)
	}
}

// ==========================================
// handleNodeMetrics Tests
// ==========================================

func TestNodeMetrics_MethodNotAllowed(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/metrics/nodes", nil)
	w := httptest.NewRecorder()

	s.handleNodeMetrics(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/metrics/nodes: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestNodeMetrics_NilMetricsClient_FallsBackToRequests(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/nodes", nil)
	w := httptest.NewRecorder()

	s.handleNodeMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /api/metrics/nodes (nil metrics): status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Fallback succeeds (but test pod has no NodeName, so no node metrics)
	if _, ok := resp["error"]; ok {
		t.Error("Did not expect 'error' field - fallback to pod requests should succeed")
	}

	// Test pod has no Spec.NodeName, so no node data from fallback.
	// items may be null (nil slice) or empty array.
	if items := resp["items"]; items != nil {
		arr, ok := items.([]interface{})
		if !ok {
			t.Fatalf("Expected 'items' to be an array, got %T", items)
		}
		if len(arr) != 0 {
			t.Errorf("Expected 0 items (test pod has no NodeName), got %d", len(arr))
		}
	}
}

// ==========================================
// handleMetricsCollectNow Tests
// ==========================================

func TestMetricsCollectNow_MethodNotAllowed(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/collect", nil)
	w := httptest.NewRecorder()

	s.handleMetricsCollectNow(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/metrics/collect: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestMetricsCollectNow_NilCollector(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/metrics/collect", nil)
	w := httptest.NewRecorder()

	s.handleMetricsCollectNow(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("POST /api/metrics/collect (nil collector): status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if success, ok := resp["success"].(bool); !ok || success {
		t.Error("Expected success=false when collector not initialized")
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Expected 'error' field when collector not initialized")
	}
}

// ==========================================
// handleMetricsSummary Tests
// ==========================================

func TestMetricsSummary_MethodNotAllowed(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/metrics/history/summary", nil)
	w := httptest.NewRecorder()

	s.handleMetricsSummary(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/metrics/history/summary: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestMetricsSummary_NilCollector(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/history/summary", nil)
	w := httptest.NewRecorder()

	s.handleMetricsSummary(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /api/metrics/history/summary (nil collector): status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Expected 'error' field when collector not initialized")
	}

	if enabled, ok := resp["enabled"].(bool); !ok || enabled {
		t.Error("Expected enabled=false when collector not initialized")
	}
}

// ==========================================
// handleClusterMetricsHistory Tests
// ==========================================

func TestClusterMetricsHistory_MethodNotAllowed(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/metrics/history/cluster", nil)
	w := httptest.NewRecorder()

	s.handleClusterMetricsHistory(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/metrics/history/cluster: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestClusterMetricsHistory_NilCollector(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/history/cluster?minutes=10", nil)
	w := httptest.NewRecorder()

	s.handleClusterMetricsHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /api/metrics/history/cluster (nil collector): status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Expected 'error' field when collector not initialized")
	}
}

// ==========================================
// handleNodeMetricsHistory Tests
// ==========================================

func TestNodeMetricsHistory_MethodNotAllowed(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/metrics/history/nodes", nil)
	w := httptest.NewRecorder()

	s.handleNodeMetricsHistory(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/metrics/history/nodes: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestNodeMetricsHistory_NilCollector(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/history/nodes?node=node-1", nil)
	w := httptest.NewRecorder()

	s.handleNodeMetricsHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /api/metrics/history/nodes (nil collector): status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Expected 'error' field when collector not initialized")
	}
}

func TestNodeMetricsHistory_MissingNodeParam(t *testing.T) {
	s := setupMetricsTestServer(t)
	// metricsCollector must not be nil for the param check to be reached,
	// but since it IS nil, the nil collector check fires first and returns error JSON.
	// This test verifies the nil-collector path handles missing params gracefully.
	req := httptest.NewRequest(http.MethodGet, "/api/metrics/history/nodes", nil)
	w := httptest.NewRecorder()

	s.handleNodeMetricsHistory(w, req)

	// With nil collector, returns 200 with error JSON before reaching param check
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/metrics/history/nodes (no node, nil collector): status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ==========================================
// handlePodMetricsHistory Tests
// ==========================================

func TestPodMetricsHistory_MethodNotAllowed(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/metrics/history/pods", nil)
	w := httptest.NewRecorder()

	s.handlePodMetricsHistory(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/metrics/history/pods: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPodMetricsHistory_NilCollector(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/history/pods?pod=test-pod&namespace=default", nil)
	w := httptest.NewRecorder()

	s.handlePodMetricsHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /api/metrics/history/pods (nil collector): status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Expected 'error' field when collector not initialized")
	}
}

// ==========================================
// handleAggregatedMetrics Tests
// ==========================================

func TestAggregatedMetrics_MethodNotAllowed(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/metrics/history/aggregated", nil)
	w := httptest.NewRecorder()

	s.handleAggregatedMetrics(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/metrics/history/aggregated: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestAggregatedMetrics_NilCollector(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/history/aggregated?hours=1", nil)
	w := httptest.NewRecorder()

	s.handleAggregatedMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /api/metrics/history/aggregated (nil collector): status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Expected 'error' field when collector not initialized")
	}
}

// ==========================================
// Metrics Fallback with Resource Requests
// ==========================================

// setupMetricsTestServerWithResources creates a server with pods that have resource requests set.
func setupMetricsTestServerWithResources(t *testing.T) *Server {
	t.Helper()

	cfg := &config.Config{
		Language:    "en",
		EnableAudit: false,
		LLM: config.LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
	}

	authConfig := &AuthConfig{
		Enabled:         false,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "admin123",
		Quiet:           true,
	}
	authManager := NewAuthManager(authConfig)

	fakeClientset := fake.NewClientset( //nolint:staticcheck
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "web-app", Namespace: "default"},
			Spec: corev1.PodSpec{
				NodeName: "node-1",
				Containers: []corev1.Container{
					{
						Name:  "web",
						Image: "nginx:latest",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("250m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "api-server", Namespace: "default"},
			Spec: corev1.PodSpec{
				NodeName: "node-1",
				Containers: []corev1.Container{
					{
						Name:  "api",
						Image: "api:latest",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("256Mi"),
							},
						},
					},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
		},
	)

	k8sClient := &k8s.Client{
		Clientset: fakeClientset,
		// Metrics is nil - triggers fallback to pod resource requests
	}

	return &Server{
		cfg:              cfg,
		k8sClient:        k8sClient,
		mcpClient:        mcp.NewClient(),
		authManager:      authManager,
		port:             8080,
		pendingApprovals: make(map[string]*PendingToolApproval),
	}
}

func TestPodMetrics_FallbackWithResourceRequests(t *testing.T) {
	s := setupMetricsTestServerWithResources(t)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/pods?namespace=default", nil)
	w := httptest.NewRecorder()

	s.handlePodMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := resp["error"]; ok {
		t.Error("Did not expect error - fallback should succeed")
	}

	items, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatal("Expected 'items' field to be an array")
	}
	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	// Verify CPU/Memory values from resource requests
	for _, item := range items {
		m := item.(map[string]interface{})
		name := m["name"].(string)
		cpu := m["cpu"].(float64)
		mem := m["memory"].(float64)

		switch name {
		case "web-app":
			if cpu != 250 {
				t.Errorf("web-app CPU = %v, want 250", cpu)
			}
			if mem != 128 {
				t.Errorf("web-app Memory = %v, want 128", mem)
			}
		case "api-server":
			if cpu != 500 {
				t.Errorf("api-server CPU = %v, want 500", cpu)
			}
			if mem != 256 {
				t.Errorf("api-server Memory = %v, want 256", mem)
			}
		default:
			t.Errorf("Unexpected pod: %s", name)
		}
	}
}

func TestNodeMetrics_FallbackWithResourceRequests(t *testing.T) {
	s := setupMetricsTestServerWithResources(t)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/nodes", nil)
	w := httptest.NewRecorder()

	s.handleNodeMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := resp["error"]; ok {
		t.Error("Did not expect error - fallback should succeed")
	}

	items, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatal("Expected 'items' field to be an array")
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(items))
	}

	node := items[0].(map[string]interface{})
	if node["name"] != "node-1" {
		t.Errorf("node name = %v, want node-1", node["name"])
	}
	// node-1 should aggregate both pods: 250+500=750 CPU, 128+256=384 Mem
	if cpu := node["cpu"].(float64); cpu != 750 {
		t.Errorf("node-1 CPU = %v, want 750", cpu)
	}
	if mem := node["memory"].(float64); mem != 384 {
		t.Errorf("node-1 Memory = %v, want 384", mem)
	}
}
