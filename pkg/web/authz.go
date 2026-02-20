package web

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/cloudbro-kube-ai/k13d/pkg/db"
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

// Feature represents a UI/API feature that can be gated by role
type Feature string

const (
	FeatureDashboard        Feature = "dashboard"
	FeatureTopology         Feature = "topology"
	FeatureMetrics          Feature = "metrics"
	FeatureHelmManagement   Feature = "helm_management"
	FeatureSecurityScan     Feature = "security_scan"
	FeatureAIAssistant      Feature = "ai_assistant"
	FeatureTerminal         Feature = "terminal"
	FeatureReports          Feature = "reports"
	FeatureTemplates        Feature = "templates"
	FeatureEventTimeline    Feature = "event_timeline"
	FeatureAuditLogs        Feature = "audit_logs"
	FeaturePortForward      Feature = "port_forward"
	FeatureGitOps           Feature = "gitops"
	FeatureVelero           Feature = "velero"
	FeatureCostEstimate     Feature = "cost_estimate"
	FeatureNetworkPolicy    Feature = "network_policy"
	FeatureRBACViz          Feature = "rbac_viz"
	FeatureSettingsGeneral  Feature = "settings_general"
	FeatureSettingsAdmin    Feature = "settings_admin"
	FeatureSettingsSecurity Feature = "settings_security"
	FeatureSettingsNotif    Feature = "settings_notif"
)

// AllFeatures returns all defined feature constants
func AllFeatures() []Feature {
	return []Feature{
		FeatureDashboard, FeatureTopology, FeatureMetrics,
		FeatureHelmManagement, FeatureSecurityScan, FeatureAIAssistant,
		FeatureTerminal, FeatureReports, FeatureTemplates,
		FeatureEventTimeline, FeatureAuditLogs, FeaturePortForward,
		FeatureGitOps, FeatureVelero, FeatureCostEstimate,
		FeatureNetworkPolicy, FeatureRBACViz,
		FeatureSettingsGeneral, FeatureSettingsAdmin, FeatureSettingsSecurity,
		FeatureSettingsNotif,
	}
}

// ResourceRule defines permissions for a set of resources (Teleport allow/deny block)
type ResourceRule struct {
	Resources  []string `yaml:"resources" json:"resources"`   // ["pods", "deployments", "*"]
	Actions    []Action `yaml:"actions" json:"actions"`       // ["view", "scale"]
	Namespaces []string `yaml:"namespaces" json:"namespaces"` // ["default", "*"]
}

// RoleDefinition defines a role with allow and deny rules (Teleport pattern: deny overrides allow)
type RoleDefinition struct {
	Name            string         `yaml:"name" json:"name"`
	Description     string         `yaml:"description" json:"description"`
	Allow           []ResourceRule `yaml:"allow" json:"allow"`
	Deny            []ResourceRule `yaml:"deny" json:"deny"`                         // Deny always overrides Allow
	AllowedFeatures []Feature      `yaml:"allowed_features" json:"allowed_features"` // Features this role can access ("*" = all)
	DeniedFeatures  []Feature      `yaml:"denied_features" json:"denied_features"`   // Features explicitly denied (overrides allow)
	IsCustom        bool           `yaml:"is_custom" json:"is_custom"`               // True for user-created roles
}

// Authorizer manages RBAC authorization (Teleport-inspired)
type Authorizer struct {
	roles     map[string]*RoleDefinition
	featureMx sync.RWMutex
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
	// viewer: read-only access + logs, limited features
	az.roles["viewer"] = &RoleDefinition{
		Name:        "viewer",
		Description: "Read-only access with limited features",
		Allow: []ResourceRule{
			{
				Resources:  []string{"*"},
				Actions:    []Action{ActionView, ActionLogs},
				Namespaces: []string{"*"},
			},
		},
		Deny: []ResourceRule{},
		AllowedFeatures: []Feature{
			FeatureDashboard, FeatureTopology, FeatureMetrics,
			FeatureEventTimeline, FeatureAuditLogs, FeatureSettingsGeneral,
		},
	}

	// user: broad access with specific restrictions, most features except admin settings
	az.roles["user"] = &RoleDefinition{
		Name:        "user",
		Description: "Standard user with broad access",
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
		AllowedFeatures: []Feature{"*"},
		DeniedFeatures:  []Feature{FeatureSettingsAdmin, FeatureSettingsSecurity},
	}

	// admin: full access, no deny rules, all features
	az.roles["admin"] = &RoleDefinition{
		Name:        "admin",
		Description: "Full access to all resources and features",
		Allow: []ResourceRule{
			{
				Resources:  []string{"*"},
				Actions:    []Action{ActionView, ActionDelete, ActionScale, ActionRestart, ActionExec, ActionPortForward, ActionApply, ActionLogs, ActionCreate, ActionEdit},
				Namespaces: []string{"*"},
			},
		},
		Deny:            []ResourceRule{},
		AllowedFeatures: []Feature{"*"},
	}
}

// GetRole returns a role definition by name
func (az *Authorizer) GetRole(name string) *RoleDefinition {
	az.featureMx.RLock()
	defer az.featureMx.RUnlock()
	return az.roles[name]
}

// RegisterRole adds or replaces a role definition
func (az *Authorizer) RegisterRole(role *RoleDefinition) {
	az.featureMx.Lock()
	defer az.featureMx.Unlock()
	az.roles[role.Name] = role
}

// DeleteRole removes a custom role. Built-in roles (admin, user, viewer) cannot be deleted.
func (az *Authorizer) DeleteRole(name string) error {
	if name == "admin" || name == "user" || name == "viewer" {
		return fmt.Errorf("cannot delete built-in role: %s", name)
	}
	az.featureMx.Lock()
	defer az.featureMx.Unlock()
	if _, exists := az.roles[name]; !exists {
		return fmt.Errorf("role not found: %s", name)
	}
	delete(az.roles, name)
	return nil
}

// ListRoles returns all registered role definitions
func (az *Authorizer) ListRoles() []*RoleDefinition {
	az.featureMx.RLock()
	defer az.featureMx.RUnlock()
	roles := make([]*RoleDefinition, 0, len(az.roles))
	for _, r := range az.roles {
		roles = append(roles, r)
	}
	return roles
}

// IsFeatureAllowed checks if a role is allowed to access a feature.
// Deny overrides allow. "*" in AllowedFeatures means all features.
func (az *Authorizer) IsFeatureAllowed(role string, feature Feature) bool {
	az.featureMx.RLock()
	defer az.featureMx.RUnlock()

	roleDef, exists := az.roles[role]
	if !exists {
		return false
	}

	// Check denied features first (deny overrides allow)
	for _, df := range roleDef.DeniedFeatures {
		if df == feature || df == "*" {
			return false
		}
	}

	// Check allowed features
	for _, af := range roleDef.AllowedFeatures {
		if af == "*" || af == feature {
			return true
		}
	}

	return false
}

// GetFeaturePermissions returns a map of all features to their allowed/denied status for a role
func (az *Authorizer) GetFeaturePermissions(role string) map[Feature]bool {
	perms := make(map[Feature]bool)
	for _, f := range AllFeatures() {
		perms[f] = az.IsFeatureAllowed(role, f)
	}
	return perms
}

// FeatureMiddleware creates HTTP middleware that checks feature-level access
func (az *Authorizer) FeatureMiddleware(feature Feature) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			role := r.Header.Get("X-User-Role")
			if role == "" {
				role = "viewer"
			}

			if !az.IsFeatureAllowed(role, feature) {
				username := r.Header.Get("X-Username")
				_ = db.RecordAudit(db.AuditEntry{
					User:       username,
					Action:     "feature_denied",
					Resource:   string(feature),
					Details:    fmt.Sprintf("Role %s denied access to feature %s", role, feature),
					ActionType: db.ActionTypeAuthzDenied,
					Source:     "web",
					Success:    false,
				})
				http.Error(w, fmt.Sprintf("Forbidden: role %s does not have access to feature %s", role, feature), http.StatusForbidden)
				return
			}

			next(w, r)
		}
	}
}

// IsAllowed checks if a role is allowed to perform an action on a resource in a namespace
// Returns (allowed, reason) - deny always overrides allow (Teleport pattern)
func (az *Authorizer) IsAllowed(role, resource string, action Action, namespace string) (bool, string) {
	az.featureMx.RLock()
	roleDef, exists := az.roles[role]
	az.featureMx.RUnlock()
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
				_ = db.RecordAudit(db.AuditEntry{
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
