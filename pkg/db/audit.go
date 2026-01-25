package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ActionType categorizes the type of action
type ActionType string

const (
	ActionTypeView     ActionType = "view"     // Read-only operations (excluded from audit by default)
	ActionTypeMutation ActionType = "mutation" // Create, Update, Delete operations
	ActionTypeLLM      ActionType = "llm"      // AI/LLM tool executions
	ActionTypeAuth     ActionType = "auth"     // Authentication related
	ActionTypeConfig   ActionType = "config"   // Configuration changes
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	// Core fields
	User       string     `json:"user"`        // Web UI or TUI username
	Action     string     `json:"action"`      // Action performed (scale, delete, restart, etc.)
	Resource   string     `json:"resource"`    // Target resource (deployment/nginx, pod/xyz, etc.)
	Details    string     `json:"details"`     // Human-readable description
	ActionType ActionType `json:"action_type"` // Category of action

	// Kubernetes context
	K8sUser    string `json:"k8s_user"`    // User from kubeconfig
	K8sContext string `json:"k8s_context"` // Current kubeconfig context
	K8sCluster string `json:"k8s_cluster"` // Kubernetes cluster name
	Namespace  string `json:"namespace"`   // Target namespace

	// LLM-specific fields
	LLMRequest  string `json:"llm_request,omitempty"`  // Original user question
	LLMResponse string `json:"llm_response,omitempty"` // AI response summary
	LLMTool     string `json:"llm_tool,omitempty"`     // Tool name (kubectl, bash, etc.)
	LLMCommand  string `json:"llm_command,omitempty"`  // Actual command executed
	LLMApproved bool   `json:"llm_approved,omitempty"` // Was the command approved?

	// Source tracking
	Source    string `json:"source"` // "web", "tui", "api"
	ClientIP  string `json:"client_ip,omitempty"`
	SessionID string `json:"session_id,omitempty"`

	// Result
	Success  bool   `json:"success"`
	ErrorMsg string `json:"error_msg,omitempty"`
}

// AuditConfig controls audit behavior
type AuditConfig struct {
	IncludeViews bool   // Include view/read operations (default: false)
	FileLogPath  string // Path to .audit file (empty = disabled)
}

var (
	auditConfig = AuditConfig{
		IncludeViews: false,
		FileLogPath:  "",
	}
	auditFileMu sync.Mutex
	auditFile   *os.File
)

// InitAuditFile initializes the file-based audit log
func InitAuditFile(path string) error {
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".config", "k13d", "audit.log")
	}

	auditConfig.FileLogPath = path

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	auditFile = f
	return nil
}

// CloseAuditFile closes the audit file
func CloseAuditFile() {
	if auditFile != nil {
		auditFile.Close()
	}
}

// SetAuditConfig updates audit configuration
func SetAuditConfig(cfg AuditConfig) {
	auditConfig = cfg
}

// RecordAudit records an audit entry to both database and file
func RecordAudit(entry AuditEntry) error {
	// Skip view actions unless explicitly configured
	if entry.ActionType == ActionTypeView && !auditConfig.IncludeViews {
		return nil
	}

	// Also skip if action is literally "view" (backward compatibility)
	if strings.ToLower(entry.Action) == "view" && !auditConfig.IncludeViews {
		return nil
	}

	// Default action type for backward compatibility
	if entry.ActionType == "" {
		entry.ActionType = ActionTypeMutation
	}

	// Default success to true unless error is set
	if entry.ErrorMsg == "" {
		entry.Success = true
	}

	now := time.Now()

	// Record to database
	if DB != nil {
		query := `INSERT INTO audit_logs (
			timestamp, user, action, resource, details, action_type,
			k8s_user, k8s_context, k8s_cluster, namespace,
			llm_request, llm_response, llm_tool, llm_command, llm_approved,
			source, client_ip, session_id, success, error_msg
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		_, err := DB.Exec(query, now,
			entry.User, entry.Action, entry.Resource, entry.Details, string(entry.ActionType),
			entry.K8sUser, entry.K8sContext, entry.K8sCluster, entry.Namespace,
			entry.LLMRequest, entry.LLMResponse, entry.LLMTool, entry.LLMCommand, entry.LLMApproved,
			entry.Source, entry.ClientIP, entry.SessionID, entry.Success, entry.ErrorMsg)
		if err != nil {
			return err
		}
	}

	// Record to file
	if auditFile != nil {
		auditFileMu.Lock()
		defer auditFileMu.Unlock()

		logLine := formatAuditLogLine(now, entry)
		auditFile.WriteString(logLine + "\n")
	}

	return nil
}

// formatAuditLogLine creates a human-readable audit log line
func formatAuditLogLine(ts time.Time, entry AuditEntry) string {
	// Format: TIMESTAMP | USER (K8S_USER) | ACTION | RESOURCE | DETAILS | [LLM: TOOL COMMAND]
	user := entry.User
	if entry.K8sUser != "" && entry.K8sUser != entry.User {
		user = fmt.Sprintf("%s (k8s:%s)", entry.User, entry.K8sUser)
	}

	base := fmt.Sprintf("%s | %-20s | %-15s | %-40s | %s",
		ts.Format("2006-01-02 15:04:05"),
		truncate(user, 20),
		entry.Action,
		truncate(entry.Resource, 40),
		entry.Details)

	// Add LLM details if present
	if entry.LLMTool != "" && entry.LLMCommand != "" {
		approved := "APPROVED"
		if !entry.LLMApproved {
			approved = "DENIED"
		}
		base += fmt.Sprintf(" | [LLM %s: %s '%s']", approved, entry.LLMTool, truncate(entry.LLMCommand, 100))
	}

	// Add error if present
	if entry.ErrorMsg != "" {
		base += fmt.Sprintf(" | ERROR: %s", entry.ErrorMsg)
	}

	return base
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// GetAuditLogs retrieves audit logs with optional filters
func GetAuditLogs() ([]map[string]interface{}, error) {
	return GetAuditLogsFiltered(AuditFilter{Limit: 100})
}

// AuditFilter specifies filter criteria for audit log queries
type AuditFilter struct {
	Limit      int
	User       string
	Action     string
	ActionType ActionType
	Resource   string
	K8sUser    string
	K8sContext string
	Source     string
	OnlyLLM    bool
	OnlyErrors bool
	Since      time.Time
}

// GetAuditLogsFiltered retrieves audit logs with filters
func GetAuditLogsFiltered(filter AuditFilter) ([]map[string]interface{}, error) {
	if DB == nil {
		return nil, nil
	}

	query := `SELECT id, timestamp, user, action, resource, details, action_type,
		k8s_user, k8s_context, k8s_cluster, namespace,
		llm_request, llm_response, llm_tool, llm_command, llm_approved,
		source, client_ip, session_id, success, error_msg
		FROM audit_logs WHERE 1=1`

	var args []interface{}

	if filter.User != "" {
		query += " AND user = ?"
		args = append(args, filter.User)
	}
	if filter.Action != "" {
		query += " AND action = ?"
		args = append(args, filter.Action)
	}
	if filter.ActionType != "" {
		query += " AND action_type = ?"
		args = append(args, string(filter.ActionType))
	}
	if filter.Resource != "" {
		query += " AND resource LIKE ?"
		args = append(args, "%"+filter.Resource+"%")
	}
	if filter.K8sUser != "" {
		query += " AND k8s_user = ?"
		args = append(args, filter.K8sUser)
	}
	if filter.K8sContext != "" {
		query += " AND k8s_context = ?"
		args = append(args, filter.K8sContext)
	}
	if filter.Source != "" {
		query += " AND source = ?"
		args = append(args, filter.Source)
	}
	if filter.OnlyLLM {
		query += " AND action_type = 'llm'"
	}
	if filter.OnlyErrors {
		query += " AND success = 0"
	}
	if !filter.Since.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.Since)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	} else {
		query += " LIMIT 100"
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id int
		var ts time.Time
		var user, action, res, details, actionType string
		var k8sUser, k8sContext, k8sCluster, namespace string
		var llmReq, llmResp, llmTool, llmCmd string
		var llmApproved, success bool
		var source, clientIP, sessionID, errorMsg string

		if err := rows.Scan(&id, &ts, &user, &action, &res, &details, &actionType,
			&k8sUser, &k8sContext, &k8sCluster, &namespace,
			&llmReq, &llmResp, &llmTool, &llmCmd, &llmApproved,
			&source, &clientIP, &sessionID, &success, &errorMsg); err != nil {
			return nil, err
		}

		entry := map[string]interface{}{
			"id":          id,
			"timestamp":   ts,
			"user":        user,
			"action":      action,
			"resource":    res,
			"details":     details,
			"action_type": actionType,
			"k8s_user":    k8sUser,
			"k8s_context": k8sContext,
			"k8s_cluster": k8sCluster,
			"namespace":   namespace,
			"source":      source,
			"success":     success,
		}

		// Include LLM fields only if present
		if llmReq != "" {
			entry["llm_request"] = llmReq
		}
		if llmResp != "" {
			entry["llm_response"] = llmResp
		}
		if llmTool != "" {
			entry["llm_tool"] = llmTool
			entry["llm_command"] = llmCmd
			entry["llm_approved"] = llmApproved
		}
		if errorMsg != "" {
			entry["error_msg"] = errorMsg
		}
		if clientIP != "" {
			entry["client_ip"] = clientIP
		}

		logs = append(logs, entry)
	}
	return logs, nil
}

// GetAuditLogsJSON returns audit logs as JSON
func GetAuditLogsJSON(filter AuditFilter) ([]byte, error) {
	logs, err := GetAuditLogsFiltered(filter)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(logs, "", "  ")
}

// SecurityScanRecord represents a security scan record
type SecurityScanRecord struct {
	ID                 int64     `json:"id"`
	ScanTime           time.Time `json:"scan_time"`
	ClusterName        string    `json:"cluster_name"`
	Namespace          string    `json:"namespace"`
	ScanType           string    `json:"scan_type"` // full, quick
	DurationMs         int64     `json:"duration_ms"`
	OverallScore       float64   `json:"overall_score"`
	RiskLevel          string    `json:"risk_level"`
	ToolsUsed          string    `json:"tools_used"`
	CriticalCount      int       `json:"critical_count"`
	HighCount          int       `json:"high_count"`
	MediumCount        int       `json:"medium_count"`
	LowCount           int       `json:"low_count"`
	PodIssuesCount     int       `json:"pod_issues_count"`
	RBACIssuesCount    int       `json:"rbac_issues_count"`
	NetworkIssuesCount int       `json:"network_issues_count"`
	CISPassCount       int       `json:"cis_pass_count"`
	CISFailCount       int       `json:"cis_fail_count"`
	CISScore           float64   `json:"cis_score"`
	ScanResult         string    `json:"scan_result,omitempty"` // Full JSON result
	TriggeredBy        string    `json:"triggered_by"`
	Source             string    `json:"source"`
}

// RecordSecurityScan records a security scan to the database
func RecordSecurityScan(record SecurityScanRecord) error {
	if DB == nil {
		return nil
	}

	query := `INSERT INTO security_scans (
		scan_time, cluster_name, namespace, scan_type, duration_ms,
		overall_score, risk_level, tools_used,
		critical_count, high_count, medium_count, low_count,
		pod_issues_count, rbac_issues_count, network_issues_count,
		cis_pass_count, cis_fail_count, cis_score,
		scan_result, triggered_by, source
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := DB.Exec(query,
		record.ScanTime, record.ClusterName, record.Namespace, record.ScanType, record.DurationMs,
		record.OverallScore, record.RiskLevel, record.ToolsUsed,
		record.CriticalCount, record.HighCount, record.MediumCount, record.LowCount,
		record.PodIssuesCount, record.RBACIssuesCount, record.NetworkIssuesCount,
		record.CISPassCount, record.CISFailCount, record.CISScore,
		record.ScanResult, record.TriggeredBy, record.Source)

	return err
}

// SecurityScanFilter specifies filter criteria for security scan queries
type SecurityScanFilter struct {
	Limit       int
	ClusterName string
	Namespace   string
	ScanType    string
	RiskLevel   string
	Since       time.Time
	MinScore    float64
	MaxScore    float64
}

// GetSecurityScans retrieves security scan history with filters
func GetSecurityScans(filter SecurityScanFilter) ([]SecurityScanRecord, error) {
	if DB == nil {
		return nil, nil
	}

	query := `SELECT id, scan_time, cluster_name, namespace, scan_type, duration_ms,
		overall_score, risk_level, tools_used,
		critical_count, high_count, medium_count, low_count,
		pod_issues_count, rbac_issues_count, network_issues_count,
		cis_pass_count, cis_fail_count, cis_score,
		triggered_by, source
		FROM security_scans WHERE 1=1`

	var args []interface{}

	if filter.ClusterName != "" {
		query += " AND cluster_name = ?"
		args = append(args, filter.ClusterName)
	}
	if filter.Namespace != "" {
		query += " AND namespace = ?"
		args = append(args, filter.Namespace)
	}
	if filter.ScanType != "" {
		query += " AND scan_type = ?"
		args = append(args, filter.ScanType)
	}
	if filter.RiskLevel != "" {
		query += " AND risk_level = ?"
		args = append(args, filter.RiskLevel)
	}
	if !filter.Since.IsZero() {
		query += " AND scan_time >= ?"
		args = append(args, filter.Since)
	}
	if filter.MinScore > 0 {
		query += " AND overall_score >= ?"
		args = append(args, filter.MinScore)
	}
	if filter.MaxScore > 0 {
		query += " AND overall_score <= ?"
		args = append(args, filter.MaxScore)
	}

	query += " ORDER BY scan_time DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	} else {
		query += " LIMIT 100"
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []SecurityScanRecord
	for rows.Next() {
		var r SecurityScanRecord
		err := rows.Scan(
			&r.ID, &r.ScanTime, &r.ClusterName, &r.Namespace, &r.ScanType, &r.DurationMs,
			&r.OverallScore, &r.RiskLevel, &r.ToolsUsed,
			&r.CriticalCount, &r.HighCount, &r.MediumCount, &r.LowCount,
			&r.PodIssuesCount, &r.RBACIssuesCount, &r.NetworkIssuesCount,
			&r.CISPassCount, &r.CISFailCount, &r.CISScore,
			&r.TriggeredBy, &r.Source)
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}

	return records, nil
}

// GetSecurityScanByID retrieves a specific security scan including full results
func GetSecurityScanByID(id int64) (*SecurityScanRecord, error) {
	if DB == nil {
		return nil, nil
	}

	query := `SELECT id, scan_time, cluster_name, namespace, scan_type, duration_ms,
		overall_score, risk_level, tools_used,
		critical_count, high_count, medium_count, low_count,
		pod_issues_count, rbac_issues_count, network_issues_count,
		cis_pass_count, cis_fail_count, cis_score,
		scan_result, triggered_by, source
		FROM security_scans WHERE id = ?`

	var r SecurityScanRecord
	err := DB.QueryRow(query, id).Scan(
		&r.ID, &r.ScanTime, &r.ClusterName, &r.Namespace, &r.ScanType, &r.DurationMs,
		&r.OverallScore, &r.RiskLevel, &r.ToolsUsed,
		&r.CriticalCount, &r.HighCount, &r.MediumCount, &r.LowCount,
		&r.PodIssuesCount, &r.RBACIssuesCount, &r.NetworkIssuesCount,
		&r.CISPassCount, &r.CISFailCount, &r.CISScore,
		&r.ScanResult, &r.TriggeredBy, &r.Source)

	if err != nil {
		return nil, err
	}

	return &r, nil
}

// GetSecurityScanStats returns statistics about security scans
func GetSecurityScanStats(clusterName string, days int) (map[string]interface{}, error) {
	if DB == nil {
		return nil, nil
	}

	stats := make(map[string]interface{})

	// Get total scans
	var totalScans int
	var avgScore float64
	var latestRisk string

	since := time.Now().AddDate(0, 0, -days)

	query := `SELECT COUNT(*), COALESCE(AVG(overall_score), 0) FROM security_scans WHERE scan_time >= ?`
	args := []interface{}{since}
	if clusterName != "" {
		query += " AND cluster_name = ?"
		args = append(args, clusterName)
	}

	DB.QueryRow(query, args...).Scan(&totalScans, &avgScore)

	// Get latest risk level
	query = `SELECT risk_level FROM security_scans WHERE scan_time >= ?`
	if clusterName != "" {
		query += " AND cluster_name = ?"
	}
	query += " ORDER BY scan_time DESC LIMIT 1"
	DB.QueryRow(query, args...).Scan(&latestRisk)

	// Get risk distribution
	riskDist := make(map[string]int)
	query = `SELECT risk_level, COUNT(*) FROM security_scans WHERE scan_time >= ?`
	if clusterName != "" {
		query += " AND cluster_name = ?"
	}
	query += " GROUP BY risk_level"

	rows, err := DB.Query(query, args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var risk string
			var count int
			rows.Scan(&risk, &count)
			riskDist[risk] = count
		}
	}

	// Get score trend (last 10 scans)
	var scoreTrend []map[string]interface{}
	query = `SELECT scan_time, overall_score, risk_level FROM security_scans WHERE scan_time >= ?`
	if clusterName != "" {
		query += " AND cluster_name = ?"
	}
	query += " ORDER BY scan_time DESC LIMIT 10"

	rows, err = DB.Query(query, args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var scanTime time.Time
			var score float64
			var risk string
			rows.Scan(&scanTime, &score, &risk)
			scoreTrend = append(scoreTrend, map[string]interface{}{
				"time":       scanTime,
				"score":      score,
				"risk_level": risk,
			})
		}
	}

	stats["total_scans"] = totalScans
	stats["average_score"] = avgScore
	stats["latest_risk_level"] = latestRisk
	stats["risk_distribution"] = riskDist
	stats["score_trend"] = scoreTrend
	stats["period_days"] = days

	return stats, nil
}
