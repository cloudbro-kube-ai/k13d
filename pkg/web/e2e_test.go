package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
)

// E2E Test helpers
func setupTestServer(t *testing.T) (*Server, *AuthManager) {
	t.Helper()

	cfg := &config.Config{
		Language:     "en",
		BeginnerMode: false,
		EnableAudit:  true,
		LogLevel:     "debug",
		LLM: config.LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
			Endpoint: "",
		},
	}

	authConfig := &AuthConfig{
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local", // Use local auth mode for tests
		DefaultAdmin:    "admin",
		DefaultPassword: "admin123",
		Quiet:           true,
	}
	authManager := NewAuthManager(authConfig)

	// Create fake k8s client
	fakeClientset := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-1",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				NodeName: "test-node",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				PodIP: "10.0.0.1",
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-service",
				Namespace: "default",
			},
			Spec: corev1.ServiceSpec{
				Type:      corev1.ServiceTypeClusterIP,
				ClusterIP: "10.0.0.100",
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-node",
				Labels: map[string]string{
					"node-role.kubernetes.io/control-plane": "",
				},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
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
	server.reportGenerator = NewReportGenerator(server)

	return server, authManager
}

// E2E Test: Full login flow
func TestE2E_LoginFlow(t *testing.T) {
	_, authManager := setupTestServer(t)

	// Step 1: Login
	loginBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "admin123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	authManager.HandleLogin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login failed: expected 200, got %d", w.Code)
	}

	var loginResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}

	token, ok := loginResp["token"].(string)
	if !ok || token == "" {
		t.Fatal("expected token in login response")
	}

	// Step 2: Access protected endpoint with token
	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meW := httptest.NewRecorder()

	authManager.AuthMiddleware(authManager.HandleCurrentUser).ServeHTTP(meW, meReq)

	if meW.Code != http.StatusOK {
		t.Errorf("expected 200 for authenticated request, got %d", meW.Code)
	}

	// Step 3: Logout
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutReq.AddCookie(&http.Cookie{Name: "k13d_session", Value: token})
	logoutW := httptest.NewRecorder()

	authManager.HandleLogout(logoutW, logoutReq)

	if logoutW.Code != http.StatusOK {
		t.Errorf("logout failed: expected 200, got %d", logoutW.Code)
	}

	// Step 4: Verify session is invalidated
	meReq2 := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq2.Header.Set("Authorization", "Bearer "+token)
	meW2 := httptest.NewRecorder()

	authManager.AuthMiddleware(authManager.HandleCurrentUser).ServeHTTP(meW2, meReq2)

	if meW2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalidated session, got %d", meW2.Code)
	}
}

// E2E Test: K8s resources access
func TestE2E_K8sResourcesAccess(t *testing.T) {
	server, authManager := setupTestServer(t)

	// Login first
	session, err := authManager.Authenticate("admin", "admin123")
	if err != nil {
		t.Fatalf("authentication failed: %v", err)
	}

	tests := []struct {
		name           string
		resource       string
		expectedKind   string
		expectedStatus int
		expectError    bool
	}{
		{"pods", "pods", "pods", http.StatusOK, false},
		{"services", "services", "services", http.StatusOK, false},
		{"namespaces", "namespaces", "namespaces", http.StatusOK, false},
		{"nodes", "nodes", "nodes", http.StatusOK, false},
		{"unknown", "unknown-resource", "unknown-resource", http.StatusOK, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/k8s/"+tt.resource+"?namespace=default", nil)
			req.Header.Set("Authorization", "Bearer "+session.ID)
			w := httptest.NewRecorder()

			authManager.AuthMiddleware(http.HandlerFunc(server.handleK8sResource)).ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp K8sResourceResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}

				if resp.Kind != tt.expectedKind {
					t.Errorf("expected kind %s, got %s", tt.expectedKind, resp.Kind)
				}

				// Check error field for unknown resources
				if tt.expectError && resp.Error == "" {
					t.Errorf("expected error in response for unknown resource")
				}
			}
		})
	}
}

// E2E Test: Health endpoint
func TestE2E_HealthEndpoint(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health check failed: expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse health response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}

	if resp["k8s_ready"] != true {
		t.Errorf("expected k8s_ready to be true")
	}
}

// E2E Test: Settings endpoint
func TestE2E_SettingsEndpoint(t *testing.T) {
	server, authManager := setupTestServer(t)

	session, _ := authManager.Authenticate("admin", "admin123")

	// GET settings
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	req.Header.Set("Authorization", "Bearer "+session.ID)
	w := httptest.NewRecorder()

	authManager.AuthMiddleware(http.HandlerFunc(server.handleSettings)).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("get settings failed: expected 200, got %d", w.Code)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &settings); err != nil {
		t.Fatalf("failed to parse settings: %v", err)
	}

	if settings["language"] != "en" {
		t.Errorf("expected language 'en', got %v", settings["language"])
	}
}

// E2E Test: User management flow
func TestE2E_UserManagement(t *testing.T) {
	_, authManager := setupTestServer(t)

	// Create a new user
	err := authManager.CreateUser("testuser", "testpass123", "user")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Login with new user
	session, err := authManager.Authenticate("testuser", "testpass123")
	if err != nil {
		t.Fatalf("failed to login with new user: %v", err)
	}

	if session.Role != "user" {
		t.Errorf("expected role 'user', got %s", session.Role)
	}

	// Change password
	err = authManager.ChangePassword("testuser", "testpass123", "newpass456")
	if err != nil {
		t.Fatalf("failed to change password: %v", err)
	}

	// Login with new password
	_, err = authManager.Authenticate("testuser", "newpass456")
	if err != nil {
		t.Error("failed to login with new password")
	}

	// Old password should fail
	_, err = authManager.Authenticate("testuser", "testpass123")
	if err == nil {
		t.Error("old password should not work")
	}

	// Delete user
	err = authManager.DeleteUser("testuser")
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	// Login should fail
	_, err = authManager.Authenticate("testuser", "newpass456")
	if err == nil {
		t.Error("deleted user should not be able to login")
	}
}

// E2E Test: LDAP integration (when disabled)
func TestE2E_LDAPDisabled(t *testing.T) {
	authConfig := &AuthConfig{
		Enabled:         true,
		SessionDuration: time.Hour,
		LDAP:            nil,
	}
	authManager := NewAuthManager(authConfig)

	if authManager.IsLDAPEnabled() {
		t.Error("LDAP should be disabled")
	}

	ldapConfig := authManager.GetLDAPConfig()
	if ldapConfig != nil {
		t.Error("LDAP config should be nil when disabled")
	}
}

// E2E Test: LDAP integration (when enabled)
func TestE2E_LDAPEnabled(t *testing.T) {
	authConfig := &AuthConfig{
		Enabled:         true,
		SessionDuration: time.Hour,
		LDAP: &LDAPConfig{
			Enabled:      true,
			Host:         "ldap.example.com",
			Port:         389,
			AdminGroups:  []string{"k8s-admins"},
			UserGroups:   []string{"k8s-users"},
			ViewerGroups: []string{"k8s-viewers"},
		},
	}
	authManager := NewAuthManager(authConfig)

	if !authManager.IsLDAPEnabled() {
		t.Error("LDAP should be enabled")
	}

	ldapConfig := authManager.GetLDAPConfig()
	if ldapConfig == nil {
		t.Fatal("LDAP config should not be nil")
	}

	if ldapConfig.Host != "ldap.example.com" {
		t.Errorf("expected host 'ldap.example.com', got %s", ldapConfig.Host)
	}

	// Test LDAP status endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/auth/ldap/status", nil)
	w := httptest.NewRecorder()

	authManager.HandleLDAPStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("LDAP status failed: expected 200, got %d", w.Code)
	}

	var status map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("failed to parse LDAP status: %v", err)
	}

	if status["enabled"] != true {
		t.Error("expected LDAP to be enabled in status")
	}
}

// E2E Test: Reports generation
func TestE2E_ReportsGeneration(t *testing.T) {
	server, authManager := setupTestServer(t)

	session, _ := authManager.Authenticate("admin", "admin123")

	tests := []struct {
		name       string
		reportType string
		wantStatus int
	}{
		{"cluster health", "cluster-health", http.StatusOK},
		{"resource usage", "resource-usage", http.StatusOK},
		{"security audit", "security-audit", http.StatusOK},
		{"ai interactions", "ai-interactions", http.StatusOK},
		{"unknown type returns available types", "unknown-type", http.StatusOK}, // Returns available_types list
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/reports?type="+tt.reportType, nil)
			req.Header.Set("Authorization", "Bearer "+session.ID)
			w := httptest.NewRecorder()

			authManager.AuthMiddleware(http.HandlerFunc(server.reportGenerator.HandleReports)).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

// E2E Test: Reports comprehensive report generation
func TestE2E_ReportsComprehensive(t *testing.T) {
	server, authManager := setupTestServer(t)

	session, _ := authManager.Authenticate("admin", "admin123")

	req := httptest.NewRequest(http.MethodGet, "/api/reports", nil)
	req.Header.Set("Authorization", "Bearer "+session.ID)
	w := httptest.NewRecorder()

	authManager.AuthMiddleware(http.HandlerFunc(server.reportGenerator.HandleReports)).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Check for comprehensive report fields
	if _, ok := resp["generated_at"]; !ok {
		t.Error("expected generated_at in response")
	}

	if _, ok := resp["cluster_info"]; !ok {
		t.Error("expected cluster_info in response")
	}

	if _, ok := resp["health_score"]; !ok {
		t.Error("expected health_score in response")
	}
}

// E2E Test: Agentic chat endpoint without AI client
func TestE2E_AgenticChatWithoutAI(t *testing.T) {
	server, authManager := setupTestServer(t)

	session, _ := authManager.Authenticate("admin", "admin123")

	body, _ := json.Marshal(ChatRequest{Message: "Hello"})
	req := httptest.NewRequest(http.MethodPost, "/api/chat/agentic", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+session.ID)
	w := httptest.NewRecorder()

	authManager.AuthMiddleware(http.HandlerFunc(server.handleAgenticChat)).ServeHTTP(w, req)

	// Should return 503 Service Unavailable when AI client is not configured
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

// E2E Test: Agentic chat endpoint requires tool support
func TestE2E_AgenticChatRequiresToolSupport(t *testing.T) {
	server, authManager := setupTestServer(t)

	// Create mock AI server that doesn't support tools
	mockAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		resp := `{"id":"test","choices":[{"delta":{"content":"AI Response"},"finish_reason":"stop"}]}`
		w.Write([]byte("data: " + resp + "\n\n"))
		flusher.Flush()
		w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer mockAIServer.Close()

	// Set up AI client (without tool support simulation - we test the check)
	aiCfg := &config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
		Endpoint: mockAIServer.URL,
		APIKey:   "test-key",
	}
	aiClient, err := ai.NewClient(aiCfg)
	if err != nil {
		t.Fatalf("failed to create AI client: %v", err)
	}
	server.aiClient = aiClient

	session, _ := authManager.Authenticate("admin", "admin123")

	body, _ := json.Marshal(ChatRequest{Message: "Hello AI"})
	req := httptest.NewRequest(http.MethodPost, "/api/chat/agentic", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+session.ID)
	w := httptest.NewRecorder()

	authManager.AuthMiddleware(http.HandlerFunc(server.handleAgenticChat)).ServeHTTP(w, req)

	// The response depends on whether the AI client supports tools
	// If it doesn't support tools, we expect 400 Bad Request
	// If it does, we expect 200 OK with SSE response
	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Errorf("expected 200 or 400, got %d", w.Code)
	}
}

// E2E Test: CORS middleware
func TestE2E_CORSHeaders(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test preflight request with allowed origin
	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// With allowed origin, expect CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:8080" {
		t.Errorf("expected Access-Control-Allow-Origin: http://localhost:8080, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}

	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}

	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("expected Access-Control-Allow-Headers header")
	}

	// Test with disallowed origin
	req2 := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req2.Header.Set("Origin", "http://evil.com")
	w2 := httptest.NewRecorder()

	handler.ServeHTTP(w2, req2)

	// With disallowed origin, no CORS origin header should be set
	if w2.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected no Access-Control-Allow-Origin for disallowed origin, got %q", w2.Header().Get("Access-Control-Allow-Origin"))
	}
}

// E2E Test: Session expiration
func TestE2E_SessionExpiration(t *testing.T) {
	authConfig := &AuthConfig{
		Enabled:         true,
		SessionDuration: 100 * time.Millisecond, // Very short for testing
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "admin123",
		Quiet:           true,
	}
	authManager := NewAuthManager(authConfig)

	// Login
	session, err := authManager.Authenticate("admin", "admin123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Verify session works
	_, err = authManager.ValidateSession(session.ID)
	if err != nil {
		t.Error("session should be valid immediately after login")
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Session should be expired
	_, err = authManager.ValidateSession(session.ID)
	if err == nil {
		t.Error("session should be expired")
	}
}

// E2E Test: Audit logs endpoint
func TestE2E_AuditLogs(t *testing.T) {
	server, authManager := setupTestServer(t)

	// Initialize test database
	tmpDir, err := os.MkdirTemp("", "k13d-e2e-audit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = db.Init(filepath.Join(tmpDir, "e2e_audit.db"))
	if err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer db.Close()

	// Insert test audit entries
	testEntries := []db.AuditEntry{
		{User: "admin", Action: "delete", Resource: "pod/nginx", ActionType: db.ActionTypeMutation, Source: "web", Success: true},
		{User: "admin", Action: "scale", Resource: "deployment/app", ActionType: db.ActionTypeMutation, Source: "tui", Success: true},
		{User: "user1", Action: "ask", Resource: "pod/nginx", ActionType: db.ActionTypeLLM, Source: "web", Success: true, LLMTool: "kubectl"},
	}

	for _, e := range testEntries {
		if err := db.RecordAudit(e); err != nil {
			t.Fatalf("Failed to record audit: %v", err)
		}
	}

	session, err := authManager.Authenticate("admin", "admin123")
	if err != nil {
		t.Fatalf("authentication failed: %v", err)
	}

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCount  int // -1 to skip count check
	}{
		{"all logs", "", http.StatusOK, 3},
		{"filter by user", "user=admin", http.StatusOK, 2},
		{"filter by source", "source=web", http.StatusOK, 2},
		{"filter only LLM", "only_llm=true", http.StatusOK, 1},
		{"filter by action", "action=delete", http.StatusOK, 1},
		{"filter by resource", "resource=nginx", http.StatusOK, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/audit"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.Header.Set("Authorization", "Bearer "+session.ID)
			w := httptest.NewRecorder()

			authManager.AuthMiddleware(http.HandlerFunc(server.handleAuditLogs)).ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}

				logs, ok := resp["logs"].([]interface{})
				if !ok {
					t.Fatal("expected logs array in response")
				}

				if tt.expectedCount >= 0 && len(logs) != tt.expectedCount {
					t.Errorf("expected %d logs, got %d", tt.expectedCount, len(logs))
				}

				// Verify timestamp is present
				if _, ok := resp["timestamp"]; !ok {
					t.Error("expected timestamp in response")
				}
			}
		})
	}
}

// E2E Test: Audit logs unauthorized access
func TestE2E_AuditLogsUnauthorized(t *testing.T) {
	server, authManager := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/audit", nil)
	w := httptest.NewRecorder()

	// Call without authentication
	authManager.AuthMiddleware(http.HandlerFunc(server.handleAuditLogs)).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// E2E Test: Audit logs method not allowed
func TestE2E_AuditLogsMethodNotAllowed(t *testing.T) {
	server, authManager := setupTestServer(t)

	session, _ := authManager.Authenticate("admin", "admin123")

	req := httptest.NewRequest(http.MethodPost, "/api/audit", nil)
	req.Header.Set("Authorization", "Bearer "+session.ID)
	w := httptest.NewRecorder()

	authManager.AuthMiddleware(http.HandlerFunc(server.handleAuditLogs)).ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// E2E Test: Audit logs with database migration
func TestE2E_AuditLogsWithMigration(t *testing.T) {
	server, authManager := setupTestServer(t)

	// Create database with old schema, then re-init to trigger migration
	tmpDir, err := os.MkdirTemp("", "k13d-e2e-migrate-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "migrate_test.db")

	// Initialize with full schema first
	err = db.Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}

	// Drop and recreate with minimal schema
	_, err = db.DB.Exec("DROP TABLE IF EXISTS audit_logs")
	if err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}

	// Create old-style table
	_, err = db.DB.Exec(`
		CREATE TABLE audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			user TEXT,
			action TEXT,
			resource TEXT,
			details TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create old table: %v", err)
	}

	// Insert record with old schema
	_, err = db.DB.Exec(`INSERT INTO audit_logs (user, action, resource, details) VALUES (?, ?, ?, ?)`,
		"legacyuser", "update", "deployment/legacy", "legacy operation")
	if err != nil {
		t.Fatalf("Failed to insert legacy record: %v", err)
	}

	db.Close()

	// Re-initialize - this should trigger migration
	err = db.Init(dbPath)
	if err != nil {
		t.Fatalf("Re-init failed: %v", err)
	}
	defer db.Close()

	// Add a new record after migration
	err = db.RecordAudit(db.AuditEntry{
		User:       "newuser",
		Action:     "create",
		Resource:   "pod/new",
		ActionType: db.ActionTypeMutation,
		Source:     "web",
		Success:    true,
	})
	if err != nil {
		t.Fatalf("Failed to record audit after migration: %v", err)
	}

	// Now test the API
	session, _ := authManager.Authenticate("admin", "admin123")

	req := httptest.NewRequest(http.MethodGet, "/api/audit", nil)
	req.Header.Set("Authorization", "Bearer "+session.ID)
	w := httptest.NewRecorder()

	authManager.AuthMiddleware(http.HandlerFunc(server.handleAuditLogs)).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
		return
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	logs, ok := resp["logs"].([]interface{})
	if !ok {
		t.Fatal("expected logs array in response")
	}

	// Should have both legacy and new records
	if len(logs) != 2 {
		t.Errorf("expected 2 logs (1 legacy + 1 new), got %d", len(logs))
	}
}

// E2E Test: Security Headers middleware
func TestE2E_SecurityHeaders(t *testing.T) {
	handler := securityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check all security headers
	tests := []struct {
		header   string
		expected string
	}{
		{"X-Frame-Options", "DENY"},
		{"X-Content-Type-Options", "nosniff"},
		{"X-XSS-Protection", "1; mode=block"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, tt := range tests {
		value := w.Header().Get(tt.header)
		if value != tt.expected {
			t.Errorf("%s = %q, want %q", tt.header, value, tt.expected)
		}
	}

	// Check CSP header exists
	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected Content-Security-Policy header")
	}

	// Check Permissions-Policy header exists
	pp := w.Header().Get("Permissions-Policy")
	if pp == "" {
		t.Error("expected Permissions-Policy header")
	}
}

// E2E Test: CSRF Token generation and validation
func TestE2E_CSRFProtection(t *testing.T) {
	authConfig := &AuthConfig{
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "testpassword123",
		Quiet:           true,
	}
	authManager := NewAuthManager(authConfig)

	// Authenticate to get a session
	session, err := authManager.Authenticate("admin", "testpassword123")
	if err != nil {
		t.Fatalf("authentication failed: %v", err)
	}

	// Test 1: Generate CSRF token
	token := authManager.GenerateCSRFToken()
	if token == "" {
		t.Error("expected non-empty CSRF token")
	}

	// Test 2: Validate valid token
	if !authManager.ValidateCSRFToken(token) {
		t.Error("valid CSRF token should be accepted")
	}

	// Test 3: Invalid token should be rejected
	if authManager.ValidateCSRFToken("invalid-token") {
		t.Error("invalid CSRF token should be rejected")
	}

	// Test 4: CSRF middleware allows GET requests
	handler := authManager.CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+session.ID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET request should pass CSRF check, got %d", w.Code)
	}

	// Test 5: POST without CSRF token and without Bearer auth should fail
	// Note: CSRF check is skipped for Bearer token auth (API clients)
	// CSRF is enforced for cookie-based browser sessions
	postReq := httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewBufferString(`{"test":"data"}`))
	postReq.Header.Set("Content-Type", "application/json")
	// No Authorization header - simulating browser request with cookies
	w2 := httptest.NewRecorder()

	handler.ServeHTTP(w2, postReq)
	if w2.Code != http.StatusForbidden {
		t.Errorf("POST without CSRF token (no Bearer auth) should return 403, got %d", w2.Code)
	}

	// Test 6: POST with valid CSRF token (no Bearer auth) should pass
	postReq2 := httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewBufferString(`{"test":"data"}`))
	postReq2.Header.Set("Content-Type", "application/json")
	postReq2.Header.Set("X-CSRF-Token", token)
	w3 := httptest.NewRecorder()

	handler.ServeHTTP(w3, postReq2)
	if w3.Code != http.StatusOK {
		t.Errorf("POST with valid CSRF token should pass, got %d", w3.Code)
	}

	// Test 7: POST with Bearer auth skips CSRF check (for API clients)
	postReq3 := httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewBufferString(`{"test":"data"}`))
	postReq3.Header.Set("Authorization", "Bearer "+session.ID)
	postReq3.Header.Set("Content-Type", "application/json")
	// No CSRF token, but has Bearer auth
	w4 := httptest.NewRecorder()

	handler.ServeHTTP(w4, postReq3)
	if w4.Code != http.StatusOK {
		t.Errorf("POST with Bearer auth should skip CSRF check, got %d", w4.Code)
	}

	_ = session // silence unused warning
}

// E2E Test: CSRF Token endpoint
func TestE2E_CSRFTokenEndpoint(t *testing.T) {
	authConfig := &AuthConfig{
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "testpassword123",
		Quiet:           true,
	}
	authManager := NewAuthManager(authConfig)

	// Test GET request to CSRF token endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/auth/csrf-token", nil)
	w := httptest.NewRecorder()

	authManager.HandleCSRFToken(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	token, ok := resp["csrf_token"]
	if !ok || token == "" {
		t.Error("expected csrf_token in response")
	}

	// Verify the token is valid
	if !authManager.ValidateCSRFToken(token) {
		t.Error("returned CSRF token should be valid")
	}
}

// E2E Test: Cookie security flags
func TestE2E_CookieSecurityFlags(t *testing.T) {
	authConfig := &AuthConfig{
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "testpassword123",
		Quiet:           true,
	}
	authManager := NewAuthManager(authConfig)

	// Test login to get session cookie
	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "testpassword123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	authManager.HandleLogin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login failed: %d - %s", w.Code, w.Body.String())
	}

	// Check cookie attributes
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "k13d_session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected k13d_session cookie")
	}

	if !sessionCookie.HttpOnly {
		t.Error("session cookie should have HttpOnly flag")
	}

	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("session cookie SameSite = %v, want Lax", sessionCookie.SameSite)
	}
}

// E2E Test: CORS with credentials
func TestE2E_CORSWithCredentials(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with allowed origin
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check credentials support
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("expected Access-Control-Allow-Credentials: true for allowed origin")
	}

	// Test with disallowed origin - should not have credentials header
	req2 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req2.Header.Set("Origin", "http://attacker.com")
	w2 := httptest.NewRecorder()

	handler.ServeHTTP(w2, req2)

	if w2.Header().Get("Access-Control-Allow-Credentials") == "true" {
		t.Error("should not allow credentials for disallowed origin")
	}
}

// E2E Test: Secure password generation (no hardcoded defaults)
func TestE2E_SecurePasswordGeneration(t *testing.T) {
	// Test that generateSecurePassword creates unique passwords
	passwords := make(map[string]bool)
	for i := 0; i < 10; i++ {
		pwd := generateSecurePassword(16)
		if len(pwd) < 16 {
			t.Errorf("password length %d < 16", len(pwd))
		}
		if passwords[pwd] {
			t.Error("generated duplicate password")
		}
		passwords[pwd] = true
	}
}

// E2E Test: Full middleware chain
func TestE2E_FullMiddlewareChain(t *testing.T) {
	server, authManager := setupTestServer(t)

	// Create the full middleware chain as used in production
	mux := http.NewServeMux()
	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	handler := securityHeadersMiddleware(corsMiddleware(authManager.CSRFMiddleware(mux)))

	// Authenticate
	session, _ := authManager.Authenticate("admin", "admin123")
	csrfToken := authManager.GenerateCSRFToken()

	// Test POST with all security measures
	body := bytes.NewBufferString(`{"data":"test"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/test", body)
	req.Header.Set("Origin", "http://localhost:8080")
	req.Header.Set("Authorization", "Bearer "+session.ID)
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should pass all middleware checks
	if w.Code != http.StatusOK {
		t.Errorf("full middleware chain should pass, got %d: %s", w.Code, w.Body.String())
	}

	// Verify security headers are present
	if w.Header().Get("X-Frame-Options") == "" {
		t.Error("missing X-Frame-Options header")
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:8080" {
		t.Error("missing or incorrect CORS header")
	}

	_ = server // silence unused warning
}
