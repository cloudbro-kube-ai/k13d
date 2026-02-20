package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/db"
)

// ==================== Feature Permission Tests ====================

func TestFeaturePermissions_AdminHasAllFeatures(t *testing.T) {
	az := NewAuthorizer()

	for _, f := range AllFeatures() {
		if !az.IsFeatureAllowed("admin", f) {
			t.Errorf("admin should have access to feature %s", f)
		}
	}
}

func TestFeaturePermissions_ViewerLimitedFeatures(t *testing.T) {
	az := NewAuthorizer()

	// Viewer should have dashboard
	if !az.IsFeatureAllowed("viewer", FeatureDashboard) {
		t.Error("viewer should have access to dashboard")
	}
	if !az.IsFeatureAllowed("viewer", FeatureTopology) {
		t.Error("viewer should have access to topology")
	}
	if !az.IsFeatureAllowed("viewer", FeatureMetrics) {
		t.Error("viewer should have access to metrics")
	}

	// Viewer should NOT have helm, terminal, AI, etc.
	deniedFeatures := []Feature{
		FeatureHelmManagement, FeatureSecurityScan, FeatureAIAssistant,
		FeatureTerminal, FeatureReports, FeatureTemplates,
		FeaturePortForward, FeatureGitOps, FeatureVelero,
	}
	for _, f := range deniedFeatures {
		if az.IsFeatureAllowed("viewer", f) {
			t.Errorf("viewer should NOT have access to feature %s", f)
		}
	}
}

func TestFeaturePermissions_UserDenyOverridesWildcard(t *testing.T) {
	az := NewAuthorizer()

	// User role has AllowedFeatures: ["*"] but DeniedFeatures: [SettingsAdmin, SettingsSecurity]
	if !az.IsFeatureAllowed("user", FeatureDashboard) {
		t.Error("user should have access to dashboard (wildcard)")
	}
	if !az.IsFeatureAllowed("user", FeatureHelmManagement) {
		t.Error("user should have access to helm (wildcard)")
	}

	// Denied features should be blocked despite wildcard allow
	if az.IsFeatureAllowed("user", FeatureSettingsAdmin) {
		t.Error("user should NOT have access to settings_admin (deny overrides)")
	}
	if az.IsFeatureAllowed("user", FeatureSettingsSecurity) {
		t.Error("user should NOT have access to settings_security (deny overrides)")
	}
}

func TestFeaturePermissions_UnknownRoleDenied(t *testing.T) {
	az := NewAuthorizer()

	if az.IsFeatureAllowed("nonexistent", FeatureDashboard) {
		t.Error("unknown role should be denied all features")
	}
}

func TestFeaturePermissions_CustomRole(t *testing.T) {
	az := NewAuthorizer()

	customRole := &RoleDefinition{
		Name:            "developer",
		IsCustom:        true,
		AllowedFeatures: []Feature{FeatureDashboard, FeatureTopology, FeatureAIAssistant},
	}
	az.RegisterRole(customRole)

	if !az.IsFeatureAllowed("developer", FeatureDashboard) {
		t.Error("developer should have dashboard access")
	}
	if !az.IsFeatureAllowed("developer", FeatureAIAssistant) {
		t.Error("developer should have AI assistant access")
	}
	if az.IsFeatureAllowed("developer", FeatureHelmManagement) {
		t.Error("developer should not have helm access")
	}
	if az.IsFeatureAllowed("developer", FeatureTerminal) {
		t.Error("developer should not have terminal access")
	}
}

func TestFeaturePermissions_CustomRoleWithDeny(t *testing.T) {
	az := NewAuthorizer()

	customRole := &RoleDefinition{
		Name:            "semi-admin",
		IsCustom:        true,
		AllowedFeatures: []Feature{"*"},
		DeniedFeatures:  []Feature{FeatureTerminal, FeatureGitOps},
	}
	az.RegisterRole(customRole)

	if !az.IsFeatureAllowed("semi-admin", FeatureDashboard) {
		t.Error("semi-admin should have dashboard via wildcard")
	}
	if az.IsFeatureAllowed("semi-admin", FeatureTerminal) {
		t.Error("semi-admin should be denied terminal")
	}
	if az.IsFeatureAllowed("semi-admin", FeatureGitOps) {
		t.Error("semi-admin should be denied gitops")
	}
}

// ==================== Feature Middleware Tests ====================

func TestFeatureMiddleware_Denied(t *testing.T) {
	az := NewAuthorizer()
	handler := az.FeatureMiddleware(FeatureHelmManagement)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-User-Role", "viewer")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer accessing helm, got %d", rec.Code)
	}
}

func TestFeatureMiddleware_Allowed(t *testing.T) {
	az := NewAuthorizer()
	handler := az.FeatureMiddleware(FeatureDashboard)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-User-Role", "viewer")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for viewer accessing dashboard, got %d", rec.Code)
	}
}

func TestFeatureMiddleware_DefaultsToViewer(t *testing.T) {
	az := NewAuthorizer()
	handler := az.FeatureMiddleware(FeatureTerminal)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// No role header -> defaults to viewer -> viewer has no terminal access
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for no-role accessing terminal, got %d", rec.Code)
	}
}

func TestFeatureMiddleware_AdminAllowed(t *testing.T) {
	az := NewAuthorizer()
	handler := az.FeatureMiddleware(FeatureSettingsAdmin)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-User-Role", "admin")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin accessing settings_admin, got %d", rec.Code)
	}
}

// ==================== Role CRUD Tests ====================

func TestDeleteRole_BuiltIn(t *testing.T) {
	az := NewAuthorizer()

	for _, name := range []string{"admin", "user", "viewer"} {
		err := az.DeleteRole(name)
		if err == nil {
			t.Errorf("should not be able to delete built-in role %s", name)
		}
		if !strings.Contains(err.Error(), "built-in") {
			t.Errorf("error should mention built-in, got: %s", err.Error())
		}
	}
}

func TestDeleteRole_Custom(t *testing.T) {
	az := NewAuthorizer()

	az.RegisterRole(&RoleDefinition{Name: "temp-role", IsCustom: true})

	err := az.DeleteRole("temp-role")
	if err != nil {
		t.Errorf("should be able to delete custom role: %v", err)
	}

	// Verify it's gone
	if az.GetRole("temp-role") != nil {
		t.Error("role should be deleted")
	}
}

func TestDeleteRole_NotFound(t *testing.T) {
	az := NewAuthorizer()

	err := az.DeleteRole("nonexistent")
	if err == nil {
		t.Error("should error when deleting non-existent role")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found, got: %s", err.Error())
	}
}

func TestListRoles(t *testing.T) {
	az := NewAuthorizer()
	roles := az.ListRoles()

	if len(roles) < 3 {
		t.Errorf("expected at least 3 default roles, got %d", len(roles))
	}

	// Verify expected role names exist
	roleNames := make(map[string]bool)
	for _, r := range roles {
		roleNames[r.Name] = true
	}
	for _, expected := range []string{"admin", "user", "viewer"} {
		if !roleNames[expected] {
			t.Errorf("expected role %s in list", expected)
		}
	}
}

func TestGetRole(t *testing.T) {
	az := NewAuthorizer()

	role := az.GetRole("admin")
	if role == nil {
		t.Fatal("expected admin role to exist")
	}
	if role.Name != "admin" {
		t.Errorf("expected role name admin, got %s", role.Name)
	}

	role = az.GetRole("nonexistent")
	if role != nil {
		t.Error("expected nil for non-existent role")
	}
}

func TestGetFeaturePermissions(t *testing.T) {
	az := NewAuthorizer()

	perms := az.GetFeaturePermissions("admin")
	if len(perms) == 0 {
		t.Error("admin should have permissions")
	}
	for f, allowed := range perms {
		if !allowed {
			t.Errorf("admin should have all features allowed, but %s is denied", f)
		}
	}

	perms = az.GetFeaturePermissions("viewer")
	if perms[FeatureDashboard] != true {
		t.Error("viewer should have dashboard access")
	}
	if perms[FeatureTerminal] != false {
		t.Error("viewer should not have terminal access")
	}
}

func TestGetFeaturePermissions_UnknownRole(t *testing.T) {
	az := NewAuthorizer()

	perms := az.GetFeaturePermissions("nonexistent")
	for f, allowed := range perms {
		if allowed {
			t.Errorf("unknown role should have no features, but %s is allowed", f)
		}
	}
}

// ==================== Role Handler Tests ====================

func setupRoleTestServer(t *testing.T) *Server {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	if err := db.InitCustomRolesTable(); err != nil {
		t.Fatalf("Failed to init custom_roles table: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
	})

	cfg := config.NewDefaultConfig()
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
		authManager:      NewAuthManager(authConfig),
		authorizer:       NewAuthorizer(),
		port:             8080,
		pendingApprovals: make(map[string]*PendingToolApproval),
	}
}

func TestHandleRoles_ListReturnsDefaultRoles(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/roles", nil)
	w := httptest.NewRecorder()

	s.handleRoles(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	roles, ok := body["roles"].([]interface{})
	if !ok {
		t.Fatal("expected roles array in response")
	}
	if len(roles) < 3 {
		t.Errorf("expected at least 3 roles, got %d", len(roles))
	}
}

func TestHandleRoles_CreateCustomRole(t *testing.T) {
	s := setupRoleTestServer(t)

	roleJSON := `{
		"name": "devops",
		"description": "DevOps team role",
		"allowed_features": ["dashboard", "topology", "helm_management"]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/roles", strings.NewReader(roleJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleRoles(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if body["status"] != "created" {
		t.Errorf("expected status=created, got %v", body["status"])
	}

	// Verify the role is registered in memory
	role := s.authorizer.GetRole("devops")
	if role == nil {
		t.Fatal("expected devops role to be registered")
	}
	if !role.IsCustom {
		t.Error("expected IsCustom=true")
	}
}

func TestHandleRoles_CreateBuiltInNameConflict(t *testing.T) {
	s := setupRoleTestServer(t)

	roleJSON := `{"name": "admin"}`
	req := httptest.NewRequest(http.MethodPost, "/api/roles", strings.NewReader(roleJSON))
	w := httptest.NewRecorder()

	s.handleRoles(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for built-in name, got %d", w.Code)
	}
}

func TestHandleRoles_CreateEmptyName(t *testing.T) {
	s := setupRoleTestServer(t)

	roleJSON := `{"name": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/roles", strings.NewReader(roleJSON))
	w := httptest.NewRecorder()

	s.handleRoles(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", w.Code)
	}
}

func TestHandleRoles_CreateInvalidBody(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/roles", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	s.handleRoles(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d", w.Code)
	}
}

func TestHandleRoles_MethodNotAllowed(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/roles", nil)
	w := httptest.NewRecorder()

	s.handleRoles(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleRoleByName_GetRole(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/roles/admin", nil)
	w := httptest.NewRecorder()

	s.handleRoleByName(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var role RoleDefinition
	if err := json.Unmarshal(w.Body.Bytes(), &role); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if role.Name != "admin" {
		t.Errorf("expected name=admin, got %s", role.Name)
	}
}

func TestHandleRoleByName_GetNotFound(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/roles/nonexistent", nil)
	w := httptest.NewRecorder()

	s.handleRoleByName(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleRoleByName_DeleteCustomRole(t *testing.T) {
	s := setupRoleTestServer(t)

	// First create a custom role
	s.authorizer.RegisterRole(&RoleDefinition{Name: "temp", IsCustom: true})
	if err := db.SaveCustomRole("temp", `{"name":"temp"}`); err != nil {
		t.Fatalf("SaveCustomRole failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/roles/temp", nil)
	w := httptest.NewRecorder()

	s.handleRoleByName(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	// Verify it's gone from memory
	if s.authorizer.GetRole("temp") != nil {
		t.Error("role should be deleted from memory")
	}
}

func TestHandleRoleByName_DeleteBuiltInForbidden(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/roles/admin", nil)
	w := httptest.NewRecorder()

	s.handleRoleByName(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for deleting built-in role, got %d", w.Code)
	}
}

func TestHandleRoleByName_UpdateCustomRole(t *testing.T) {
	s := setupRoleTestServer(t)

	// Create a custom role first
	s.authorizer.RegisterRole(&RoleDefinition{Name: "devops", IsCustom: true})
	if err := db.SaveCustomRole("devops", `{"name":"devops"}`); err != nil {
		t.Fatalf("SaveCustomRole failed: %v", err)
	}

	updateJSON := `{
		"description": "Updated DevOps role",
		"allowed_features": ["dashboard", "topology", "helm_management", "terminal"]
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/roles/devops", strings.NewReader(updateJSON))
	w := httptest.NewRecorder()

	s.handleRoleByName(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	// Verify the update in memory
	role := s.authorizer.GetRole("devops")
	if role == nil {
		t.Fatal("expected devops role to exist")
	}
	if role.Description != "Updated DevOps role" {
		t.Errorf("expected updated description, got %s", role.Description)
	}
}

func TestHandleRoleByName_UpdateBuiltInForbidden(t *testing.T) {
	s := setupRoleTestServer(t)

	updateJSON := `{"description": "try to update"}`
	req := httptest.NewRequest(http.MethodPut, "/api/roles/admin", strings.NewReader(updateJSON))
	w := httptest.NewRecorder()

	s.handleRoleByName(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for updating built-in role, got %d", w.Code)
	}
}

func TestHandleRoleByName_EmptyName(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/roles/", nil)
	w := httptest.NewRecorder()

	s.handleRoleByName(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", w.Code)
	}
}

func TestHandleRoleByName_MethodNotAllowed(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/roles/admin", nil)
	w := httptest.NewRecorder()

	s.handleRoleByName(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ==================== User Permissions Handler Tests ====================

func TestHandleUserPermissions_ViewerRole(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/user/permissions", nil)
	req.Header.Set("X-User-Role", "viewer")
	w := httptest.NewRecorder()

	s.handleUserPermissions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if body["role"] != "viewer" {
		t.Errorf("expected role=viewer, got %v", body["role"])
	}

	features, ok := body["features"].(map[string]interface{})
	if !ok {
		t.Fatal("expected features map in response")
	}

	if features["dashboard"] != true {
		t.Error("expected dashboard=true for viewer")
	}
	if features["terminal"] != false {
		t.Error("expected terminal=false for viewer")
	}
}

func TestHandleUserPermissions_DefaultsToViewer(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/user/permissions", nil)
	// No X-User-Role header
	w := httptest.NewRecorder()

	s.handleUserPermissions(w, req)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if body["role"] != "viewer" {
		t.Errorf("expected default role=viewer, got %v", body["role"])
	}
}

func TestHandleUserPermissions_MethodNotAllowed(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/user/permissions", nil)
	w := httptest.NewRecorder()

	s.handleUserPermissions(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ==================== Tool Approval Settings Handler Tests ====================

func TestHandleToolApprovalSettings_Get(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/tool-approval", nil)
	w := httptest.NewRecorder()

	s.handleToolApprovalSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var policy config.ToolApprovalPolicy
	if err := json.Unmarshal(w.Body.Bytes(), &policy); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Default policy values
	if policy.ApprovalTimeoutSeconds != 60 {
		t.Errorf("expected default timeout=60, got %d", policy.ApprovalTimeoutSeconds)
	}
}

func TestHandleToolApprovalSettings_PutAdminOnly(t *testing.T) {
	s := setupRoleTestServer(t)

	policyJSON := `{
		"auto_approve_read_only": true,
		"require_approval_for_write": true,
		"block_dangerous": true,
		"approval_timeout_seconds": 120
	}`

	// Non-admin should be rejected
	req := httptest.NewRequest(http.MethodPut, "/api/settings/tool-approval", strings.NewReader(policyJSON))
	req.Header.Set("X-User-Role", "user")
	w := httptest.NewRecorder()

	s.handleToolApprovalSettings(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", w.Code)
	}

	// Admin should be allowed
	req = httptest.NewRequest(http.MethodPut, "/api/settings/tool-approval", strings.NewReader(policyJSON))
	req.Header.Set("X-User-Role", "admin")
	w = httptest.NewRecorder()

	s.handleToolApprovalSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin, got %d; body: %s", w.Code, w.Body.String())
	}

	// Verify the update
	s.aiMu.RLock()
	if !s.cfg.Authorization.ToolApproval.BlockDangerous {
		t.Error("expected BlockDangerous=true after update")
	}
	if s.cfg.Authorization.ToolApproval.ApprovalTimeoutSeconds != 120 {
		t.Errorf("expected timeout=120, got %d", s.cfg.Authorization.ToolApproval.ApprovalTimeoutSeconds)
	}
	s.aiMu.RUnlock()
}

func TestHandleToolApprovalSettings_TimeoutBounds(t *testing.T) {
	s := setupRoleTestServer(t)

	// Test timeout clamping: negative -> 60
	policyJSON := `{"approval_timeout_seconds": -5}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings/tool-approval", strings.NewReader(policyJSON))
	req.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()

	s.handleToolApprovalSettings(w, req)

	s.aiMu.RLock()
	timeout := s.cfg.Authorization.ToolApproval.ApprovalTimeoutSeconds
	s.aiMu.RUnlock()

	if timeout != 60 {
		t.Errorf("expected negative timeout clamped to 60, got %d", timeout)
	}

	// Test timeout clamping: over 600 -> 600
	policyJSON = `{"approval_timeout_seconds": 9999}`
	req = httptest.NewRequest(http.MethodPut, "/api/settings/tool-approval", strings.NewReader(policyJSON))
	req.Header.Set("X-User-Role", "admin")
	w = httptest.NewRecorder()

	s.handleToolApprovalSettings(w, req)

	s.aiMu.RLock()
	timeout = s.cfg.Authorization.ToolApproval.ApprovalTimeoutSeconds
	s.aiMu.RUnlock()

	if timeout != 600 {
		t.Errorf("expected high timeout clamped to 600, got %d", timeout)
	}
}

func TestHandleToolApprovalSettings_InvalidBody(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/tool-approval", strings.NewReader("not json"))
	req.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()

	s.handleToolApprovalSettings(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d", w.Code)
	}
}

func TestHandleToolApprovalSettings_MethodNotAllowed(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/settings/tool-approval", nil)
	w := httptest.NewRecorder()

	s.handleToolApprovalSettings(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestGetToolApprovalTimeout(t *testing.T) {
	s := setupRoleTestServer(t)

	// Default timeout
	timeout := s.getToolApprovalTimeout()
	if timeout != 60*time.Second {
		t.Errorf("expected default 60s, got %v", timeout)
	}

	// Custom timeout
	s.aiMu.Lock()
	s.cfg.Authorization.ToolApproval.ApprovalTimeoutSeconds = 120
	s.aiMu.Unlock()

	timeout = s.getToolApprovalTimeout()
	if timeout != 120*time.Second {
		t.Errorf("expected 120s, got %v", timeout)
	}

	// Zero timeout -> defaults to 60
	s.aiMu.Lock()
	s.cfg.Authorization.ToolApproval.ApprovalTimeoutSeconds = 0
	s.aiMu.Unlock()

	timeout = s.getToolApprovalTimeout()
	if timeout != 60*time.Second {
		t.Errorf("expected 0 clamped to 60s, got %v", timeout)
	}
}

// ==================== Agent Settings Handler Tests ====================

func TestHandleAgentSettings_Get(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/agent", nil)
	w := httptest.NewRecorder()

	s.handleAgentSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if body["max_iterations"] == nil {
		t.Error("expected max_iterations in response")
	}
	if body["temperature"] == nil {
		t.Error("expected temperature in response")
	}
}

func TestHandleAgentSettings_PutAdminOnly(t *testing.T) {
	s := setupRoleTestServer(t)

	settingsJSON := `{
		"max_iterations": 15,
		"reasoning_effort": "high",
		"temperature": 0.5,
		"max_tokens": 8192
	}`

	// Non-admin should be rejected
	req := httptest.NewRequest(http.MethodPut, "/api/settings/agent", strings.NewReader(settingsJSON))
	req.Header.Set("X-User-Role", "user")
	w := httptest.NewRecorder()

	s.handleAgentSettings(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", w.Code)
	}

	// Admin should be allowed
	req = httptest.NewRequest(http.MethodPut, "/api/settings/agent", strings.NewReader(settingsJSON))
	req.Header.Set("X-User-Role", "admin")
	w = httptest.NewRecorder()

	s.handleAgentSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	// Verify update
	s.aiMu.RLock()
	if s.cfg.LLM.MaxIterations != 15 {
		t.Errorf("expected max_iterations=15, got %d", s.cfg.LLM.MaxIterations)
	}
	if s.cfg.LLM.ReasoningEffort != "high" {
		t.Errorf("expected reasoning_effort=high, got %s", s.cfg.LLM.ReasoningEffort)
	}
	if s.cfg.LLM.Temperature != 0.5 {
		t.Errorf("expected temperature=0.5, got %f", s.cfg.LLM.Temperature)
	}
	if s.cfg.LLM.MaxTokens != 8192 {
		t.Errorf("expected max_tokens=8192, got %d", s.cfg.LLM.MaxTokens)
	}
	s.aiMu.RUnlock()
}

func TestHandleAgentSettings_ClampValues(t *testing.T) {
	s := setupRoleTestServer(t)

	// Test clamping: max_iterations over 30 -> 30, temperature over 2.0 -> 2.0
	settingsJSON := `{
		"max_iterations": 100,
		"temperature": 5.0,
		"max_tokens": -10
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings/agent", strings.NewReader(settingsJSON))
	req.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()

	s.handleAgentSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	s.aiMu.RLock()
	if s.cfg.LLM.MaxIterations != 30 {
		t.Errorf("expected max_iterations clamped to 30, got %d", s.cfg.LLM.MaxIterations)
	}
	if s.cfg.LLM.Temperature != 2.0 {
		t.Errorf("expected temperature clamped to 2.0, got %f", s.cfg.LLM.Temperature)
	}
	if s.cfg.LLM.MaxTokens != 0 {
		t.Errorf("expected max_tokens clamped to 0, got %d", s.cfg.LLM.MaxTokens)
	}
	s.aiMu.RUnlock()

	// Test clamping: below minimum
	settingsJSON = `{"max_iterations": 0, "temperature": -1.0}`
	req = httptest.NewRequest(http.MethodPut, "/api/settings/agent", strings.NewReader(settingsJSON))
	req.Header.Set("X-User-Role", "admin")
	w = httptest.NewRecorder()

	s.handleAgentSettings(w, req)

	s.aiMu.RLock()
	if s.cfg.LLM.MaxIterations != 1 {
		t.Errorf("expected max_iterations clamped to 1, got %d", s.cfg.LLM.MaxIterations)
	}
	if s.cfg.LLM.Temperature != 0 {
		t.Errorf("expected temperature clamped to 0, got %f", s.cfg.LLM.Temperature)
	}
	s.aiMu.RUnlock()
}

func TestHandleAgentSettings_InvalidReasoningEffort(t *testing.T) {
	s := setupRoleTestServer(t)

	// Set a known value first
	s.aiMu.Lock()
	s.cfg.LLM.ReasoningEffort = "medium"
	s.aiMu.Unlock()

	settingsJSON := `{"reasoning_effort": "invalid_value", "max_iterations": 5}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings/agent", strings.NewReader(settingsJSON))
	req.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()

	s.handleAgentSettings(w, req)

	// Invalid reasoning_effort should be ignored (kept as previous value)
	s.aiMu.RLock()
	if s.cfg.LLM.ReasoningEffort != "medium" {
		t.Errorf("expected reasoning_effort unchanged for invalid value, got %s", s.cfg.LLM.ReasoningEffort)
	}
	s.aiMu.RUnlock()
}

func TestHandleAgentSettings_InvalidBody(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/agent", strings.NewReader("not json"))
	req.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()

	s.handleAgentSettings(w, req)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if body["code"] != ErrCodeBadRequest {
		t.Errorf("expected error code %s, got %v", ErrCodeBadRequest, body["code"])
	}
}

func TestHandleAgentSettings_MethodNotAllowed(t *testing.T) {
	s := setupRoleTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/settings/agent", nil)
	w := httptest.NewRecorder()

	s.handleAgentSettings(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ==================== AllFeatures Tests ====================

func TestAllFeatures_Complete(t *testing.T) {
	features := AllFeatures()
	if len(features) == 0 {
		t.Error("AllFeatures should return non-empty list")
	}

	// Verify key features are in the list
	featureSet := make(map[Feature]bool)
	for _, f := range features {
		featureSet[f] = true
	}

	expectedFeatures := []Feature{
		FeatureDashboard, FeatureTopology, FeatureMetrics,
		FeatureHelmManagement, FeatureAIAssistant, FeatureTerminal,
		FeatureSettingsAdmin,
	}
	for _, f := range expectedFeatures {
		if !featureSet[f] {
			t.Errorf("expected feature %s in AllFeatures()", f)
		}
	}
}

// ==================== NewAuthorizerWithRoles Tests ====================

func TestNewAuthorizerWithRoles(t *testing.T) {
	customRoles := []RoleDefinition{
		{
			Name:            "ops",
			AllowedFeatures: []Feature{FeatureDashboard, FeatureTerminal},
			Allow: []ResourceRule{
				{Resources: []string{"*"}, Actions: []Action{ActionView}, Namespaces: []string{"*"}},
			},
		},
	}

	az := NewAuthorizerWithRoles(customRoles)

	// Custom role should exist
	if !az.IsFeatureAllowed("ops", FeatureDashboard) {
		t.Error("ops should have dashboard access")
	}

	// Built-in roles should still exist
	if !az.IsFeatureAllowed("admin", FeatureDashboard) {
		t.Error("admin should still have dashboard access")
	}
}

// ==================== Config ToolApprovalPolicy Tests ====================

func TestDefaultToolApprovalPolicy(t *testing.T) {
	policy := config.DefaultToolApprovalPolicy()

	if !policy.AutoApproveReadOnly {
		t.Error("expected AutoApproveReadOnly=true by default")
	}
	if !policy.RequireApprovalForWrite {
		t.Error("expected RequireApprovalForWrite=true by default")
	}
	if !policy.RequireApprovalForUnknown {
		t.Error("expected RequireApprovalForUnknown=true by default")
	}
	if policy.BlockDangerous {
		t.Error("expected BlockDangerous=false by default")
	}
	if policy.ApprovalTimeoutSeconds != 60 {
		t.Errorf("expected timeout=60 by default, got %d", policy.ApprovalTimeoutSeconds)
	}
	if policy.BlockedPatterns == nil {
		t.Error("expected BlockedPatterns non-nil")
	}
}

// ==================== BlockedPatterns nil safety test ====================

func TestHandleToolApprovalSettings_NilBlockedPatterns(t *testing.T) {
	s := setupRoleTestServer(t)

	// Send policy without blocked_patterns - should default to empty slice
	policyJSON := `{
		"auto_approve_read_only": true,
		"approval_timeout_seconds": 90
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings/tool-approval", strings.NewReader(policyJSON))
	req.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()

	s.handleToolApprovalSettings(w, req)

	s.aiMu.RLock()
	if s.cfg.Authorization.ToolApproval.BlockedPatterns == nil {
		t.Error("expected BlockedPatterns to be non-nil (empty slice)")
	}
	s.aiMu.RUnlock()
}

// ==================== handleToolApprovalSettings with blocked patterns ====================

func TestHandleToolApprovalSettings_WithBlockedPatterns(t *testing.T) {
	s := setupRoleTestServer(t)

	policyJSON := `{
		"blocked_patterns": ["rm -rf", "kubectl delete ns"],
		"block_dangerous": true,
		"approval_timeout_seconds": 120
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings/tool-approval", bytes.NewBufferString(policyJSON))
	req.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()

	s.handleToolApprovalSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	s.aiMu.RLock()
	patterns := s.cfg.Authorization.ToolApproval.BlockedPatterns
	s.aiMu.RUnlock()

	if len(patterns) != 2 {
		t.Errorf("expected 2 blocked patterns, got %d", len(patterns))
	}
}
