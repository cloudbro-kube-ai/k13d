package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/db"
	"github.com/cloudbro-kube-ai/k13d/pkg/i18n"
	"github.com/cloudbro-kube-ai/k13d/pkg/log"
)

// ==========================================
// Settings Handlers
// ==========================================

func (s *Server) handleAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse filter parameters
	filter := db.AuditFilter{
		Limit: 200, // Default limit
	}

	// Parse query parameters
	if r.URL.Query().Get("only_llm") == "true" {
		filter.OnlyLLM = true
	}
	if r.URL.Query().Get("only_errors") == "true" {
		filter.OnlyErrors = true
	}
	if user := r.URL.Query().Get("user"); user != "" {
		filter.User = user
	}
	if k8sUser := r.URL.Query().Get("k8s_user"); k8sUser != "" {
		filter.K8sUser = k8sUser
	}
	if action := r.URL.Query().Get("action"); action != "" {
		filter.Action = action
	}
	if resource := r.URL.Query().Get("resource"); resource != "" {
		filter.Resource = resource
	}
	if source := r.URL.Query().Get("source"); source != "" {
		filter.Source = source
	}

	logs, err := db.GetAuditLogsFiltered(filter)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"logs":  []interface{}{},
		})
		return
	}

	if logs == nil {
		logs = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":      logs,
		"timestamp": time.Now(),
	})
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// Load timezone from SQLite (web-only setting)
		timezone := db.GetWebSettingWithDefault("general.timezone", "auto")

		// Return current settings (without sensitive data)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"language":      s.cfg.Language,
			"beginner_mode": s.cfg.BeginnerMode,
			"enable_audit":  s.cfg.EnableAudit,
			"log_level":     s.cfg.LogLevel,
			"timezone":      timezone,
			"llm": map[string]interface{}{
				"provider":         s.cfg.LLM.Provider,
				"model":            s.cfg.LLM.Model,
				"endpoint":         s.cfg.LLM.Endpoint,
				"reasoning_effort": s.cfg.LLM.ReasoningEffort,
			},
		})

	case http.MethodPut:
		var newSettings struct {
			Language     string `json:"language"`
			BeginnerMode bool   `json:"beginner_mode"`
			EnableAudit  bool   `json:"enable_audit"`
			LogLevel     string `json:"log_level"`
			Timezone     string `json:"timezone"`
		}

		if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
			WriteErrorSimple(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Update settings (protected by mutex for concurrent access)
		s.aiMu.Lock()
		s.cfg.Language = newSettings.Language
		s.cfg.BeginnerMode = newSettings.BeginnerMode
		s.cfg.EnableAudit = newSettings.EnableAudit
		s.cfg.LogLevel = newSettings.LogLevel
		s.aiMu.Unlock()

		// Apply language change to i18n system
		i18n.SetLanguage(newSettings.Language)

		// Save to YAML
		if err := s.cfg.Save(); err != nil {
			WriteErrorSimple(w, http.StatusInternalServerError, "Failed to save settings")
			return
		}

		// Also persist to SQLite for web UI settings
		dbSettings := map[string]string{
			"general.language":  newSettings.Language,
			"general.log_level": newSettings.LogLevel,
		}
		if newSettings.Timezone != "" {
			dbSettings["general.timezone"] = newSettings.Timezone
		}
		if err := db.SaveWebSettings(dbSettings); err != nil {
			log.Warnf("Failed to save settings to SQLite: %v", err)
		}

		// Record audit
		username := r.Header.Get("X-Username")
		_ = db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "update_settings",
			Resource: "settings",
			Details:  fmt.Sprintf("Settings updated (timezone: %s)", newSettings.Timezone),
		})

		_ = json.NewEncoder(w).Encode(map[string]string{"status": "saved"})

	default:
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// ==========================================
// Model Management Handlers
// ==========================================

// handleModels manages LLM model profiles
func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// Return all model profiles (mask API keys)
		models := make([]map[string]interface{}, len(s.cfg.Models))
		for i, m := range s.cfg.Models {
			models[i] = map[string]interface{}{
				"name":            m.Name,
				"provider":        m.Provider,
				"model":           m.Model,
				"endpoint":        m.Endpoint,
				"description":     m.Description,
				"has_api_key":     m.APIKey != "",
				"is_active":       m.Name == s.cfg.ActiveModel,
				"skip_tls_verify": m.SkipTLSVerify,
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"models":       models,
			"active_model": s.cfg.ActiveModel,
		})

	case http.MethodPost:
		// Add new model profile
		var profile config.ModelProfile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			WriteErrorSimple(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if profile.Name == "" || profile.Provider == "" || profile.Model == "" {
			WriteErrorSimple(w, http.StatusBadRequest, "Name, provider, and model are required")
			return
		}

		s.cfg.AddModelProfile(profile)
		if err := s.cfg.Save(); err != nil {
			WriteErrorSimple(w, http.StatusInternalServerError, "Failed to save config")
			return
		}

		// Record audit
		username := r.Header.Get("X-Username")
		_ = db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "add_model_profile",
			Resource: "model",
			Details:  fmt.Sprintf("Added model profile: %s (%s/%s)", profile.Name, profile.Provider, profile.Model),
		})

		_ = json.NewEncoder(w).Encode(map[string]string{"status": "created", "name": profile.Name})

	case http.MethodDelete:
		// Delete model profile
		name := r.URL.Query().Get("name")
		if name == "" {
			WriteErrorSimple(w, http.StatusBadRequest, "Model name required")
			return
		}

		if !s.cfg.RemoveModelProfile(name) {
			WriteErrorSimple(w, http.StatusNotFound, "Model not found")
			return
		}

		if err := s.cfg.Save(); err != nil {
			WriteErrorSimple(w, http.StatusInternalServerError, "Failed to save config")
			return
		}

		// Sync deletion to SQLite
		if err := db.DeleteModelProfile(name); err != nil {
			log.Warnf("Failed to delete model profile from SQLite: %v", err)
		}

		// Record audit
		username := r.Header.Get("X-Username")
		_ = db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "delete_model_profile",
			Resource: "model",
			Details:  fmt.Sprintf("Deleted model profile: %s", name),
		})

		_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleActiveModel switches the active LLM model
func (s *Server) handleActiveModel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		profile := s.cfg.GetActiveModelProfile()
		if profile == nil {
			WriteErrorSimple(w, http.StatusNotFound, "No active model")
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":        profile.Name,
			"provider":    profile.Provider,
			"model":       profile.Model,
			"description": profile.Description,
		})

	case http.MethodPut:
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteErrorSimple(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		s.aiMu.Lock()
		if !s.cfg.SetActiveModel(req.Name) {
			s.aiMu.Unlock()
			WriteErrorSimple(w, http.StatusNotFound, "Model not found")
			return
		}

		// Recreate AI client with new model
		newClient, err := ai.NewClient(&s.cfg.LLM)
		if err != nil {
			s.aiMu.Unlock()
			WriteErrorSimple(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create AI client: %v", err))
			return
		}
		s.aiClient = newClient

		// Capture config values under lock before releasing
		activeModel := s.cfg.ActiveModel
		llmProvider := s.cfg.LLM.Provider
		llmModel := s.cfg.LLM.Model
		llmEndpoint := s.cfg.LLM.Endpoint
		llmAPIKey := s.cfg.LLM.APIKey
		s.aiMu.Unlock()

		// Re-register MCP tools
		for _, serverName := range s.mcpClient.GetConnectedServers() {
			s.registerMCPTools(serverName)
		}

		if err := s.cfg.Save(); err != nil {
			WriteErrorSimple(w, http.StatusInternalServerError, "Failed to save config")
			return
		}

		// Also persist LLM settings to SQLite so DB stays in sync after restart
		llmDBSettings := map[string]string{
			"llm.active_model": activeModel,
			"llm.provider":     llmProvider,
			"llm.model":        llmModel,
			"llm.endpoint":     llmEndpoint,
		}
		if llmAPIKey != "" {
			llmDBSettings["llm.api_key"] = llmAPIKey
		}
		if err := db.SaveWebSettings(llmDBSettings); err != nil {
			log.Warnf("Failed to save model switch to SQLite: %v", err)
		}
		// Update active flag in model_profiles table
		if err := db.SetActiveModelProfile(req.Name); err != nil {
			log.Warnf("Failed to update active model profile in SQLite: %v", err)
		}

		// Record audit
		username := r.Header.Get("X-Username")
		_ = db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "switch_model",
			Resource: "model",
			Details:  fmt.Sprintf("Switched to model: %s", req.Name),
		})

		_ = json.NewEncoder(w).Encode(map[string]string{"status": "switched", "active_model": req.Name})

	default:
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// ==========================================
// MCP Server Management Handlers
// ==========================================

// handleMCPServers manages MCP server configurations
func (s *Server) handleMCPServers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// Return all MCP server configurations with status
		servers := make([]map[string]interface{}, len(s.cfg.MCP.Servers))
		for i, srv := range s.cfg.MCP.Servers {
			servers[i] = map[string]interface{}{
				"name":        srv.Name,
				"command":     srv.Command,
				"args":        srv.Args,
				"description": srv.Description,
				"enabled":     srv.Enabled,
				"connected":   s.mcpClient.IsConnected(srv.Name),
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"servers":   servers,
			"connected": s.mcpClient.GetConnectedServers(),
		})

	case http.MethodPost:
		// Add new MCP server
		var server config.MCPServer
		if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
			WriteErrorSimple(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if server.Name == "" || server.Command == "" {
			WriteErrorSimple(w, http.StatusBadRequest, "Name and command are required")
			return
		}

		s.cfg.AddMCPServer(server)
		if err := s.cfg.Save(); err != nil {
			WriteErrorSimple(w, http.StatusInternalServerError, "Failed to save config")
			return
		}

		// If enabled, try to connect
		if server.Enabled {
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()
			if err := s.mcpClient.Connect(ctx, server); err != nil {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  "created",
					"name":    server.Name,
					"warning": fmt.Sprintf("Server added but failed to connect: %v", err),
				})
				return
			}
			s.registerMCPTools(server.Name)
		}

		// Record audit
		username := r.Header.Get("X-Username")
		_ = db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "add_mcp_server",
			Resource: "mcp",
			Details:  fmt.Sprintf("Added MCP server: %s (%s)", server.Name, server.Command),
		})

		_ = json.NewEncoder(w).Encode(map[string]string{"status": "created", "name": server.Name})

	case http.MethodPut:
		// Toggle MCP server enabled/disabled or reconnect
		var req struct {
			Name   string `json:"name"`
			Action string `json:"action"` // "enable", "disable", "reconnect"
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteErrorSimple(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		switch req.Action {
		case "enable":
			if !s.cfg.ToggleMCPServer(req.Name, true) {
				WriteErrorSimple(w, http.StatusNotFound, "Server not found")
				return
			}
			// Try to connect
			for _, srv := range s.cfg.MCP.Servers {
				if srv.Name == req.Name {
					ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
					if err := s.mcpClient.Connect(ctx, srv); err != nil {
						cancel()
						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"status":  "enabled",
							"warning": fmt.Sprintf("Enabled but failed to connect: %v", err),
						})
						_ = s.cfg.Save()
						return
					}
					cancel()
					s.registerMCPTools(srv.Name)
					break
				}
			}

		case "disable":
			if !s.cfg.ToggleMCPServer(req.Name, false) {
				WriteErrorSimple(w, http.StatusNotFound, "Server not found")
				return
			}
			// Disconnect and unregister tools
			_ = s.mcpClient.Disconnect(req.Name)
			if s.aiClient != nil {
				s.aiClient.GetToolRegistry().UnregisterMCPTools(req.Name)
			}

		case "reconnect":
			// Disconnect first
			_ = s.mcpClient.Disconnect(req.Name)
			if s.aiClient != nil {
				s.aiClient.GetToolRegistry().UnregisterMCPTools(req.Name)
			}
			// Reconnect
			for _, srv := range s.cfg.MCP.Servers {
				if srv.Name == req.Name && srv.Enabled {
					ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
					if err := s.mcpClient.Connect(ctx, srv); err != nil {
						cancel()
						WriteErrorSimple(w, http.StatusInternalServerError, fmt.Sprintf("Failed to reconnect: %v", err))
						return
					}
					cancel()
					s.registerMCPTools(srv.Name)
					break
				}
			}

		default:
			WriteErrorSimple(w, http.StatusBadRequest, "Invalid action (use: enable, disable, reconnect)")
			return
		}

		if err := s.cfg.Save(); err != nil {
			WriteErrorSimple(w, http.StatusInternalServerError, "Failed to save config")
			return
		}

		// Record audit
		username := r.Header.Get("X-Username")
		_ = db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   fmt.Sprintf("mcp_server_%s", req.Action),
			Resource: "mcp",
			Details:  fmt.Sprintf("MCP server %s: %s", req.Action, req.Name),
		})

		_ = json.NewEncoder(w).Encode(map[string]string{"status": req.Action, "name": req.Name})

	case http.MethodDelete:
		// Delete MCP server
		name := r.URL.Query().Get("name")
		if name == "" {
			WriteErrorSimple(w, http.StatusBadRequest, "Server name required")
			return
		}

		// Disconnect first
		_ = s.mcpClient.Disconnect(name)
		if s.aiClient != nil {
			s.aiClient.GetToolRegistry().UnregisterMCPTools(name)
		}

		if !s.cfg.RemoveMCPServer(name) {
			WriteErrorSimple(w, http.StatusNotFound, "Server not found")
			return
		}

		if err := s.cfg.Save(); err != nil {
			WriteErrorSimple(w, http.StatusInternalServerError, "Failed to save config")
			return
		}

		// Record audit
		username := r.Header.Get("X-Username")
		_ = db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "delete_mcp_server",
			Resource: "mcp",
			Details:  fmt.Sprintf("Deleted MCP server: %s", name),
		})

		_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleMCPTools returns available tools from MCP servers
func (s *Server) handleMCPTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get MCP tools
	mcpTools := s.mcpClient.GetAllTools()
	tools := make([]map[string]interface{}, len(mcpTools))
	for i, t := range mcpTools {
		tools[i] = map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"server":      t.ServerName,
			"schema":      t.InputSchema,
		}
	}

	// Also include built-in tools
	var builtinTools []map[string]interface{}
	if s.aiClient != nil {
		for _, t := range s.aiClient.GetToolRegistry().List() {
			if t.Type != "mcp" {
				builtinTools = append(builtinTools, map[string]interface{}{
					"name":        t.Name,
					"description": t.Description,
					"type":        string(t.Type),
				})
			}
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"mcp_tools":     tools,
		"builtin_tools": builtinTools,
	})
}
