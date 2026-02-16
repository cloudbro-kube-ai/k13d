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
)

// ==========================================
// Settings Handlers
// ==========================================

func (s *Server) handleAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"logs":  []interface{}{},
		})
		return
	}

	if logs == nil {
		logs = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
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
		json.NewEncoder(w).Encode(map[string]interface{}{
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
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Update settings (protected by mutex for concurrent access)
		s.aiMu.Lock()
		s.cfg.Language = newSettings.Language
		s.cfg.BeginnerMode = newSettings.BeginnerMode
		s.cfg.EnableAudit = newSettings.EnableAudit
		s.cfg.LogLevel = newSettings.LogLevel
		s.aiMu.Unlock()

		// Save to YAML
		if err := s.cfg.Save(); err != nil {
			http.Error(w, "Failed to save settings", http.StatusInternalServerError)
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
			fmt.Printf("Warning: failed to save settings to SQLite: %v\n", err)
		}

		// Record audit
		username := r.Header.Get("X-Username")
		db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "update_settings",
			Resource: "settings",
			Details:  fmt.Sprintf("Settings updated (timezone: %s)", newSettings.Timezone),
		})

		json.NewEncoder(w).Encode(map[string]string{"status": "saved"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
				"name":        m.Name,
				"provider":    m.Provider,
				"model":       m.Model,
				"endpoint":    m.Endpoint,
				"description": m.Description,
				"has_api_key": m.APIKey != "",
				"is_active":   m.Name == s.cfg.ActiveModel,
			}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models":       models,
			"active_model": s.cfg.ActiveModel,
		})

	case http.MethodPost:
		// Add new model profile
		var profile config.ModelProfile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if profile.Name == "" || profile.Provider == "" || profile.Model == "" {
			http.Error(w, "Name, provider, and model are required", http.StatusBadRequest)
			return
		}

		s.cfg.AddModelProfile(profile)
		if err := s.cfg.Save(); err != nil {
			http.Error(w, "Failed to save config", http.StatusInternalServerError)
			return
		}

		// Record audit
		username := r.Header.Get("X-Username")
		db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "add_model_profile",
			Resource: "model",
			Details:  fmt.Sprintf("Added model profile: %s (%s/%s)", profile.Name, profile.Provider, profile.Model),
		})

		json.NewEncoder(w).Encode(map[string]string{"status": "created", "name": profile.Name})

	case http.MethodDelete:
		// Delete model profile
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, "Model name required", http.StatusBadRequest)
			return
		}

		if !s.cfg.RemoveModelProfile(name) {
			http.Error(w, "Model not found", http.StatusNotFound)
			return
		}

		if err := s.cfg.Save(); err != nil {
			http.Error(w, "Failed to save config", http.StatusInternalServerError)
			return
		}

		// Record audit
		username := r.Header.Get("X-Username")
		db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "delete_model_profile",
			Resource: "model",
			Details:  fmt.Sprintf("Deleted model profile: %s", name),
		})

		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleActiveModel switches the active LLM model
func (s *Server) handleActiveModel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		profile := s.cfg.GetActiveModelProfile()
		if profile == nil {
			http.Error(w, "No active model", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
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
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.aiMu.Lock()
		if !s.cfg.SetActiveModel(req.Name) {
			s.aiMu.Unlock()
			http.Error(w, "Model not found", http.StatusNotFound)
			return
		}

		// Recreate AI client with new model
		newClient, err := ai.NewClient(&s.cfg.LLM)
		if err != nil {
			s.aiMu.Unlock()
			http.Error(w, fmt.Sprintf("Failed to create AI client: %v", err), http.StatusInternalServerError)
			return
		}
		s.aiClient = newClient
		s.aiMu.Unlock()

		// Re-register MCP tools
		for _, serverName := range s.mcpClient.GetConnectedServers() {
			s.registerMCPTools(serverName)
		}

		if err := s.cfg.Save(); err != nil {
			http.Error(w, "Failed to save config", http.StatusInternalServerError)
			return
		}

		// Record audit
		username := r.Header.Get("X-Username")
		db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "switch_model",
			Resource: "model",
			Details:  fmt.Sprintf("Switched to model: %s", req.Name),
		})

		json.NewEncoder(w).Encode(map[string]string{"status": "switched", "active_model": req.Name})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		json.NewEncoder(w).Encode(map[string]interface{}{
			"servers":   servers,
			"connected": s.mcpClient.GetConnectedServers(),
		})

	case http.MethodPost:
		// Add new MCP server
		var server config.MCPServer
		if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if server.Name == "" || server.Command == "" {
			http.Error(w, "Name and command are required", http.StatusBadRequest)
			return
		}

		s.cfg.AddMCPServer(server)
		if err := s.cfg.Save(); err != nil {
			http.Error(w, "Failed to save config", http.StatusInternalServerError)
			return
		}

		// If enabled, try to connect
		if server.Enabled {
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()
			if err := s.mcpClient.Connect(ctx, server); err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{
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
		db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "add_mcp_server",
			Resource: "mcp",
			Details:  fmt.Sprintf("Added MCP server: %s (%s)", server.Name, server.Command),
		})

		json.NewEncoder(w).Encode(map[string]string{"status": "created", "name": server.Name})

	case http.MethodPut:
		// Toggle MCP server enabled/disabled or reconnect
		var req struct {
			Name   string `json:"name"`
			Action string `json:"action"` // "enable", "disable", "reconnect"
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		switch req.Action {
		case "enable":
			if !s.cfg.ToggleMCPServer(req.Name, true) {
				http.Error(w, "Server not found", http.StatusNotFound)
				return
			}
			// Try to connect
			for _, srv := range s.cfg.MCP.Servers {
				if srv.Name == req.Name {
					ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
					if err := s.mcpClient.Connect(ctx, srv); err != nil {
						cancel()
						json.NewEncoder(w).Encode(map[string]interface{}{
							"status":  "enabled",
							"warning": fmt.Sprintf("Enabled but failed to connect: %v", err),
						})
						s.cfg.Save()
						return
					}
					cancel()
					s.registerMCPTools(srv.Name)
					break
				}
			}

		case "disable":
			if !s.cfg.ToggleMCPServer(req.Name, false) {
				http.Error(w, "Server not found", http.StatusNotFound)
				return
			}
			// Disconnect and unregister tools
			s.mcpClient.Disconnect(req.Name)
			if s.aiClient != nil {
				s.aiClient.GetToolRegistry().UnregisterMCPTools(req.Name)
			}

		case "reconnect":
			// Disconnect first
			s.mcpClient.Disconnect(req.Name)
			if s.aiClient != nil {
				s.aiClient.GetToolRegistry().UnregisterMCPTools(req.Name)
			}
			// Reconnect
			for _, srv := range s.cfg.MCP.Servers {
				if srv.Name == req.Name && srv.Enabled {
					ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
					if err := s.mcpClient.Connect(ctx, srv); err != nil {
						cancel()
						http.Error(w, fmt.Sprintf("Failed to reconnect: %v", err), http.StatusInternalServerError)
						return
					}
					cancel()
					s.registerMCPTools(srv.Name)
					break
				}
			}

		default:
			http.Error(w, "Invalid action (use: enable, disable, reconnect)", http.StatusBadRequest)
			return
		}

		if err := s.cfg.Save(); err != nil {
			http.Error(w, "Failed to save config", http.StatusInternalServerError)
			return
		}

		// Record audit
		username := r.Header.Get("X-Username")
		db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   fmt.Sprintf("mcp_server_%s", req.Action),
			Resource: "mcp",
			Details:  fmt.Sprintf("MCP server %s: %s", req.Action, req.Name),
		})

		json.NewEncoder(w).Encode(map[string]string{"status": req.Action, "name": req.Name})

	case http.MethodDelete:
		// Delete MCP server
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, "Server name required", http.StatusBadRequest)
			return
		}

		// Disconnect first
		s.mcpClient.Disconnect(name)
		if s.aiClient != nil {
			s.aiClient.GetToolRegistry().UnregisterMCPTools(name)
		}

		if !s.cfg.RemoveMCPServer(name) {
			http.Error(w, "Server not found", http.StatusNotFound)
			return
		}

		if err := s.cfg.Save(); err != nil {
			http.Error(w, "Failed to save config", http.StatusInternalServerError)
			return
		}

		// Record audit
		username := r.Header.Get("X-Username")
		db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "delete_mcp_server",
			Resource: "mcp",
			Details:  fmt.Sprintf("Deleted MCP server: %s", name),
		})

		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleMCPTools returns available tools from MCP servers
func (s *Server) handleMCPTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	json.NewEncoder(w).Encode(map[string]interface{}{
		"mcp_tools":     tools,
		"builtin_tools": builtinTools,
	})
}
