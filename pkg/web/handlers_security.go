package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/security"
)

// ==========================================
// Security Scanning Handlers
// ==========================================

func (s *Server) handleSecurityScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if s.securityScanner == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Security scanner not initialized",
		})
		return
	}

	namespace := r.URL.Query().Get("namespace")
	username := r.Context().Value("username")
	triggeredBy := ""
	if username != nil {
		triggeredBy = username.(string)
	}

	result, err := s.securityScanner.Scan(r.Context(), namespace)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Record scan to database
	s.recordSecurityScan(result, namespace, "full", triggeredBy, "web")

	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleSecurityQuickScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if s.securityScanner == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Security scanner not initialized",
		})
		return
	}

	namespace := r.URL.Query().Get("namespace")
	username := r.Context().Value("username")
	triggeredBy := ""
	if username != nil {
		triggeredBy = username.(string)
	}

	result, err := s.securityScanner.QuickScan(r.Context(), namespace)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Record scan to database
	s.recordSecurityScan(result, namespace, "quick", triggeredBy, "web")

	json.NewEncoder(w).Encode(result)
}

// recordSecurityScan saves scan results to database
func (s *Server) recordSecurityScan(result *security.ScanResult, namespace, scanType, triggeredBy, source string) {
	if result == nil {
		return
	}

	// Calculate duration in milliseconds
	durationMs := int64(0)
	if result.Duration != "" {
		if d, err := time.ParseDuration(result.Duration); err == nil {
			durationMs = d.Milliseconds()
		}
	}

	// Build tools list
	tools := []string{"k13d-security-scanner"}
	if s.securityScanner.TrivyAvailable() {
		tools = append(tools, "trivy")
	}
	if s.securityScanner.KubeBenchAvailable() {
		tools = append(tools, "kube-bench")
	}

	record := db.SecurityScanRecord{
		ScanTime:     result.ScanTime,
		ClusterName:  result.ClusterName,
		Namespace:    namespace,
		ScanType:     scanType,
		DurationMs:   durationMs,
		OverallScore: result.OverallScore,
		RiskLevel:    result.RiskLevel,
		ToolsUsed:    strings.Join(tools, ","),
		TriggeredBy:  triggeredBy,
		Source:       source,
	}

	// Count issues
	if result.ImageVulns != nil {
		record.CriticalCount = result.ImageVulns.CriticalCount
		record.HighCount = result.ImageVulns.HighCount
		record.MediumCount = result.ImageVulns.MediumCount
		record.LowCount = result.ImageVulns.LowCount
	}
	record.PodIssuesCount = len(result.PodSecurityIssues)
	record.RBACIssuesCount = len(result.RBACIssues)
	record.NetworkIssuesCount = len(result.NetworkIssues)

	if result.CISBenchmark != nil {
		record.CISPassCount = result.CISBenchmark.PassCount
		record.CISFailCount = result.CISBenchmark.FailCount
		record.CISScore = result.CISBenchmark.Score
	}

	// Store full result as JSON (optional, can be large)
	if resultJSON, err := json.Marshal(result); err == nil {
		record.ScanResult = string(resultJSON)
	}

	db.RecordSecurityScan(record)
}

func (s *Server) handleSecurityScanHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	filter := db.SecurityScanFilter{
		Limit:       50,
		ClusterName: r.URL.Query().Get("cluster"),
		Namespace:   r.URL.Query().Get("namespace"),
		ScanType:    r.URL.Query().Get("type"),
		RiskLevel:   r.URL.Query().Get("risk"),
	}

	if days := r.URL.Query().Get("days"); days != "" {
		var d int
		fmt.Sscanf(days, "%d", &d)
		if d > 0 {
			filter.Since = time.Now().AddDate(0, 0, -d)
		}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		fmt.Sscanf(limit, "%d", &filter.Limit)
	}

	scans, err := db.GetSecurityScans(filter)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"scans": scans,
		"count": len(scans),
	})
}

func (s *Server) handleSecurityScanStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	clusterName := r.URL.Query().Get("cluster")
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		fmt.Sscanf(d, "%d", &days)
	}

	stats, err := db.GetSecurityScanStats(clusterName, days)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleSecurityScanDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Extract ID from path: /api/security/scan/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/security/scan/")
	var id int64
	fmt.Sscanf(path, "%d", &id)

	if id <= 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid scan ID",
		})
		return
	}

	scan, err := db.GetSecurityScanByID(id)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(scan)
}

// ==========================================
// Trivy Management Handlers
// ==========================================

func (s *Server) handleTrivyStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	downloader := security.NewTrivyDownloader()
	status := downloader.GetStatus(r.Context())

	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleTrivyInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	downloader := security.NewTrivyDownloader()

	// Check if already installed
	status := downloader.GetStatus(r.Context())
	if status.Installed {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Trivy is already installed",
			"path":    status.Path,
			"version": status.Version,
		})
		return
	}

	// Download and install
	err := downloader.Download(r.Context(), func(progress int, msg string) {
		// Progress callback - could use SSE for real-time updates
	})

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Verify installation
	newStatus := downloader.GetStatus(r.Context())
	if newStatus.Installed {
		// Update scanner with new trivy path
		if s.securityScanner != nil {
			s.securityScanner.SetTrivyPath(downloader.GetTrivyPath())
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Trivy installed successfully",
			"path":    newStatus.Path,
			"version": newStatus.Version,
		})
	} else {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Installation completed but verification failed",
		})
	}
}

func (s *Server) handleTrivyInstructions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"instructions": security.GetInstallInstructions(),
	})
}
