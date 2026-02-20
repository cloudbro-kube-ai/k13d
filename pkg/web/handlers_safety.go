package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/db"
)

// handleToolApprovalSettings handles GET/PUT for tool approval policy settings.
// GET returns the current policy; PUT (admin-only) updates it.
func (s *Server) handleToolApprovalSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.aiMu.RLock()
		policy := s.cfg.Authorization.ToolApproval
		s.aiMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(policy)

	case http.MethodPut:
		// Check admin role
		role := r.Header.Get("X-User-Role")
		if role != "admin" {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}

		var policy config.ToolApprovalPolicy
		if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate timeout bounds
		if policy.ApprovalTimeoutSeconds <= 0 {
			policy.ApprovalTimeoutSeconds = 60
		}
		if policy.ApprovalTimeoutSeconds > 600 {
			policy.ApprovalTimeoutSeconds = 600
		}

		// Ensure BlockedPatterns is never nil
		if policy.BlockedPatterns == nil {
			policy.BlockedPatterns = []string{}
		}

		// Update in-memory config
		s.aiMu.Lock()
		s.cfg.Authorization.ToolApproval = policy
		s.aiMu.Unlock()

		// Save to config file
		if err := s.cfg.Save(); err != nil {
			fmt.Printf("Warning: failed to save tool approval settings to config: %v\n", err)
		}

		// Persist to web_settings DB for reload on restart
		if err := db.SaveWebSettings(map[string]string{
			"tool_approval.auto_approve_read_only":       fmt.Sprintf("%v", policy.AutoApproveReadOnly),
			"tool_approval.require_approval_for_write":   fmt.Sprintf("%v", policy.RequireApprovalForWrite),
			"tool_approval.require_approval_for_unknown": fmt.Sprintf("%v", policy.RequireApprovalForUnknown),
			"tool_approval.block_dangerous":              fmt.Sprintf("%v", policy.BlockDangerous),
			"tool_approval.approval_timeout_seconds":     fmt.Sprintf("%d", policy.ApprovalTimeoutSeconds),
			"tool_approval.blocked_patterns":             strings.Join(policy.BlockedPatterns, "\n"),
		}); err != nil {
			fmt.Printf("Warning: failed to save tool approval settings to SQLite: %v\n", err)
		}

		// Record audit
		username := r.Header.Get("X-Username")
		_ = db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "update_tool_approval_settings",
			Resource: "settings",
			Details:  fmt.Sprintf("AutoApproveReadOnly=%v, BlockDangerous=%v, Timeout=%ds", policy.AutoApproveReadOnly, policy.BlockDangerous, policy.ApprovalTimeoutSeconds),
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "saved"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getToolApprovalTimeout returns the configured tool approval timeout duration.
// Used by handleAgenticChat to replace the hardcoded 60s timeout.
func (s *Server) getToolApprovalTimeout() time.Duration {
	s.aiMu.RLock()
	seconds := s.cfg.Authorization.ToolApproval.ApprovalTimeoutSeconds
	s.aiMu.RUnlock()

	if seconds <= 0 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}
