package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/helm"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/mcp"
)

// setupHelmTestServer creates a test server with a Helm client for handler testing.
// The Helm client is created without a real kubeconfig, so operations that need
// a cluster connection will fail - which lets us test error handling paths.
func setupHelmTestServer(t *testing.T) *Server {
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
	helmClient := helm.NewClient("", "")

	return &Server{
		cfg:              cfg,
		helmClient:       helmClient,
		mcpClient:        mcp.NewClient(),
		authManager:      authManager,
		port:             8080,
		pendingApprovals: make(map[string]*PendingToolApproval),
	}
}

// ==========================================
// handleHelmReleases Tests
// ==========================================

func TestHelmReleases_MethodNotAllowed(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/helm/releases", nil)
	w := httptest.NewRecorder()

	s.handleHelmReleases(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/helm/releases: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHelmReleases_GET(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/releases?namespace=default", nil)
	w := httptest.NewRecorder()

	s.handleHelmReleases(w, req)

	// Without a real cluster, this may either return empty list or error.
	// The handler should not panic regardless.
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/helm/releases: unexpected status = %d", w.Code)
	}

	if w.Code == http.StatusOK {
		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		if _, ok := resp["items"]; !ok {
			t.Error("Response missing 'items' key")
		}
	}
}

func TestHelmReleases_AllNamespaces(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/releases?all=true", nil)
	w := httptest.NewRecorder()

	s.handleHelmReleases(w, req)

	// Should not panic; either success or error from helm SDK
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/helm/releases?all=true: unexpected status = %d", w.Code)
	}
}

// ==========================================
// handleHelmRelease Tests
// ==========================================

func TestHelmRelease_EmptyPath(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/release/", nil)
	w := httptest.NewRecorder()

	s.handleHelmRelease(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("GET /api/helm/release/ (empty name): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Release name required") {
		t.Errorf("Expected 'Release name required' error, got: %s", w.Body.String())
	}
}

func TestHelmRelease_GetByName(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/release/my-release?namespace=default", nil)
	w := httptest.NewRecorder()

	s.handleHelmRelease(w, req)

	// Without a real cluster, this will likely fail with 500
	// but should not panic
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/helm/release/my-release: unexpected status = %d", w.Code)
	}
}

func TestHelmRelease_History(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/release/my-release/history?namespace=default", nil)
	w := httptest.NewRecorder()

	s.handleHelmRelease(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/helm/release/my-release/history: unexpected status = %d", w.Code)
	}
}

func TestHelmRelease_Values(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/release/my-release/values?namespace=default&all=true", nil)
	w := httptest.NewRecorder()

	s.handleHelmRelease(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/helm/release/my-release/values: unexpected status = %d", w.Code)
	}
}

func TestHelmRelease_Manifest(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/release/my-release/manifest?namespace=default", nil)
	w := httptest.NewRecorder()

	s.handleHelmRelease(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/helm/release/my-release/manifest: unexpected status = %d", w.Code)
	}
}

func TestHelmRelease_Notes(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/release/my-release/notes?namespace=default", nil)
	w := httptest.NewRecorder()

	s.handleHelmRelease(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/helm/release/my-release/notes: unexpected status = %d", w.Code)
	}
}

func TestHelmRelease_UnknownAction(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/release/my-release/unknown-action", nil)
	w := httptest.NewRecorder()

	s.handleHelmRelease(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("GET /api/helm/release/my-release/unknown-action: status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHelmRelease_DefaultNamespace(t *testing.T) {
	s := setupHelmTestServer(t)

	// No namespace param - should default to "default"
	req := httptest.NewRequest(http.MethodGet, "/api/helm/release/my-release", nil)
	w := httptest.NewRecorder()

	s.handleHelmRelease(w, req)

	// Should not panic - will either succeed or return error
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/helm/release/my-release (no ns): unexpected status = %d", w.Code)
	}
}

// ==========================================
// handleHelmInstall Tests
// ==========================================

func TestHelmInstall_MethodNotAllowed(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/install", nil)
	w := httptest.NewRecorder()

	s.handleHelmInstall(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/helm/install: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHelmInstall_InvalidBody(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/helm/install", strings.NewReader("invalid-json"))
	w := httptest.NewRecorder()

	s.handleHelmInstall(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/helm/install (bad body): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHelmInstall_MissingRequired(t *testing.T) {
	s := setupHelmTestServer(t)

	body := `{"name":"","chart":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/helm/install", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleHelmInstall(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/helm/install (empty fields): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Name and chart are required") {
		t.Errorf("Expected 'Name and chart are required' error, got: %s", w.Body.String())
	}
}

func TestHelmInstall_MissingChart(t *testing.T) {
	s := setupHelmTestServer(t)

	body := `{"name":"my-release","chart":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/helm/install", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleHelmInstall(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/helm/install (no chart): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ==========================================
// handleHelmUpgrade Tests
// ==========================================

func TestHelmUpgrade_MethodNotAllowed(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/upgrade", nil)
	w := httptest.NewRecorder()

	s.handleHelmUpgrade(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/helm/upgrade: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHelmUpgrade_InvalidBody(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/helm/upgrade", strings.NewReader("not-json"))
	w := httptest.NewRecorder()

	s.handleHelmUpgrade(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/helm/upgrade (bad body): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHelmUpgrade_MissingRequired(t *testing.T) {
	s := setupHelmTestServer(t)

	body := `{"name":"","chart":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/helm/upgrade", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleHelmUpgrade(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/helm/upgrade (empty): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ==========================================
// handleHelmUninstall Tests
// ==========================================

func TestHelmUninstall_MethodNotAllowed(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/uninstall", nil)
	w := httptest.NewRecorder()

	s.handleHelmUninstall(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/helm/uninstall: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHelmUninstall_InvalidBody(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/helm/uninstall", strings.NewReader("bad"))
	w := httptest.NewRecorder()

	s.handleHelmUninstall(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/helm/uninstall (bad body): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHelmUninstall_MissingName(t *testing.T) {
	s := setupHelmTestServer(t)

	body := `{"name":"","namespace":"default"}`
	req := httptest.NewRequest(http.MethodPost, "/api/helm/uninstall", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleHelmUninstall(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/helm/uninstall (no name): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHelmUninstall_AllowsDeleteMethod(t *testing.T) {
	s := setupHelmTestServer(t)

	body := `{"name":"my-release","namespace":"default"}`
	req := httptest.NewRequest(http.MethodDelete, "/api/helm/uninstall", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleHelmUninstall(w, req)

	// DELETE is allowed (along with POST), but will fail because no real cluster
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("DELETE /api/helm/uninstall: unexpected status = %d", w.Code)
	}
}

// ==========================================
// handleHelmRollback Tests
// ==========================================

func TestHelmRollback_MethodNotAllowed(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/rollback", nil)
	w := httptest.NewRecorder()

	s.handleHelmRollback(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/helm/rollback: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHelmRollback_InvalidBody(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/helm/rollback", strings.NewReader("{invalid"))
	w := httptest.NewRecorder()

	s.handleHelmRollback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/helm/rollback (bad body): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHelmRollback_MissingRequired(t *testing.T) {
	s := setupHelmTestServer(t)

	body := `{"name":"","revision":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/helm/rollback", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleHelmRollback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/helm/rollback (empty): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Name and revision are required") {
		t.Errorf("Expected 'Name and revision are required', got: %s", w.Body.String())
	}
}

// ==========================================
// handleHelmRepos Tests
// ==========================================

func TestHelmRepos_GET(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/repos", nil)
	w := httptest.NewRecorder()

	s.handleHelmRepos(w, req)

	// ListRepositories reads the repo file; if it doesn't exist, returns empty list
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/helm/repos: unexpected status = %d", w.Code)
	}

	if w.Code == http.StatusOK {
		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		if _, ok := resp["items"]; !ok {
			t.Error("Response missing 'items' key")
		}
	}
}

func TestHelmRepos_POST_InvalidBody(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/helm/repos", strings.NewReader("bad"))
	w := httptest.NewRecorder()

	s.handleHelmRepos(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/helm/repos (bad body): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHelmRepos_POST_MissingFields(t *testing.T) {
	s := setupHelmTestServer(t)

	body := `{"name":"","url":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/helm/repos", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleHelmRepos(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/helm/repos (empty fields): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHelmRepos_DELETE_MissingName(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/helm/repos", nil)
	w := httptest.NewRecorder()

	s.handleHelmRepos(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("DELETE /api/helm/repos (no name): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Repository name required") {
		t.Errorf("Expected 'Repository name required', got: %s", w.Body.String())
	}
}

func TestHelmRepos_UnsupportedMethod(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/helm/repos", nil)
	w := httptest.NewRecorder()

	s.handleHelmRepos(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("PATCH /api/helm/repos: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ==========================================
// handleHelmSearch Tests
// ==========================================

func TestHelmSearch_MethodNotAllowed(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/helm/search", nil)
	w := httptest.NewRecorder()

	s.handleHelmSearch(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/helm/search: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHelmSearch_MissingQuery(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/search", nil)
	w := httptest.NewRecorder()

	s.handleHelmSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("GET /api/helm/search (no q): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Search keyword required") {
		t.Errorf("Expected 'Search keyword required', got: %s", w.Body.String())
	}
}

func TestHelmSearch_WithQuery(t *testing.T) {
	s := setupHelmTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/helm/search?q=nginx", nil)
	w := httptest.NewRecorder()

	s.handleHelmSearch(w, req)

	// SearchCharts tries to load repo index file; may fail if no repos configured
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/helm/search?q=nginx: unexpected status = %d", w.Code)
	}

	if w.Code == http.StatusOK {
		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		if _, ok := resp["items"]; !ok {
			t.Error("Response missing 'items' key")
		}
	}
}
