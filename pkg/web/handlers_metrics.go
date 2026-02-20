package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/db"
)

// ==========================================
// Metrics Handlers
// ==========================================

// PodMetricItem represents pod resource usage for API response
type PodMetricItem struct {
	Name      string  `json:"name"`
	Namespace string  `json:"namespace"`
	CPU       float64 `json:"cpu"`    // millicores
	Memory    float64 `json:"memory"` // MiB
}

func (s *Server) handlePodMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	namespace := r.URL.Query().Get("namespace")
	w.Header().Set("Content-Type", "application/json")

	// Try to get metrics from metrics-server
	metricsMap, err := s.k8sClient.GetPodMetrics(r.Context(), namespace)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Metrics server not available: " + err.Error(),
			"items": []PodMetricItem{},
		})
		return
	}

	// Convert map to slice
	var items []PodMetricItem
	for name, values := range metricsMap {
		if len(values) >= 2 {
			items = append(items, PodMetricItem{
				Name:      name,
				Namespace: namespace,
				CPU:       float64(values[0]),
				Memory:    float64(values[1]),
			})
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items": items,
	})
}

func (s *Server) handleNodeMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	metricsMap, err := s.k8sClient.GetNodeMetrics(r.Context())
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Metrics server not available: " + err.Error(),
			"items": []interface{}{},
		})
		return
	}

	// Convert map to slice
	var items []map[string]interface{}
	for name, values := range metricsMap {
		if len(values) >= 2 {
			items = append(items, map[string]interface{}{
				"name":   name,
				"cpu":    values[0],
				"memory": values[1],
			})
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items": items,
	})
}

// ==========================================
// Time-Series Metrics History Handlers
// ==========================================

func (s *Server) handleClusterMetricsHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if s.metricsCollector == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Metrics collector not initialized",
			"items": []interface{}{},
		})
		return
	}

	// Parse query parameters - support both minutes and hours
	minutes := 5 // Default to 5 minutes
	if m := r.URL.Query().Get("minutes"); m != "" {
		_, _ = fmt.Sscanf(m, "%d", &minutes)
	} else if h := r.URL.Query().Get("hours"); h != "" {
		var hours int
		_, _ = fmt.Sscanf(h, "%d", &hours)
		minutes = hours * 60
	}
	limit := 1000
	if l := r.URL.Query().Get("limit"); l != "" {
		_, _ = fmt.Sscanf(l, "%d", &limit)
	}

	contextName, _ := s.k8sClient.GetCurrentContext()
	end := time.Now()
	start := end.Add(-time.Duration(minutes) * time.Minute)

	// Try ring buffer cache first for recent data
	var metrics []db.ClusterMetrics
	if cache := s.metricsCollector.GetCache(); cache != nil {
		metrics = cache.GetClusterMetrics(contextName, start, end, limit)
	}

	// Fall back to SQLite if cache doesn't cover the range
	if metrics == nil {
		var err error
		metrics, err = s.metricsCollector.GetStore().GetClusterMetrics(r.Context(), contextName, start, end, limit)
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": err.Error(),
				"items": []interface{}{},
			})
			return
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items":   metrics,
		"context": contextName,
		"start":   start,
		"end":     end,
	})
}

func (s *Server) handleNodeMetricsHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if s.metricsCollector == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Metrics collector not initialized",
			"items": []interface{}{},
		})
		return
	}

	nodeName := r.URL.Query().Get("node")
	if nodeName == "" {
		http.Error(w, "node parameter required", http.StatusBadRequest)
		return
	}

	// Parse query parameters - support both minutes and hours
	minutes := 5 // Default to 5 minutes
	if m := r.URL.Query().Get("minutes"); m != "" {
		_, _ = fmt.Sscanf(m, "%d", &minutes)
	} else if h := r.URL.Query().Get("hours"); h != "" {
		var hours int
		_, _ = fmt.Sscanf(h, "%d", &hours)
		minutes = hours * 60
	}
	limit := 1000
	if l := r.URL.Query().Get("limit"); l != "" {
		_, _ = fmt.Sscanf(l, "%d", &limit)
	}

	contextName, _ := s.k8sClient.GetCurrentContext()
	end := time.Now()
	start := end.Add(-time.Duration(minutes) * time.Minute)

	// Try ring buffer cache first
	var metrics []db.NodeMetrics
	if cache := s.metricsCollector.GetCache(); cache != nil {
		metrics = cache.GetNodeMetrics(contextName, nodeName, start, end, limit)
	}

	if metrics == nil {
		var err error
		metrics, err = s.metricsCollector.GetStore().GetNodeMetricsHistory(r.Context(), contextName, nodeName, start, end, limit)
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": err.Error(),
				"items": []interface{}{},
			})
			return
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items":   metrics,
		"context": contextName,
		"node":    nodeName,
		"start":   start,
		"end":     end,
	})
}

func (s *Server) handlePodMetricsHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if s.metricsCollector == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Metrics collector not initialized",
			"items": []interface{}{},
		})
		return
	}

	namespace := r.URL.Query().Get("namespace")
	podName := r.URL.Query().Get("pod")
	if podName == "" {
		http.Error(w, "pod parameter required", http.StatusBadRequest)
		return
	}
	if namespace == "" {
		namespace = "default"
	}

	// Parse query parameters - support both minutes and hours
	minutes := 5 // Default to 5 minutes
	if m := r.URL.Query().Get("minutes"); m != "" {
		_, _ = fmt.Sscanf(m, "%d", &minutes)
	} else if h := r.URL.Query().Get("hours"); h != "" {
		var hours int
		_, _ = fmt.Sscanf(h, "%d", &hours)
		minutes = hours * 60
	}
	limit := 1000
	if l := r.URL.Query().Get("limit"); l != "" {
		_, _ = fmt.Sscanf(l, "%d", &limit)
	}

	contextName, _ := s.k8sClient.GetCurrentContext()
	end := time.Now()
	start := end.Add(-time.Duration(minutes) * time.Minute)

	// Try ring buffer cache first
	var metrics []db.PodMetrics
	if cache := s.metricsCollector.GetCache(); cache != nil {
		metrics = cache.GetPodMetrics(contextName, namespace, podName, start, end, limit)
	}

	if metrics == nil {
		var err error
		metrics, err = s.metricsCollector.GetStore().GetPodMetricsHistory(r.Context(), contextName, namespace, podName, start, end, limit)
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": err.Error(),
				"items": []interface{}{},
			})
			return
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items":     metrics,
		"context":   contextName,
		"namespace": namespace,
		"pod":       podName,
		"start":     start,
		"end":       end,
	})
}

func (s *Server) handleMetricsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if s.metricsCollector == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":      "Metrics collector not initialized",
			"enabled":    false,
			"collecting": false,
		})
		return
	}

	contextName, _ := s.k8sClient.GetCurrentContext()

	summary, err := s.metricsCollector.GetStore().GetMetricsSummary(r.Context(), contextName)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   err.Error(),
			"enabled": true,
		})
		return
	}

	summary["enabled"] = true
	summary["collecting"] = s.metricsCollector.IsRunning()
	summary["context"] = contextName

	_ = json.NewEncoder(w).Encode(summary)
}

func (s *Server) handleAggregatedMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if s.metricsCollector == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Metrics collector not initialized",
			"items": []interface{}{},
		})
		return
	}

	// Parse query parameters - support both minutes and hours
	minutes := 5 // Default to 5 minutes
	if m := r.URL.Query().Get("minutes"); m != "" {
		_, _ = fmt.Sscanf(m, "%d", &minutes)
	} else if h := r.URL.Query().Get("hours"); h != "" {
		var hours int
		_, _ = fmt.Sscanf(h, "%d", &hours)
		minutes = hours * 60
	}
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		// Auto-select interval based on time range
		if minutes <= 60 {
			interval = "minute"
		} else if minutes <= 1440 { // 24 hours
			interval = "hour"
		} else {
			interval = "day"
		}
	}

	contextName, _ := s.k8sClient.GetCurrentContext()
	end := time.Now()
	start := end.Add(-time.Duration(minutes) * time.Minute)

	metrics, err := s.metricsCollector.GetStore().GetAggregatedClusterMetrics(r.Context(), contextName, start, end, interval)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"items": []interface{}{},
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items":    metrics,
		"context":  contextName,
		"interval": interval,
		"start":    start,
		"end":      end,
	})
}

func (s *Server) handleMetricsCollectNow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if s.metricsCollector == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Metrics collector not initialized",
		})
		return
	}

	if err := s.metricsCollector.CollectNow(); err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Metrics collection triggered",
		"timestamp": time.Now(),
	})
}
