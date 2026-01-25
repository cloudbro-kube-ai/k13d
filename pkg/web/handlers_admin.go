package web

import (
	"net/http"
)

// ==========================================
// Admin Handlers
// ==========================================

// handleAdminUsers handles listing users and creating new users
func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.authManager.HandleListUsers(w, r)
	case http.MethodPost:
		s.authManager.HandleCreateUser(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAdminUserAction handles individual user operations (update/delete)
func (s *Server) handleAdminUserAction(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut, http.MethodPatch:
		s.authManager.HandleUpdateUser(w, r)
	case http.MethodDelete:
		s.authManager.HandleDeleteUser(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
