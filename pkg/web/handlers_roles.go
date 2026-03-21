package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cloudbro-kube-ai/k13d/pkg/db"
)

// handleRoles handles GET (list all roles) and POST (create custom role)
func (s *Server) handleRoles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListRoles(w, r)
	case http.MethodPost:
		s.handleCreateRole(w, r)
	default:
		writeMethodNotAllowed(w)
	}
}

// handleListRoles returns all registered roles (built-in + custom)
func (s *Server) handleListRoles(w http.ResponseWriter, r *http.Request) {
	roles := s.authorizer.ListRoles()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"roles": roles,
		"total": len(roles),
	})
}

// roleRequest represents a role creation/update request
type roleRequest struct {
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	Allow           []ResourceRule `json:"allow"`
	Deny            []ResourceRule `json:"deny"`
	AllowedFeatures []Feature      `json:"allowed_features"`
	DeniedFeatures  []Feature      `json:"denied_features"`
}

// handleCreateRole creates a new custom role
func (s *Server) handleCreateRole(w http.ResponseWriter, r *http.Request) {
	var req roleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	if req.Name == "" {
		WriteError(w, NewAPIError(ErrCodeValidation, "Role name is required"))
		return
	}

	// Prevent overwriting built-in roles
	if req.Name == "admin" || req.Name == "user" || req.Name == "viewer" {
		WriteError(w, NewAPIError(ErrCodeConflict, "Cannot create role with built-in name: "+req.Name))
		return
	}

	role := &RoleDefinition{
		Name:            req.Name,
		Description:     req.Description,
		Allow:           req.Allow,
		Deny:            req.Deny,
		AllowedFeatures: req.AllowedFeatures,
		DeniedFeatures:  req.DeniedFeatures,
		IsCustom:        true,
	}

	// Persist to database
	defJSON, err := json.Marshal(role)
	if err != nil {
		WriteError(w, NewAPIError(ErrCodeInternalError, "Failed to serialize role"))
		return
	}
	if err := db.SaveCustomRole(req.Name, string(defJSON)); err != nil {
		WriteError(w, NewAPIError(ErrCodeInternalError, "Failed to save role: "+err.Error()))
		return
	}

	// Register in memory
	s.authorizer.RegisterRole(role)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "created",
		"name":   req.Name,
	})
}

// handleRoleByName handles PUT (update) and DELETE for a specific role
func (s *Server) handleRoleByName(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/roles/")
	if name == "" {
		WriteError(w, NewAPIError(ErrCodeValidation, "Role name is required"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetRole(w, r, name)
	case http.MethodPut:
		s.handleUpdateRole(w, r, name)
	case http.MethodDelete:
		s.handleDeleteRole(w, r, name)
	default:
		writeMethodNotAllowed(w)
	}
}

// handleGetRole returns a single role definition
func (s *Server) handleGetRole(w http.ResponseWriter, _ *http.Request, name string) {
	role := s.authorizer.GetRole(name)
	if role == nil {
		WriteError(w, NewAPIError(ErrCodeNotFound, "Role not found: "+name))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(role)
}

// handleUpdateRole updates a custom role (built-in roles cannot be modified)
func (s *Server) handleUpdateRole(w http.ResponseWriter, r *http.Request, name string) {
	if name == "admin" || name == "user" || name == "viewer" {
		WriteError(w, NewAPIError(ErrCodeForbidden, "Cannot modify built-in role: "+name))
		return
	}

	var req roleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	role := &RoleDefinition{
		Name:            name,
		Description:     req.Description,
		Allow:           req.Allow,
		Deny:            req.Deny,
		AllowedFeatures: req.AllowedFeatures,
		DeniedFeatures:  req.DeniedFeatures,
		IsCustom:        true,
	}

	defJSON, err := json.Marshal(role)
	if err != nil {
		WriteError(w, NewAPIError(ErrCodeInternalError, "Failed to serialize role"))
		return
	}
	if err := db.SaveCustomRole(name, string(defJSON)); err != nil {
		WriteError(w, NewAPIError(ErrCodeInternalError, "Failed to save role: "+err.Error()))
		return
	}

	s.authorizer.RegisterRole(role)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "updated",
		"name":   name,
	})
}

// handleDeleteRole removes a custom role
func (s *Server) handleDeleteRole(w http.ResponseWriter, _ *http.Request, name string) {
	if err := s.authorizer.DeleteRole(name); err != nil {
		if strings.Contains(err.Error(), "built-in") {
			WriteError(w, NewAPIError(ErrCodeForbidden, err.Error()))
		} else {
			WriteError(w, NewAPIError(ErrCodeNotFound, err.Error()))
		}
		return
	}

	// Remove from database
	if err := db.DeleteCustomRole(name); err != nil {
		// Already removed from memory; log but don't fail
		_ = err
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "deleted",
		"name":   name,
	})
}

// handleUserPermissions returns the feature permissions for the current user
func (s *Server) handleUserPermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	role := r.Header.Get("X-User-Role")
	if role == "" {
		role = "viewer"
	}

	perms := s.authorizer.GetFeaturePermissions(role)

	// Convert Feature keys to strings for JSON
	featureMap := make(map[string]bool, len(perms))
	for f, allowed := range perms {
		featureMap[string(f)] = allowed
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"role":     role,
		"features": featureMap,
	})
}
