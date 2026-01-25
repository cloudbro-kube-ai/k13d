package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"
)

// ==========================================
// LLM Usage Tracking Handlers
// ==========================================

// handleLLMUsage retrieves LLM usage records with optional filtering
// GET /api/llm/usage?user=xxx&model=xxx&start=xxx&end=xxx&limit=xxx
func (s *Server) handleLLMUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Build filter from query parameters
	filter := db.LLMUsageFilter{
		User:     r.URL.Query().Get("user"),
		Model:    r.URL.Query().Get("model"),
		Provider: r.URL.Query().Get("provider"),
		Limit:    100, // Default limit
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		var limit int
		fmt.Sscanf(limitStr, "%d", &limit)
		if limit > 0 && limit <= 1000 {
			filter.Limit = limit
		}
	}

	// Parse start time
	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartTime = t
		}
	}

	// Parse end time
	if endStr := r.URL.Query().Get("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndTime = t
		}
	}

	// Default to last 24 hours if no time range specified
	if filter.StartTime.IsZero() && filter.EndTime.IsZero() {
		filter.StartTime = time.Now().Add(-24 * time.Hour)
	}

	records, err := db.GetLLMUsage(r.Context(), filter)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"items": []interface{}{},
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": records,
		"count": len(records),
		"filter": map[string]interface{}{
			"user":     filter.User,
			"model":    filter.Model,
			"provider": filter.Provider,
			"start":    filter.StartTime,
			"end":      filter.EndTime,
			"limit":    filter.Limit,
		},
	})
}

// handleLLMUsageStats retrieves aggregated LLM usage statistics
// GET /api/llm/usage/stats?minutes=5&user=xxx&provider=xxx
// GET /api/llm/usage/stats?hours=24&user=xxx&provider=xxx
func (s *Server) handleLLMUsageStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Build filter from query parameters
	filter := db.LLMUsageFilter{
		User:     r.URL.Query().Get("user"),
		Provider: r.URL.Query().Get("provider"),
	}

	// Parse time range - support both minutes and hours (default 5 minutes)
	minutes := 5
	if minutesStr := r.URL.Query().Get("minutes"); minutesStr != "" {
		fmt.Sscanf(minutesStr, "%d", &minutes)
		if minutes <= 0 || minutes > 10080 { // Max 7 days in minutes
			minutes = 5
		}
	} else if hoursStr := r.URL.Query().Get("hours"); hoursStr != "" {
		var hours int
		fmt.Sscanf(hoursStr, "%d", &hours)
		if hours > 0 && hours <= 168 {
			minutes = hours * 60
		}
	}

	filter.StartTime = time.Now().Add(-time.Duration(minutes) * time.Minute)

	stats, err := db.GetLLMUsageStats(r.Context(), filter)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"stats":      stats,
		"minutes":    minutes,
		"start_time": filter.StartTime,
		"end_time":   time.Now(),
	})
}
