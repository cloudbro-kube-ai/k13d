package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"
)

// Action represents a resource action type for RBAC (Teleport-inspired)
type Action string

const (
	ActionView        Action = "view"
	ActionDelete      Action = "delete"
	ActionScale       Action = "scale"
	ActionRestart     Action = "restart"
	ActionExec        Action = "exec"
	ActionPortForward Action = "port-forward"
	ActionApply       Action = "apply"
	ActionLogs        Action = "logs"
	ActionCreate      Action = "create"
	ActionEdit        Action = "edit"
)

// ResourceRule defines permissions for a set of resources (Teleport allow/deny block)
type ResourceRule struct {
	Resources  []string `yaml:"resources" json:"resources"`   // ["pods", "deployments", "*"]
	Actions    []Action `yaml:"actions" json:"actions"`       // ["view", "scale"]
	Namespaces []string `yaml:"namespaces" json:"namespaces"` // ["default", "*"]
}

// RoleDefinition defines a role with allow and deny rules (Teleport pattern: deny overrides allow)
type RoleDefinition struct {
	Name  string         `yaml:"name" json:"name"`
	Allow []ResourceRule `yaml:"allow" json:"allow"`
	Deny  []ResourceRule `yaml:"deny" json:"deny"` // Deny always overrides Allow
}

// Authorizer manages RBAC authorization (Teleport-inspired)
type Authorizer struct {
	roles map[string]*RoleDefinition
}

// NewAuthorizer creates a new Authorizer with default roles
func NewAuthorizer() *Authorizer {
	az := &Authorizer{
		roles: make(map[string]*RoleDefinition),
	}
	az.registerDefaultRoles()
	return az
}

// NewAuthorizerWithRoles creates an Authorizer with custom roles merged with defaults
func NewAuthorizerWithRoles(customRoles []RoleDefinition) *Authorizer {
	az := NewAuthorizer()
	for i := range customRoles {
		az.roles[customRoles[i].Name] = &customRoles[i]
	}
	return az
}

// registerDefaultRoles sets up the three default roles (backward-compatible)
func (az *Authorizer) registerDefaultRoles() {
	// viewer: read-only access + logs
	az.roles["viewer"] = &RoleDefinition{
		Name: "viewer",
		Allow: []ResourceRule{
			{
				Resources:  []string{"*"},
				Actions:    []Action{ActionView, ActionLogs},
				Namespaces: []string{"*"},
			},
		},
		Deny: []ResourceRule{},
	}

	// user: broad access with specific restrictions
	az.roles["user"] = &RoleDefinition{
		Name: "user",
		Allow: []ResourceRule{
			{
				Resources:  []string{"*"},
				Actions:    []Action{ActionView, ActionLogs, ActionScale, ActionRestart, ActionCreate, ActionApply, ActionEdit, ActionPortForward},
				Namespaces: []string{"*"},
			},
		},
		Deny: []ResourceRule{
			// Deny exec in kube-system
			{
				Resources:  []string{"*"},
				Actions:    []Action{ActionExec},
				Namespaces: []string{"kube-system"},
			},
			// Deny delete on nodes and namespaces
			{
				Resources:  []string{"nodes", "namespaces"},
				Actions:    []Action{ActionDelete},
				Namespaces: []string{"*"},
			},
		},
	}

	// admin: full access, no deny rules
	az.roles["admin"] = &RoleDefinition{
		Name: "admin",
		Allow: []ResourceRule{
			{
				Resources:  []string{"*"},
				Actions:    []Action{ActionView, ActionDelete, ActionScale, ActionRestart, ActionExec, ActionPortForward, ActionApply, ActionLogs, ActionCreate, ActionEdit},
				Namespaces: []string{"*"},
			},
		},
		Deny: []ResourceRule{},
	}
}

// GetRole returns a role definition by name
func (az *Authorizer) GetRole(name string) *RoleDefinition {
	return az.roles[name]
}

// IsAllowed checks if a role is allowed to perform an action on a resource in a namespace
// Returns (allowed, reason) - deny always overrides allow (Teleport pattern)
func (az *Authorizer) IsAllowed(role, resource string, action Action, namespace string) (bool, string) {
	roleDef, exists := az.roles[role]
	if !exists {
		return false, fmt.Sprintf("unknown role: %s", role)
	}

	// Step 1: Check deny rules first (deny always wins)
	for _, denyRule := range roleDef.Deny {
		if matchesRule(denyRule, resource, action, namespace) {
			return false, fmt.Sprintf("denied by deny rule: %s cannot %s %s in %s",
				role, action, resource, namespace)
		}
	}

	// Step 2: Check allow rules
	for _, allowRule := range roleDef.Allow {
		if matchesRule(allowRule, resource, action, namespace) {
			return true, "allowed"
		}
	}

	// Step 3: Default deny (no matching allow rule)
	return false, fmt.Sprintf("no allow rule for: %s to %s %s in %s",
		role, action, resource, namespace)
}

// matchesRule checks if a rule matches the given resource, action, and namespace
func matchesRule(rule ResourceRule, resource string, action Action, namespace string) bool {
	return matchesResources(rule.Resources, resource) &&
		matchesActions(rule.Actions, action) &&
		matchesNamespaces(rule.Namespaces, namespace)
}

// matchesResources checks if a resource matches a list of resource patterns
func matchesResources(patterns []string, resource string) bool {
	resource = strings.ToLower(resource)
	for _, pattern := range patterns {
		if pattern == "*" {
			return true
		}
		if matchesPattern(strings.ToLower(pattern), resource) {
			return true
		}
	}
	return false
}

// matchesActions checks if an action is in the allowed list
func matchesActions(actions []Action, target Action) bool {
	for _, a := range actions {
		if a == target || a == "*" {
			return true
		}
	}
	return false
}

// matchesNamespaces checks if a namespace matches a list of namespace patterns
func matchesNamespaces(patterns []string, namespace string) bool {
	if namespace == "" {
		// Cluster-scoped resources have no namespace; treat as matching any
		return true
	}
	namespace = strings.ToLower(namespace)
	for _, pattern := range patterns {
		if pattern == "*" {
			return true
		}
		if matchesPattern(strings.ToLower(pattern), namespace) {
			return true
		}
	}
	return false
}

// matchesPattern checks if a string matches a simple wildcard pattern
// Supports: "*" (match all), "dev-*" (prefix match), exact match
func matchesPattern(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(value, prefix)
	}
	return pattern == value
}

// AuthzMiddleware creates an HTTP middleware that checks RBAC authorization
// It extracts the user role from the X-User-Role header (set by AuthMiddleware)
func (az *Authorizer) AuthzMiddleware(resource string, action Action) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			role := r.Header.Get("X-User-Role")
			username := r.Header.Get("X-Username")

			// If no role is set, default to viewer (most restrictive)
			if role == "" {
				role = "viewer"
			}

			// Extract namespace from request (query param or body)
			namespace := r.URL.Query().Get("namespace")
			if namespace == "" {
				namespace = r.URL.Query().Get("ns")
			}

			allowed, reason := az.IsAllowed(role, resource, action, namespace)
			if !allowed {
				// Record authorization denial in audit log
				db.RecordAudit(db.AuditEntry{
					User:            username,
					Action:          "authz_denied",
					Resource:        resource,
					Details:         reason,
					ActionType:      db.ActionTypeAuthzDenied,
					Source:          "web",
					ClientIP:        r.RemoteAddr,
					Success:         false,
					ErrorMsg:        reason,
					RequestedAction: string(action),
					TargetResource:  resource,
					TargetNamespace: namespace,
					AuthzDecision:   "denied",
				})

				http.Error(w, fmt.Sprintf("Forbidden: %s", reason), http.StatusForbidden)
				return
			}

			next(w, r)
		}
	}
}
