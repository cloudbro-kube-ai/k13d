package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudbro-kube-ai/k13d/pkg/db"
	"github.com/cloudbro-kube-ai/k13d/pkg/helm"
)

// ==========================================
// Helm Handlers
// ==========================================

// handleHelmReleases lists Helm releases
func (s *Server) handleHelmReleases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	namespace := r.URL.Query().Get("namespace")
	allNamespaces := r.URL.Query().Get("all") == "true"

	releases, err := s.helmClient.ListReleases(r.Context(), namespace, allNamespaces)
	if err != nil {
		writeK8sError(w, fmt.Errorf("failed to list releases: %w", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items": releases,
	})
}

// handleHelmRelease handles single release operations (get, history, values, manifest)
func (s *Server) handleHelmRelease(w http.ResponseWriter, r *http.Request) {
	// Extract release name and action from path: /api/helm/release/{name}/{action}
	path := strings.TrimPrefix(r.URL.Path, "/api/helm/release/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		WriteError(w, NewAPIError(ErrCodeValidation, "Release name required"))
		return
	}

	name := parts[0]
	action := "get"
	if len(parts) > 1 {
		action = parts[1]
	}

	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	w.Header().Set("Content-Type", "application/json")

	switch action {
	case "get", "":
		release, err := s.helmClient.GetRelease(r.Context(), name, namespace)
		if err != nil {
			writeK8sError(w, fmt.Errorf("failed to get release: %w", err))
			return
		}
		_ = json.NewEncoder(w).Encode(release)

	case "history":
		history, err := s.helmClient.GetReleaseHistory(r.Context(), name, namespace)
		if err != nil {
			writeK8sError(w, fmt.Errorf("failed to get release history: %w", err))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"items": history,
		})

	case "values":
		allValues := r.URL.Query().Get("all") == "true"
		values, err := s.helmClient.GetReleaseValues(r.Context(), name, namespace, allValues)
		if err != nil {
			writeK8sError(w, fmt.Errorf("failed to get release values: %w", err))
			return
		}
		_ = json.NewEncoder(w).Encode(values)

	case "manifest":
		manifest, err := s.helmClient.GetReleaseManifest(r.Context(), name, namespace)
		if err != nil {
			writeK8sError(w, fmt.Errorf("failed to get release manifest: %w", err))
			return
		}
		w.Header().Set("Content-Type", "text/yaml")
		_, _ = w.Write([]byte(manifest))

	case "notes":
		notes, err := s.helmClient.GetReleaseNotes(r.Context(), name, namespace)
		if err != nil {
			writeK8sError(w, fmt.Errorf("failed to get release notes: %w", err))
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(notes))

	default:
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Unknown action"))
	}
}

// handleHelmInstall installs a Helm chart
func (s *Server) handleHelmInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req struct {
		Name            string                 `json:"name"`
		Chart           string                 `json:"chart"`
		Namespace       string                 `json:"namespace"`
		Values          map[string]interface{} `json:"values"`
		CreateNamespace bool                   `json:"createNamespace"`
		Wait            bool                   `json:"wait"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	if req.Name == "" || req.Chart == "" {
		WriteError(w, NewAPIError(ErrCodeValidation, "Name and chart are required"))
		return
	}

	if req.Namespace == "" {
		req.Namespace = "default"
	}

	opts := &helm.InstallOptions{
		CreateNamespace: req.CreateNamespace,
		Wait:            req.Wait,
	}

	release, err := s.helmClient.InstallRelease(r.Context(), req.Name, req.Chart, req.Namespace, req.Values, opts)
	if err != nil {
		writeK8sError(w, fmt.Errorf("failed to install release: %w", err))
		return
	}

	// Record audit
	username := r.Header.Get("X-Username")
	_ = db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "helm_install",
		Resource: "helm",
		Details:  fmt.Sprintf("Installed %s from %s in %s", req.Name, req.Chart, req.Namespace),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(release)
}

// handleHelmUpgrade upgrades a Helm release
func (s *Server) handleHelmUpgrade(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req struct {
		Name        string                 `json:"name"`
		Chart       string                 `json:"chart"`
		Namespace   string                 `json:"namespace"`
		Values      map[string]interface{} `json:"values"`
		ReuseValues bool                   `json:"reuseValues"`
		ResetValues bool                   `json:"resetValues"`
		Wait        bool                   `json:"wait"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	if req.Name == "" || req.Chart == "" {
		WriteError(w, NewAPIError(ErrCodeValidation, "Name and chart are required"))
		return
	}

	if req.Namespace == "" {
		req.Namespace = "default"
	}

	opts := &helm.UpgradeOptions{
		Wait:        req.Wait,
		ReuseValues: req.ReuseValues,
		ResetValues: req.ResetValues,
	}

	release, err := s.helmClient.UpgradeRelease(r.Context(), req.Name, req.Chart, req.Namespace, req.Values, opts)
	if err != nil {
		writeK8sError(w, fmt.Errorf("failed to upgrade release: %w", err))
		return
	}

	// Record audit
	username := r.Header.Get("X-Username")
	_ = db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "helm_upgrade",
		Resource: "helm",
		Details:  fmt.Sprintf("Upgraded %s to %s in %s", req.Name, req.Chart, req.Namespace),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(release)
}

// handleHelmUninstall uninstalls a Helm release
func (s *Server) handleHelmUninstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		writeMethodNotAllowed(w)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Namespace   string `json:"namespace"`
		KeepHistory bool   `json:"keepHistory"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	if req.Name == "" {
		WriteError(w, NewAPIError(ErrCodeValidation, "Name is required"))
		return
	}

	if req.Namespace == "" {
		req.Namespace = "default"
	}

	err := s.helmClient.UninstallRelease(r.Context(), req.Name, req.Namespace, req.KeepHistory)
	if err != nil {
		writeK8sError(w, fmt.Errorf("failed to uninstall release: %w", err))
		return
	}

	// Record audit
	username := r.Header.Get("X-Username")
	_ = db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "helm_uninstall",
		Resource: "helm",
		Details:  fmt.Sprintf("Uninstalled %s from %s", req.Name, req.Namespace),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "uninstalled"})
}

// handleHelmRollback rolls back a Helm release
func (s *Server) handleHelmRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Revision  int    `json:"revision"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	if req.Name == "" || req.Revision == 0 {
		WriteError(w, NewAPIError(ErrCodeValidation, "Name and revision are required"))
		return
	}

	if req.Namespace == "" {
		req.Namespace = "default"
	}

	err := s.helmClient.RollbackRelease(r.Context(), req.Name, req.Namespace, req.Revision)
	if err != nil {
		writeK8sError(w, fmt.Errorf("failed to rollback release: %w", err))
		return
	}

	// Record audit
	username := r.Header.Get("X-Username")
	_ = db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "helm_rollback",
		Resource: "helm",
		Details:  fmt.Sprintf("Rolled back %s to revision %d in %s", req.Name, req.Revision, req.Namespace),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "rolled back"})
}

// handleHelmRepos manages Helm repositories
func (s *Server) handleHelmRepos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		repos, err := s.helmClient.ListRepositories()
		if err != nil {
			writeK8sError(w, fmt.Errorf("failed to list repositories: %w", err))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"items": repos,
		})

	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
			return
		}

		if req.Name == "" || req.URL == "" {
			WriteError(w, NewAPIError(ErrCodeValidation, "Name and URL are required"))
			return
		}

		if err := s.helmClient.AddRepository(req.Name, req.URL); err != nil {
			writeK8sError(w, fmt.Errorf("failed to add repository: %w", err))
			return
		}

		// Record audit
		username := r.Header.Get("X-Username")
		_ = db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "helm_repo_add",
			Resource: "helm",
			Details:  fmt.Sprintf("Added repository %s (%s)", req.Name, req.URL),
		})

		_ = json.NewEncoder(w).Encode(map[string]string{"status": "added"})

	case http.MethodPut:
		// Update (refresh) all repositories
		if err := s.helmClient.UpdateRepositories(); err != nil {
			writeK8sError(w, fmt.Errorf("failed to update repositories: %w", err))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	case http.MethodDelete:
		name := r.URL.Query().Get("name")
		if name == "" {
			WriteError(w, NewAPIError(ErrCodeValidation, "Repository name required"))
			return
		}

		if err := s.helmClient.RemoveRepository(name); err != nil {
			writeK8sError(w, fmt.Errorf("failed to remove repository: %w", err))
			return
		}

		// Record audit
		username := r.Header.Get("X-Username")
		_ = db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "helm_repo_remove",
			Resource: "helm",
			Details:  fmt.Sprintf("Removed repository %s", name),
		})

		_ = json.NewEncoder(w).Encode(map[string]string{"status": "removed"})

	default:
		writeMethodNotAllowed(w)
	}
}

// handleHelmSearch searches for charts in repositories
func (s *Server) handleHelmSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	keyword := r.URL.Query().Get("q")
	if keyword == "" {
		WriteError(w, NewAPIError(ErrCodeValidation, "Search keyword required (q parameter)"))
		return
	}

	results, err := s.helmClient.SearchCharts(keyword)
	if err != nil {
		writeK8sError(w, fmt.Errorf("failed to search charts: %w", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items": results,
	})
}
