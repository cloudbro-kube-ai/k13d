package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorizer_DefaultRolesBackwardCompat(t *testing.T) {
	az := NewAuthorizer()

	// Admin should have full access
	tests := []struct {
		name      string
		role      string
		resource  string
		action    Action
		namespace string
		allowed   bool
	}{
		// Admin: full access
		{"admin view pods", "admin", "pods", ActionView, "default", true},
		{"admin delete pods", "admin", "pods", ActionDelete, "default", true},
		{"admin exec pods", "admin", "pods", ActionExec, "kube-system", true},
		{"admin scale deployments", "admin", "deployments", ActionScale, "production", true},
		{"admin delete nodes", "admin", "nodes", ActionDelete, "default", true},

		// Viewer: view + logs only
		{"viewer view pods", "viewer", "pods", ActionView, "default", true},
		{"viewer logs pods", "viewer", "pods", ActionLogs, "default", true},
		{"viewer delete pods", "viewer", "pods", ActionDelete, "default", false},
		{"viewer scale deploy", "viewer", "deployments", ActionScale, "default", false},
		{"viewer exec pods", "viewer", "pods", ActionExec, "default", false},

		// User: broad access with restrictions
		{"user view pods", "user", "pods", ActionView, "default", true},
		{"user scale deploy", "user", "deployments", ActionScale, "default", true},
		{"user restart deploy", "user", "deployments", ActionRestart, "default", true},
		{"user apply resources", "user", "deployments", ActionApply, "default", true},
		{"user port-forward", "user", "pods", ActionPortForward, "default", true},
		// User deny rules
		{"user exec kube-system", "user", "pods", ActionExec, "kube-system", false},
		{"user delete nodes", "user", "nodes", ActionDelete, "default", false},
		{"user delete namespaces", "user", "namespaces", ActionDelete, "default", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, reason := az.IsAllowed(tt.role, tt.resource, tt.action, tt.namespace)
			if allowed != tt.allowed {
				t.Errorf("IsAllowed(%q, %q, %q, %q) = %v (%s), want %v",
					tt.role, tt.resource, tt.action, tt.namespace, allowed, reason, tt.allowed)
			}
		})
	}
}

func TestAuthorizer_DenyOverridesAllow(t *testing.T) {
	az := &Authorizer{
		roles: map[string]*RoleDefinition{
			"restricted": {
				Name: "restricted",
				Allow: []ResourceRule{
					{
						Resources:  []string{"*"},
						Actions:    []Action{ActionView, ActionDelete},
						Namespaces: []string{"*"},
					},
				},
				Deny: []ResourceRule{
					{
						Resources:  []string{"pods"},
						Actions:    []Action{ActionDelete},
						Namespaces: []string{"production"},
					},
				},
			},
		},
	}

	// Delete pods in default should work
	allowed, _ := az.IsAllowed("restricted", "pods", ActionDelete, "default")
	if !allowed {
		t.Error("Expected delete pods in default to be allowed")
	}

	// Delete pods in production should be denied (deny overrides allow)
	allowed, reason := az.IsAllowed("restricted", "pods", ActionDelete, "production")
	if allowed {
		t.Errorf("Expected delete pods in production to be denied, got reason: %s", reason)
	}

	// View pods in production should still work (deny only applies to delete)
	allowed, _ = az.IsAllowed("restricted", "pods", ActionView, "production")
	if !allowed {
		t.Error("Expected view pods in production to be allowed")
	}
}

func TestAuthorizer_WildcardMatching(t *testing.T) {
	az := NewAuthorizer()

	// Admin with wildcard resources should match anything
	allowed, _ := az.IsAllowed("admin", "customresource", ActionView, "default")
	if !allowed {
		t.Error("Expected wildcard resource matching")
	}

	// Test wildcard namespace
	allowed, _ = az.IsAllowed("admin", "pods", ActionDelete, "any-namespace")
	if !allowed {
		t.Error("Expected wildcard namespace matching")
	}
}

func TestAuthorizer_NamespaceScope(t *testing.T) {
	az := &Authorizer{
		roles: map[string]*RoleDefinition{
			"dev": {
				Name: "dev",
				Allow: []ResourceRule{
					{
						Resources:  []string{"pods", "deployments"},
						Actions:    []Action{ActionView, ActionScale},
						Namespaces: []string{"dev-*", "staging"},
					},
				},
			},
		},
	}

	tests := []struct {
		name      string
		namespace string
		allowed   bool
	}{
		{"dev namespace prefix", "dev-team1", true},
		{"dev namespace prefix 2", "dev-frontend", true},
		{"staging exact", "staging", true},
		{"production denied", "production", false},
		{"default denied", "default", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, _ := az.IsAllowed("dev", "pods", ActionView, tt.namespace)
			if allowed != tt.allowed {
				t.Errorf("namespace %q: got %v, want %v", tt.namespace, allowed, tt.allowed)
			}
		})
	}
}

func TestAuthorizer_UnknownRole(t *testing.T) {
	az := NewAuthorizer()

	allowed, reason := az.IsAllowed("nonexistent", "pods", ActionView, "default")
	if allowed {
		t.Error("Expected unknown role to be denied")
	}
	if reason == "" {
		t.Error("Expected reason for denial")
	}
}

func TestAuthorizer_ClusterScopedResource(t *testing.T) {
	az := NewAuthorizer()

	// Cluster-scoped resources have empty namespace
	allowed, _ := az.IsAllowed("admin", "nodes", ActionView, "")
	if !allowed {
		t.Error("Expected admin to view cluster-scoped resources")
	}

	allowed, _ = az.IsAllowed("viewer", "nodes", ActionView, "")
	if !allowed {
		t.Error("Expected viewer to view cluster-scoped resources")
	}
}

func TestAuthorizer_CustomRoles(t *testing.T) {
	customRoles := []RoleDefinition{
		{
			Name: "developer",
			Allow: []ResourceRule{
				{
					Resources:  []string{"pods", "deployments"},
					Actions:    []Action{ActionView, ActionLogs, ActionScale},
					Namespaces: []string{"staging", "dev-*"},
				},
			},
			Deny: []ResourceRule{
				{
					Resources:  []string{"*"},
					Actions:    []Action{ActionDelete},
					Namespaces: []string{"*"},
				},
			},
		},
	}

	az := NewAuthorizerWithRoles(customRoles)

	// Custom role should override
	allowed, _ := az.IsAllowed("developer", "pods", ActionView, "staging")
	if !allowed {
		t.Error("Expected developer to view pods in staging")
	}

	allowed, _ = az.IsAllowed("developer", "pods", ActionDelete, "staging")
	if allowed {
		t.Error("Expected developer to be denied delete in staging")
	}

	// Built-in roles should still work
	allowed, _ = az.IsAllowed("admin", "pods", ActionDelete, "staging")
	if !allowed {
		t.Error("Expected admin to still have full access")
	}
}

func TestAuthzMiddleware_BlocksUnauthorized(t *testing.T) {
	az := NewAuthorizer()

	handler := az.AuthzMiddleware("deployments", ActionScale)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Viewer should be blocked
	req := httptest.NewRequest("POST", "/api/deployment/scale", nil)
	req.Header.Set("X-User-Role", "viewer")
	req.Header.Set("X-Username", "test-viewer")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for viewer scaling, got %d", rec.Code)
	}
}

func TestAuthzMiddleware_AllowsAuthorized(t *testing.T) {
	az := NewAuthorizer()

	handler := az.AuthzMiddleware("deployments", ActionScale)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Admin should be allowed
	req := httptest.NewRequest("POST", "/api/deployment/scale", nil)
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-Username", "test-admin")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 for admin scaling, got %d", rec.Code)
	}
}

func TestAuthzMiddleware_DefaultsToViewer(t *testing.T) {
	az := NewAuthorizer()

	handler := az.AuthzMiddleware("deployments", ActionScale)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// No role should default to viewer (denied for scale)
	req := httptest.NewRequest("POST", "/api/deployment/scale", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for no-role scaling, got %d", rec.Code)
	}
}
