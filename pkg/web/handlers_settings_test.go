package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/db"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	"github.com/cloudbro-kube-ai/k13d/pkg/mcp"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// setupSettingsTestServer creates a test server for settings/models/MCP testing.
func setupSettingsTestServer(t *testing.T) *Server {
	t.Helper()

	cfg := &config.Config{
		Language:     "en",
		BeginnerMode: true,
		EnableAudit:  true,
		LogLevel:     "info",
		ActiveModel:  "gpt-4",
		LLM: config.LLMConfig{
			Provider:        "openai",
			Model:           "gpt-4",
			Endpoint:        "https://api.openai.com/v1",
			ReasoningEffort: "medium",
		},
		Models: []config.ModelProfile{
			{
				Name:        "gpt-4",
				Provider:    "openai",
				Model:       "gpt-4",
				Endpoint:    "https://api.openai.com/v1",
				Description: "OpenAI GPT-4",
				APIKey:      "sk-test-key-123",
			},
			{
				Name:        "claude-3",
				Provider:    "anthropic",
				Model:       "claude-3-opus",
				Endpoint:    "https://api.anthropic.com",
				Description: "Anthropic Claude 3",
			},
		},
		MCP: config.MCPConfig{
			Servers: []config.MCPServer{
				{
					Name:        "kubectl-server",
					Command:     "npx",
					Args:        []string{"@anthropic/mcp-server-kubernetes"},
					Description: "Kubernetes MCP server",
					Enabled:     true,
				},
				{
					Name:        "custom-server",
					Command:     "/usr/local/bin/my-mcp",
					Description: "Custom MCP server",
					Enabled:     false,
				},
			},
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
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
	)

	k8sClient := &k8s.Client{
		Clientset: fakeClientset,
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

// ==========================================
// handleSettings Tests
// ==========================================

func TestSettings_GET(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()

	s.handleSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/settings: status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["language"] != "en" {
		t.Errorf("language = %v, want en", resp["language"])
	}
	if resp["beginner_mode"] != true {
		t.Errorf("beginner_mode = %v, want true", resp["beginner_mode"])
	}
	if resp["enable_audit"] != true {
		t.Errorf("enable_audit = %v, want true", resp["enable_audit"])
	}
	if resp["log_level"] != "info" {
		t.Errorf("log_level = %v, want info", resp["log_level"])
	}

	llm, ok := resp["llm"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'llm' object in response")
	}
	if llm["provider"] != "openai" {
		t.Errorf("llm.provider = %v, want openai", llm["provider"])
	}
	if llm["model"] != "gpt-4" {
		t.Errorf("llm.model = %v, want gpt-4", llm["model"])
	}
}

func TestSettings_MethodNotAllowed(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/settings", nil)
	w := httptest.NewRecorder()

	s.handleSettings(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE /api/settings: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestSettings_PUT_InvalidBody(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader("bad-json"))
	w := httptest.NewRecorder()

	s.handleSettings(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("PUT /api/settings (bad body): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSettings_ContentType(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()

	s.handleSettings(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", ct)
	}
}

// ==========================================
// handleModels Tests
// ==========================================

func TestModels_GET(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	w := httptest.NewRecorder()

	s.handleModels(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/models: status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	models, ok := resp["models"].([]interface{})
	if !ok {
		t.Fatal("Expected 'models' array in response")
	}
	if len(models) != 2 {
		t.Errorf("len(models) = %d, want 2", len(models))
	}

	if resp["active_model"] != "gpt-4" {
		t.Errorf("active_model = %v, want gpt-4", resp["active_model"])
	}

	// Verify API key is masked
	firstModel := models[0].(map[string]interface{})
	if firstModel["has_api_key"] != true {
		t.Error("Expected has_api_key=true for gpt-4 (has API key set)")
	}
	// Ensure raw API key is NOT exposed
	if _, hasKey := firstModel["api_key"]; hasKey {
		t.Error("Raw api_key should NOT be in response")
	}

	// Check is_active flag
	if firstModel["is_active"] != true {
		t.Error("Expected gpt-4 to have is_active=true")
	}

	secondModel := models[1].(map[string]interface{})
	if secondModel["has_api_key"] != false {
		t.Error("Expected has_api_key=false for claude-3 (no API key)")
	}
	if secondModel["is_active"] != false {
		t.Error("Expected claude-3 to have is_active=false")
	}
}

func TestModels_POST_InvalidBody(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/models", strings.NewReader("not-json"))
	w := httptest.NewRecorder()

	s.handleModels(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/models (bad body): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestModels_POST_MissingFields(t *testing.T) {
	s := setupSettingsTestServer(t)

	body := `{"name":"","provider":"","model":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/models", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleModels(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/models (empty fields): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Name, provider, and model are required") {
		t.Errorf("Expected required fields error, got: %s", w.Body.String())
	}
}

func TestModels_DELETE_MissingName(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/models", nil)
	w := httptest.NewRecorder()

	s.handleModels(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("DELETE /api/models (no name): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestModels_DELETE_NotFound(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/models?name=nonexistent", nil)
	w := httptest.NewRecorder()

	s.handleModels(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("DELETE /api/models?name=nonexistent: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestModels_MethodNotAllowed(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/models", nil)
	w := httptest.NewRecorder()

	s.handleModels(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("PATCH /api/models: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ==========================================
// handleActiveModel Tests
// ==========================================

func TestActiveModel_GET(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/models/active", nil)
	w := httptest.NewRecorder()

	s.handleActiveModel(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/models/active: status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["name"] != "gpt-4" {
		t.Errorf("name = %v, want gpt-4", resp["name"])
	}
	if resp["provider"] != "openai" {
		t.Errorf("provider = %v, want openai", resp["provider"])
	}
	if resp["model"] != "gpt-4" {
		t.Errorf("model = %v, want gpt-4", resp["model"])
	}
}

func TestActiveModel_GET_NoActiveModel(t *testing.T) {
	s := setupSettingsTestServer(t)
	// GetActiveModelProfile returns nil only when Models slice is empty
	s.cfg.ActiveModel = "nonexistent"
	s.cfg.Models = nil

	req := httptest.NewRequest(http.MethodGet, "/api/models/active", nil)
	w := httptest.NewRecorder()

	s.handleActiveModel(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GET /api/models/active (none): status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestActiveModel_PUT_InvalidBody(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/models/active", strings.NewReader("bad"))
	w := httptest.NewRecorder()

	s.handleActiveModel(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("PUT /api/models/active (bad body): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestActiveModel_PUT_NotFound(t *testing.T) {
	s := setupSettingsTestServer(t)

	body := `{"name":"nonexistent-model"}`
	req := httptest.NewRequest(http.MethodPut, "/api/models/active", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleActiveModel(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("PUT /api/models/active (not found): status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestActiveModel_MethodNotAllowed(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/models/active", nil)
	w := httptest.NewRecorder()

	s.handleActiveModel(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE /api/models/active: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ==========================================
// handleMCPServers Tests
// ==========================================

func TestMCPServers_GET(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp/servers", nil)
	w := httptest.NewRecorder()

	s.handleMCPServers(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/mcp/servers: status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	servers, ok := resp["servers"].([]interface{})
	if !ok {
		t.Fatal("Expected 'servers' array in response")
	}
	if len(servers) != 2 {
		t.Errorf("len(servers) = %d, want 2", len(servers))
	}

	// Check first server
	first := servers[0].(map[string]interface{})
	if first["name"] != "kubectl-server" {
		t.Errorf("first server name = %v, want kubectl-server", first["name"])
	}
	if first["enabled"] != true {
		t.Error("Expected kubectl-server to be enabled")
	}
	// Not connected (no real MCP server running)
	if first["connected"] != false {
		t.Error("Expected kubectl-server to not be connected in test")
	}

	// Check connected servers list
	connected, ok := resp["connected"].([]interface{})
	if !ok {
		// connected could be nil (empty slice is encoded as null)
		if resp["connected"] != nil {
			t.Errorf("Expected 'connected' to be empty array or nil, got %v", resp["connected"])
		}
	} else if len(connected) != 0 {
		t.Errorf("Expected no connected servers, got %d", len(connected))
	}
}

func TestMCPServers_POST_InvalidBody(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/mcp/servers", strings.NewReader("bad"))
	w := httptest.NewRecorder()

	s.handleMCPServers(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/mcp/servers (bad body): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMCPServers_POST_MissingFields(t *testing.T) {
	s := setupSettingsTestServer(t)

	body := `{"name":"","command":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/mcp/servers", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleMCPServers(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/mcp/servers (empty): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMCPServers_PUT_InvalidBody(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/mcp/servers", strings.NewReader("bad"))
	w := httptest.NewRecorder()

	s.handleMCPServers(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("PUT /api/mcp/servers (bad body): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMCPServers_PUT_InvalidAction(t *testing.T) {
	s := setupSettingsTestServer(t)

	body := `{"name":"kubectl-server","action":"invalid"}`
	req := httptest.NewRequest(http.MethodPut, "/api/mcp/servers", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleMCPServers(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("PUT /api/mcp/servers (invalid action): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMCPServers_DELETE_MissingName(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/mcp/servers", nil)
	w := httptest.NewRecorder()

	s.handleMCPServers(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("DELETE /api/mcp/servers (no name): status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMCPServers_DELETE_NotFound(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/mcp/servers?name=nonexistent", nil)
	w := httptest.NewRecorder()

	s.handleMCPServers(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("DELETE /api/mcp/servers?name=nonexistent: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestMCPServers_MethodNotAllowed(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/mcp/servers", nil)
	w := httptest.NewRecorder()

	s.handleMCPServers(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("PATCH /api/mcp/servers: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ==========================================
// handleMCPTools Tests
// ==========================================

func TestMCPTools_GET(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp/tools", nil)
	w := httptest.NewRecorder()

	s.handleMCPTools(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/mcp/tools: status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := resp["mcp_tools"]; !ok {
		t.Error("Response missing 'mcp_tools' key")
	}
}

func TestMCPTools_MethodNotAllowed(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/mcp/tools", nil)
	w := httptest.NewRecorder()

	s.handleMCPTools(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/mcp/tools: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ==========================================
// handleAuditLogs Tests
// ==========================================

func TestAuditLogs_MethodNotAllowed(t *testing.T) {
	s := setupSettingsTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/audit", nil)
	w := httptest.NewRecorder()

	s.handleAuditLogs(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/audit: status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestAuditLogs_GET_WithDB(t *testing.T) {
	s := setupSettingsTestServer(t)

	// Initialize a temp DB for audit logs
	tmpDir, err := os.MkdirTemp("", "k13d-audit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := db.Init(filepath.Join(tmpDir, "test.db")); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	// Record some audit entries
	db.RecordAudit(db.AuditEntry{
		User:     "admin",
		Action:   "test_action",
		Resource: "test_resource",
		Details:  "Test details",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/audit", nil)
	w := httptest.NewRecorder()

	s.handleAuditLogs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/audit: status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := resp["logs"]; !ok {
		t.Error("Response missing 'logs' key")
	}
	if _, ok := resp["timestamp"]; !ok {
		t.Error("Response missing 'timestamp' key")
	}
}

func TestAuditLogs_GET_WithFilters(t *testing.T) {
	s := setupSettingsTestServer(t)

	tmpDir, err := os.MkdirTemp("", "k13d-audit-filter-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := db.Init(filepath.Join(tmpDir, "test.db")); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/audit?only_llm=true&user=admin&action=deploy&resource=pods", nil)
	w := httptest.NewRecorder()

	s.handleAuditLogs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/audit (with filters): status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// With filters, should return valid (possibly empty) results
	if _, ok := resp["logs"]; !ok {
		t.Error("Response missing 'logs' key")
	}
}

func TestAuditLogs_GET_NilDB(t *testing.T) {
	s := setupSettingsTestServer(t)

	// Ensure DB is nil
	savedDB := db.DB
	db.DB = nil
	defer func() { db.DB = savedDB }()

	req := httptest.NewRequest(http.MethodGet, "/api/audit", nil)
	w := httptest.NewRecorder()

	s.handleAuditLogs(w, req)

	// With nil DB, GetAuditLogsFiltered returns nil,nil - response should be 200 with null logs
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/audit (nil DB): status = %d, want %d", w.Code, http.StatusOK)
	}
}
