package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/cloudbro-kube-ai/k13d/pkg/automation"
)

func (s *Server) handleGitHubAutomationWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	if s.automation == nil || !s.automation.Enabled() {
		WriteError(w, NewAPIError(ErrCodeNotFound, "GitHub automation is disabled"))
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Failed to read webhook payload"))
		return
	}
	if !automation.VerifyGitHubSignature(s.cfg.GitHub.WebhookSecret, payload, r.Header.Get("X-Hub-Signature-256")) {
		WriteError(w, NewAPIError(ErrCodeUnauthorized, "Invalid GitHub webhook signature"))
		return
	}

	event, err := automation.ParseIssueEvent(r.Header.Get("X-GitHub-Event"), payload)
	if err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, err.Error()))
		return
	}

	result := s.automation.QueueIssueEvent(event)
	w.Header().Set("Content-Type", "application/json")
	status := http.StatusAccepted
	if result.Ignored {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) handleGitHubAutomationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	jobs := []*automation.Job{}
	if s.automation != nil {
		jobs = s.automation.ListJobs()
	}

	resp := map[string]interface{}{
		"enabled": false,
		"config":  s.cfg.GitHub,
		"jobs":    jobs,
	}
	if s.automation != nil {
		resp["enabled"] = s.automation.Enabled()
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleGitHubAutomationJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	if s.automation == nil {
		WriteError(w, NewAPIError(ErrCodeNotFound, "GitHub automation is not initialized"))
		return
	}

	jobID := strings.TrimPrefix(r.URL.Path, "/api/admin/github-automation/jobs/")
	jobID = strings.TrimSpace(jobID)
	if jobID == "" || strings.Contains(jobID, "/") {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid automation job path"))
		return
	}

	job, ok := s.automation.GetJob(jobID)
	if !ok {
		WriteError(w, NewAPIError(ErrCodeNotFound, "Automation job not found"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(job)
}
