package web

import (
	"encoding/json"
	"net/http"

	"k8s.io/client-go/tools/clientcmd"
)

// ContextInfo represents a kubeconfig context
type ContextInfo struct {
	Name      string `json:"name"`
	Cluster   string `json:"cluster"`
	User      string `json:"user"`
	Namespace string `json:"namespace,omitempty"`
	IsCurrent bool   `json:"isCurrent"`
}

// ContextsResponse is the response for listing contexts
type ContextsResponse struct {
	Contexts       []ContextInfo `json:"contexts"`
	CurrentContext string        `json:"currentContext"`
}

func (s *Server) handleContexts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, err := loadingRules.Load()
	if err != nil {
		http.Error(w, "Failed to load kubeconfig: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var contexts []ContextInfo
	for name, ctx := range config.Contexts {
		contexts = append(contexts, ContextInfo{
			Name:      name,
			Cluster:   ctx.Cluster,
			User:      ctx.AuthInfo,
			Namespace: ctx.Namespace,
			IsCurrent: name == config.CurrentContext,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ContextsResponse{
		Contexts:       contexts,
		CurrentContext: config.CurrentContext,
	})
}

func (s *Server) handleContextSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Context string `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Context == "" {
		http.Error(w, "Context name is required", http.StatusBadRequest)
		return
	}

	// Verify context exists
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, err := loadingRules.Load()
	if err != nil {
		http.Error(w, "Failed to load kubeconfig: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if _, ok := config.Contexts[req.Context]; !ok {
		http.Error(w, "Context not found: "+req.Context, http.StatusNotFound)
		return
	}

	// Switch the k8s client to the new context
	if err := s.k8sClient.SwitchContext(req.Context); err != nil {
		http.Error(w, "Failed to switch context: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"context": req.Context,
		"message": "Switched to context: " + req.Context,
	})
}
