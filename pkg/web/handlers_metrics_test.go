package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/mcp"

	corev1 "k8s.io/api/core/v1"
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

	fakeClientset := fake.NewSimpleClientset(
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

func TestPodMetrics_NilMetricsClient(t *testing.T) {
	s := setupMetricsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/pods?namespace=default", nil)
	w := httptest.NewRecorder()

	s.handlePodMetrics(w, req)

	// With nil Metrics client, GetPodMetrics returns an error.
	// The handler encodes a JSON response with error field (not HTTP error).
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/metrics/pods (nil metrics): status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Expected 'error' field in response when metrics-server not available")
	}

	items, ok := resp["items"].([]interface{})
	if !ok {
		t.Error("Expected 'items' field to be an array")
	} else if len(items) != 0 {
		t.Errorf("Expected empty items array, got %d items", len(items))
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

func TestNodeMetrics_NilMetricsClient(t *testing.T) {
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

	if _, ok := resp["error"]; !ok {
		t.Error("Expected 'error' field when metrics-server not available")
	}

	items, ok := resp["items"].([]interface{})
	if !ok {
		t.Error("Expected 'items' field to be an array")
	} else if len(items) != 0 {
		t.Errorf("Expected empty items array, got %d items", len(items))
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
