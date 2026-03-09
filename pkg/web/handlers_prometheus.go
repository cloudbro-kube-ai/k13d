package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/db"
)

// handlePrometheusMetrics exposes metrics in Prometheus format
// GET /metrics (no auth required for Prometheus scraping)
func (s *Server) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	var sb strings.Builder
	ctx := r.Context()
	contextName, _ := s.k8sClient.GetCurrentContext()

	// Application info
	version := "dev"
	if s.versionInfo != nil {
		version = s.versionInfo.Version
	}
	sb.WriteString("# HELP k13d_info k13d application information\n")
	sb.WriteString("# TYPE k13d_info gauge\n")
	sb.WriteString(fmt.Sprintf("k13d_info{version=\"%s\",context=\"%s\"} 1\n", version, contextName))

	// Cluster metrics from collector (cache-first, SQLite fallback)
	if s.metricsCollector != nil {
		var metrics *db.ClusterMetrics
		if cache := s.metricsCollector.GetCache(); cache != nil {
			metrics = cache.GetLatestClusterMetrics(contextName)
		}
		if metrics == nil {
			if store := s.metricsCollector.GetStore(); store != nil {
				metrics, _ = store.GetLatestClusterMetrics(ctx, contextName)
			}
		}
		if metrics != nil {
			// Node metrics
			sb.WriteString("\n# HELP k13d_cluster_nodes_total Total number of nodes\n")
			sb.WriteString("# TYPE k13d_cluster_nodes_total gauge\n")
			sb.WriteString(fmt.Sprintf("k13d_cluster_nodes_total{context=\"%s\"} %d\n", contextName, metrics.TotalNodes))

			sb.WriteString("\n# HELP k13d_cluster_nodes_ready Number of ready nodes\n")
			sb.WriteString("# TYPE k13d_cluster_nodes_ready gauge\n")
			sb.WriteString(fmt.Sprintf("k13d_cluster_nodes_ready{context=\"%s\"} %d\n", contextName, metrics.ReadyNodes))

			// Pod metrics
			sb.WriteString("\n# HELP k13d_cluster_pods_total Total number of pods\n")
			sb.WriteString("# TYPE k13d_cluster_pods_total gauge\n")
			sb.WriteString(fmt.Sprintf("k13d_cluster_pods_total{context=\"%s\"} %d\n", contextName, metrics.TotalPods))

			sb.WriteString("\n# HELP k13d_cluster_pods_running Number of running pods\n")
			sb.WriteString("# TYPE k13d_cluster_pods_running gauge\n")
			sb.WriteString(fmt.Sprintf("k13d_cluster_pods_running{context=\"%s\"} %d\n", contextName, metrics.RunningPods))

			sb.WriteString("\n# HELP k13d_cluster_pods_pending Number of pending pods\n")
			sb.WriteString("# TYPE k13d_cluster_pods_pending gauge\n")
			sb.WriteString(fmt.Sprintf("k13d_cluster_pods_pending{context=\"%s\"} %d\n", contextName, metrics.PendingPods))

			sb.WriteString("\n# HELP k13d_cluster_pods_failed Number of failed pods\n")
			sb.WriteString("# TYPE k13d_cluster_pods_failed gauge\n")
			sb.WriteString(fmt.Sprintf("k13d_cluster_pods_failed{context=\"%s\"} %d\n", contextName, metrics.FailedPods))

			// CPU metrics
			sb.WriteString("\n# HELP k13d_cluster_cpu_capacity_millicores Total CPU capacity in millicores\n")
			sb.WriteString("# TYPE k13d_cluster_cpu_capacity_millicores gauge\n")
			sb.WriteString(fmt.Sprintf("k13d_cluster_cpu_capacity_millicores{context=\"%s\"} %d\n", contextName, metrics.TotalCPUMillis))

			sb.WriteString("\n# HELP k13d_cluster_cpu_used_millicores Used CPU in millicores\n")
			sb.WriteString("# TYPE k13d_cluster_cpu_used_millicores gauge\n")
			sb.WriteString(fmt.Sprintf("k13d_cluster_cpu_used_millicores{context=\"%s\"} %d\n", contextName, metrics.UsedCPUMillis))

			// Memory metrics
			sb.WriteString("\n# HELP k13d_cluster_memory_capacity_mb Total memory capacity in MB\n")
			sb.WriteString("# TYPE k13d_cluster_memory_capacity_mb gauge\n")
			sb.WriteString(fmt.Sprintf("k13d_cluster_memory_capacity_mb{context=\"%s\"} %d\n", contextName, metrics.TotalMemoryMB))

			sb.WriteString("\n# HELP k13d_cluster_memory_used_mb Used memory in MB\n")
			sb.WriteString("# TYPE k13d_cluster_memory_used_mb gauge\n")
			sb.WriteString(fmt.Sprintf("k13d_cluster_memory_used_mb{context=\"%s\"} %d\n", contextName, metrics.UsedMemoryMB))

			// Deployment metrics
			sb.WriteString("\n# HELP k13d_cluster_deployments_total Total number of deployments\n")
			sb.WriteString("# TYPE k13d_cluster_deployments_total gauge\n")
			sb.WriteString(fmt.Sprintf("k13d_cluster_deployments_total{context=\"%s\"} %d\n", contextName, metrics.TotalDeployments))

			sb.WriteString("\n# HELP k13d_cluster_deployments_ready Number of ready deployments\n")
			sb.WriteString("# TYPE k13d_cluster_deployments_ready gauge\n")
			sb.WriteString(fmt.Sprintf("k13d_cluster_deployments_ready{context=\"%s\"} %d\n", contextName, metrics.ReadyDeployments))
		}
	}

	// LLM usage metrics
	llmFilter := db.LLMUsageFilter{
		StartTime: time.Now().Add(-24 * time.Hour),
		EndTime:   time.Now(),
	}
	llmStats, err := db.GetLLMUsageStats(ctx, llmFilter)
	if err == nil && llmStats != nil {
		sb.WriteString("\n# HELP k13d_llm_requests_total Total LLM requests in last 24h\n")
		sb.WriteString("# TYPE k13d_llm_requests_total gauge\n")
		sb.WriteString(fmt.Sprintf("k13d_llm_requests_total %d\n", llmStats.TotalRequests))

		sb.WriteString("\n# HELP k13d_llm_tokens_total Total LLM tokens used in last 24h\n")
		sb.WriteString("# TYPE k13d_llm_tokens_total gauge\n")
		sb.WriteString(fmt.Sprintf("k13d_llm_tokens_total %d\n", llmStats.TotalTokens))
	}

	// Collector status
	if s.metricsCollector != nil {
		running := 0
		if s.metricsCollector.IsRunning() {
			running = 1
		}
		sb.WriteString("\n# HELP k13d_metrics_collector_running Metrics collector status\n")
		sb.WriteString("# TYPE k13d_metrics_collector_running gauge\n")
		sb.WriteString(fmt.Sprintf("k13d_metrics_collector_running %d\n", running))
	}

	_, _ = w.Write([]byte(sb.String()))
}

// handlePrometheusSettings handles Prometheus configuration
// GET/PUT /api/prometheus/settings
func (s *Server) handlePrometheusSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"expose_metrics":      s.cfg.Prometheus.ExposeMetrics,
			"external_url":        s.cfg.Prometheus.ExternalURL,
			"has_auth":            s.cfg.Prometheus.Username != "",
			"collect_k8s_metrics": s.cfg.Prometheus.CollectK8sMetrics,
			"collection_interval": s.cfg.Prometheus.CollectionInterval,
		})

	case http.MethodPut:
		var settings struct {
			ExposeMetrics      bool   `json:"expose_metrics"`
			ExternalURL        string `json:"external_url"`
			Username           string `json:"username"`
			Password           string `json:"password"`
			CollectK8sMetrics  bool   `json:"collect_k8s_metrics"`
			CollectionInterval int    `json:"collection_interval"`
		}

		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
			return
		}

		s.cfg.Prometheus.ExposeMetrics = settings.ExposeMetrics
		s.cfg.Prometheus.ExternalURL = settings.ExternalURL
		if settings.Username != "" {
			s.cfg.Prometheus.Username = settings.Username
		}
		if settings.Password != "" {
			s.cfg.Prometheus.Password = settings.Password
		}
		s.cfg.Prometheus.CollectK8sMetrics = settings.CollectK8sMetrics
		if settings.CollectionInterval > 0 {
			s.cfg.Prometheus.CollectionInterval = settings.CollectionInterval
		}

		if err := s.cfg.Save(); err != nil {
			WriteError(w, NewAPIError(ErrCodeInternalError, "Failed to save settings"))
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]string{"status": "saved"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePrometheusTest tests connection to external Prometheus server
// POST /api/prometheus/test
func (s *Server) handlePrometheusTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req struct {
		URL      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	if req.URL == "" {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "URL is required"))
		return
	}

	// Test connection by querying Prometheus API
	testURL := strings.TrimSuffix(req.URL, "/") + "/api/v1/status/buildinfo"

	httpReq, err := http.NewRequestWithContext(r.Context(), "GET", testURL, nil)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if req.Username != "" && req.Password != "" {
		httpReq.SetBasicAuth(req.Username, req.Password)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		})
		return
	}

	// Parse response to get version
	var promResp struct {
		Status string `json:"status"`
		Data   struct {
			Version string `json:"version"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"version": "unknown",
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"version": promResp.Data.Version,
	})
}

// handlePrometheusQuery proxies queries to external Prometheus server
// GET /api/prometheus/query
func (s *Server) handlePrometheusQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.cfg.Prometheus.ExternalURL == "" {
		WriteError(w, NewAPIError(ErrCodeNotFound, "External Prometheus not configured"))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Proxy the query to external Prometheus
	query := r.URL.Query().Get("query")
	if query == "" {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Query parameter required"))
		return
	}

	proxyURL := fmt.Sprintf("%s/api/v1/query?query=%s",
		strings.TrimSuffix(s.cfg.Prometheus.ExternalURL, "/"),
		query,
	)

	httpReq, err := http.NewRequestWithContext(r.Context(), "GET", proxyURL, nil)
	if err != nil {
		WriteError(w, NewAPIError(ErrCodeInternalError, err.Error()))
		return
	}

	if s.cfg.Prometheus.Username != "" && s.cfg.Prometheus.Password != "" {
		httpReq.SetBasicAuth(s.cfg.Prometheus.Username, s.cfg.Prometheus.Password)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		WriteError(w, NewAPIError(ErrCodeInternalError, err.Error()))
		return
	}
	defer resp.Body.Close()

	// Forward response
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
