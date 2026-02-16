package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuditLogging(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_audit.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer Close()

	entry := AuditEntry{
		User:    "Tester",
		Action:  "TEST_ACTION",
		Details: "Test Details",
	}

	if err := RecordAudit(entry); err != nil {
		t.Fatalf("Failed to record audit: %v", err)
	}

	logs, err := GetAuditLogs()
	if err != nil {
		t.Fatalf("Failed to get logs: %v", err)
	}

	if len(logs) == 0 {
		t.Errorf("Expected logs, got none")
	}

	found := false
	for _, log := range logs {
		if log["action"] == "TEST_ACTION" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Test action not found in logs")
	}
}

func TestAuditEntryFields(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_audit_fields.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer Close()

	entry := AuditEntry{
		User:       "admin",
		Action:     "scale",
		Resource:   "deployment/nginx",
		Details:    "Scaled from 2 to 5 replicas",
		ActionType: ActionTypeMutation,
		K8sUser:    "k8s-admin",
		K8sContext: "production",
		K8sCluster: "prod-cluster",
		Namespace:  "default",
		Source:     "web",
		ClientIP:   "192.168.1.1",
		SessionID:  "session-123",
		Success:    true,
	}

	if err := RecordAudit(entry); err != nil {
		t.Fatalf("Failed to record audit: %v", err)
	}

	logs, err := GetAuditLogs()
	if err != nil {
		t.Fatalf("Failed to get logs: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("Expected at least one log entry")
	}

	log := logs[0]
	if log["user"] != "admin" {
		t.Errorf("user = %v, want admin", log["user"])
	}
	if log["action"] != "scale" {
		t.Errorf("action = %v, want scale", log["action"])
	}
	if log["resource"] != "deployment/nginx" {
		t.Errorf("resource = %v, want deployment/nginx", log["resource"])
	}
	if log["k8s_user"] != "k8s-admin" {
		t.Errorf("k8s_user = %v, want k8s-admin", log["k8s_user"])
	}
	if log["namespace"] != "default" {
		t.Errorf("namespace = %v, want default", log["namespace"])
	}
	if log["source"] != "web" {
		t.Errorf("source = %v, want web", log["source"])
	}
}

func TestAuditLLMFields(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_audit_llm.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer Close()

	entry := AuditEntry{
		User:        "user1",
		Action:      "llm_execute",
		Resource:    "ai_assistant",
		Details:     "AI executed kubectl command",
		ActionType:  ActionTypeLLM,
		LLMRequest:  "Show me all running pods",
		LLMResponse: "Found 10 running pods",
		LLMTool:     "kubectl",
		LLMCommand:  "kubectl get pods --all-namespaces",
		LLMApproved: true,
		Source:      "web",
	}

	if err := RecordAudit(entry); err != nil {
		t.Fatalf("Failed to record audit: %v", err)
	}

	logs, err := GetAuditLogsFiltered(AuditFilter{OnlyLLM: true})
	if err != nil {
		t.Fatalf("Failed to get LLM logs: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("Expected at least one LLM log entry")
	}

	log := logs[0]
	if log["llm_tool"] != "kubectl" {
		t.Errorf("llm_tool = %v, want kubectl", log["llm_tool"])
	}
	if log["llm_command"] != "kubectl get pods --all-namespaces" {
		t.Errorf("llm_command = %v, want kubectl get pods --all-namespaces", log["llm_command"])
	}
	if log["llm_approved"] != true {
		t.Errorf("llm_approved = %v, want true", log["llm_approved"])
	}
}

func TestAuditFilteredQueries(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_audit_filters.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer Close()

	// Create various audit entries
	entries := []AuditEntry{
		{User: "admin", Action: "scale", Resource: "deployment/nginx", ActionType: ActionTypeMutation, Source: "web", Namespace: "default"},
		{User: "admin", Action: "delete", Resource: "pod/test-pod", ActionType: ActionTypeMutation, Source: "web", Namespace: "default", ErrorMsg: "permission denied", Success: false},
		{User: "user1", Action: "llm_execute", Resource: "ai", ActionType: ActionTypeLLM, Source: "tui", LLMTool: "kubectl"},
		{User: "user2", Action: "login", Resource: "auth", ActionType: ActionTypeAuth, Source: "web"},
		{User: "admin", Action: "update_config", Resource: "settings", ActionType: ActionTypeConfig, Source: "web"},
	}

	for _, e := range entries {
		if err := RecordAudit(e); err != nil {
			t.Fatalf("Failed to record audit: %v", err)
		}
	}

	tests := []struct {
		name     string
		filter   AuditFilter
		expected int
	}{
		{
			name:     "Filter by user",
			filter:   AuditFilter{User: "admin"},
			expected: 3,
		},
		{
			name:     "Filter by action",
			filter:   AuditFilter{Action: "scale"},
			expected: 1,
		},
		{
			name:     "Filter by action type",
			filter:   AuditFilter{ActionType: ActionTypeMutation},
			expected: 2,
		},
		{
			name:     "Filter by source",
			filter:   AuditFilter{Source: "web"},
			expected: 4,
		},
		{
			name:     "Filter only LLM",
			filter:   AuditFilter{OnlyLLM: true},
			expected: 1,
		},
		{
			name:     "Filter only errors",
			filter:   AuditFilter{OnlyErrors: true},
			expected: 1,
		},
		{
			name:     "Filter by resource pattern",
			filter:   AuditFilter{Resource: "deployment"},
			expected: 1,
		},
		{
			name:     "Limit results",
			filter:   AuditFilter{Limit: 2},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs, err := GetAuditLogsFiltered(tt.filter)
			if err != nil {
				t.Fatalf("GetAuditLogsFiltered() error = %v", err)
			}
			if len(logs) != tt.expected {
				t.Errorf("len(logs) = %d, want %d", len(logs), tt.expected)
			}
		})
	}
}

func TestAuditSkipViewActions(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_audit_views.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer Close()

	// Set config to not include views
	SetAuditConfig(AuditConfig{IncludeViews: false})

	// Record a view action
	viewEntry := AuditEntry{
		User:       "user1",
		Action:     "view",
		Resource:   "pods",
		ActionType: ActionTypeView,
	}
	if err := RecordAudit(viewEntry); err != nil {
		t.Fatalf("Failed to record audit: %v", err)
	}

	logs, err := GetAuditLogs()
	if err != nil {
		t.Fatalf("Failed to get logs: %v", err)
	}

	// View action should be skipped
	for _, log := range logs {
		if log["action"] == "view" {
			t.Error("View action should be skipped when IncludeViews is false")
		}
	}

	// Now enable views
	SetAuditConfig(AuditConfig{IncludeViews: true})

	viewEntry.Action = "view_with_include"
	if err := RecordAudit(viewEntry); err != nil {
		t.Fatalf("Failed to record audit: %v", err)
	}

	logs, err = GetAuditLogs()
	if err != nil {
		t.Fatalf("Failed to get logs: %v", err)
	}

	found := false
	for _, log := range logs {
		if log["action"] == "view_with_include" {
			found = true
			break
		}
	}
	if !found {
		t.Error("View action should be included when IncludeViews is true")
	}
}

func TestAuditLogsJSON(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_audit_json.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer Close()

	entry := AuditEntry{
		User:       "user1",
		Action:     "test",
		Resource:   "test-resource",
		ActionType: ActionTypeMutation,
	}
	if err := RecordAudit(entry); err != nil {
		t.Fatalf("Failed to record audit: %v", err)
	}

	jsonData, err := GetAuditLogsJSON(AuditFilter{Limit: 10})
	if err != nil {
		t.Fatalf("GetAuditLogsJSON() error = %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Expected non-empty JSON output")
	}

	// Basic JSON structure check
	if jsonData[0] != '[' {
		t.Error("Expected JSON array")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
		{"this is a long string", 10, "this is..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestFormatAuditLogLine(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		entry    AuditEntry
		contains []string
	}{
		{
			name: "Basic entry",
			entry: AuditEntry{
				User:     "admin",
				Action:   "scale",
				Resource: "deployment/nginx",
				Details:  "Scaled to 5 replicas",
			},
			contains: []string{"2024-01-15 10:30:00", "admin", "scale", "deployment/nginx", "Scaled to 5 replicas"},
		},
		{
			name: "Entry with K8s user",
			entry: AuditEntry{
				User:     "admin",
				K8sUser:  "k8s-admin",
				Action:   "delete",
				Resource: "pod/test",
				Details:  "Deleted pod",
			},
			contains: []string{"admin (k8s:", "delete"},
		},
		{
			name: "Entry with LLM details",
			entry: AuditEntry{
				User:        "user1",
				Action:      "llm",
				Resource:    "ai",
				Details:     "AI command",
				LLMTool:     "kubectl",
				LLMCommand:  "kubectl get pods",
				LLMApproved: true,
			},
			contains: []string{"LLM APPROVED", "kubectl", "kubectl get pods"},
		},
		{
			name: "Entry with error",
			entry: AuditEntry{
				User:     "user1",
				Action:   "create",
				Resource: "deployment",
				Details:  "Failed to create",
				ErrorMsg: "permission denied",
			},
			contains: []string{"ERROR:", "permission denied"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAuditLogLine(ts, tt.entry)
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("formatAuditLogLine() = %q, should contain %q", result, substr)
				}
			}
		})
	}
}

func TestActionTypes(t *testing.T) {
	tests := []struct {
		actionType ActionType
		expected   string
	}{
		{ActionTypeView, "view"},
		{ActionTypeMutation, "mutation"},
		{ActionTypeLLM, "llm"},
		{ActionTypeAuth, "auth"},
		{ActionTypeConfig, "config"},
	}

	for _, tt := range tests {
		t.Run(string(tt.actionType), func(t *testing.T) {
			if string(tt.actionType) != tt.expected {
				t.Errorf("ActionType = %s, want %s", tt.actionType, tt.expected)
			}
		})
	}
}

func TestAuditFileOperations(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	auditPath := filepath.Join(tmpDir, "audit.log")

	if err := InitAuditFile(auditPath); err != nil {
		t.Fatalf("InitAuditFile() error = %v", err)
	}
	defer CloseAuditFile()

	// Record an entry (this writes to file)
	dbPath := filepath.Join(t.TempDir(), "test_audit_file.db")
	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer Close()

	entry := AuditEntry{
		User:       "file-user",
		Action:     "file-action",
		Resource:   "file-resource",
		ActionType: ActionTypeMutation,
	}
	if err := RecordAudit(entry); err != nil {
		t.Fatalf("RecordAudit() error = %v", err)
	}

	// Verify file was written
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("Failed to read audit file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Audit file should not be empty")
	}

	if !strings.Contains(string(data), "file-action") {
		t.Error("Audit file should contain 'file-action'")
	}
}

func TestSecurityScanRecord(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_security_scan.db")
	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer Close()

	record := SecurityScanRecord{
		ScanTime:           time.Now(),
		ClusterName:        "test-cluster",
		Namespace:          "default",
		ScanType:           "full",
		DurationMs:         5000,
		OverallScore:       75.5,
		RiskLevel:          "medium",
		ToolsUsed:          "trivy,kube-bench",
		CriticalCount:      0,
		HighCount:          2,
		MediumCount:        5,
		LowCount:           10,
		PodIssuesCount:     3,
		RBACIssuesCount:    2,
		NetworkIssuesCount: 1,
		CISPassCount:       80,
		CISFailCount:       20,
		CISScore:           80.0,
		TriggeredBy:        "admin",
		Source:             "web",
	}

	if err := RecordSecurityScan(record); err != nil {
		t.Fatalf("RecordSecurityScan() error = %v", err)
	}

	// Retrieve scans
	scans, err := GetSecurityScans(SecurityScanFilter{Limit: 10})
	if err != nil {
		t.Fatalf("GetSecurityScans() error = %v", err)
	}

	if len(scans) == 0 {
		t.Fatal("Expected at least one scan record")
	}

	scan := scans[0]
	if scan.ClusterName != "test-cluster" {
		t.Errorf("ClusterName = %s, want test-cluster", scan.ClusterName)
	}
	if scan.OverallScore != 75.5 {
		t.Errorf("OverallScore = %f, want 75.5", scan.OverallScore)
	}
	if scan.RiskLevel != "medium" {
		t.Errorf("RiskLevel = %s, want medium", scan.RiskLevel)
	}
}

func TestSecurityScanFilters(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_security_filters.db")
	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer Close()

	// Create multiple scan records
	records := []SecurityScanRecord{
		{ScanTime: time.Now(), ClusterName: "cluster-a", Namespace: "default", ScanType: "full", OverallScore: 80, RiskLevel: "low", TriggeredBy: "admin", Source: "web"},
		{ScanTime: time.Now(), ClusterName: "cluster-a", Namespace: "kube-system", ScanType: "quick", OverallScore: 60, RiskLevel: "medium", TriggeredBy: "user1", Source: "api"},
		{ScanTime: time.Now(), ClusterName: "cluster-b", Namespace: "default", ScanType: "full", OverallScore: 40, RiskLevel: "high", TriggeredBy: "admin", Source: "web"},
	}

	for _, r := range records {
		if err := RecordSecurityScan(r); err != nil {
			t.Fatalf("RecordSecurityScan() error = %v", err)
		}
	}

	tests := []struct {
		name     string
		filter   SecurityScanFilter
		expected int
	}{
		{
			name:     "Filter by cluster",
			filter:   SecurityScanFilter{ClusterName: "cluster-a"},
			expected: 2,
		},
		{
			name:     "Filter by namespace",
			filter:   SecurityScanFilter{Namespace: "default"},
			expected: 2,
		},
		{
			name:     "Filter by scan type",
			filter:   SecurityScanFilter{ScanType: "full"},
			expected: 2,
		},
		{
			name:     "Filter by risk level",
			filter:   SecurityScanFilter{RiskLevel: "high"},
			expected: 1,
		},
		{
			name:     "Filter by min score",
			filter:   SecurityScanFilter{MinScore: 70},
			expected: 1,
		},
		{
			name:     "Filter by max score",
			filter:   SecurityScanFilter{MaxScore: 50},
			expected: 1,
		},
		{
			name:     "Limit results",
			filter:   SecurityScanFilter{Limit: 2},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scans, err := GetSecurityScans(tt.filter)
			if err != nil {
				t.Fatalf("GetSecurityScans() error = %v", err)
			}
			if len(scans) != tt.expected {
				t.Errorf("len(scans) = %d, want %d", len(scans), tt.expected)
			}
		})
	}
}

func TestSecurityScanByID(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_scan_by_id.db")
	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer Close()

	record := SecurityScanRecord{
		ScanTime:     time.Now(),
		ClusterName:  "test-cluster",
		Namespace:    "default",
		ScanType:     "full",
		OverallScore: 85,
		RiskLevel:    "low",
		ScanResult:   `{"issues": []}`,
		TriggeredBy:  "admin",
		Source:       "web",
	}

	if err := RecordSecurityScan(record); err != nil {
		t.Fatalf("RecordSecurityScan() error = %v", err)
	}

	// Get all scans to find the ID
	scans, _ := GetSecurityScans(SecurityScanFilter{Limit: 1})
	if len(scans) == 0 {
		t.Fatal("Expected at least one scan")
	}

	scan, err := GetSecurityScanByID(scans[0].ID)
	if err != nil {
		t.Fatalf("GetSecurityScanByID() error = %v", err)
	}

	if scan == nil {
		t.Fatal("Expected scan, got nil")
	}

	if scan.ClusterName != "test-cluster" {
		t.Errorf("ClusterName = %s, want test-cluster", scan.ClusterName)
	}
}

func TestSecurityScanStats(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_scan_stats.db")
	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer Close()

	// Create multiple scan records
	for i := 0; i < 5; i++ {
		record := SecurityScanRecord{
			ScanTime:     time.Now(),
			ClusterName:  "stats-cluster",
			Namespace:    "default",
			ScanType:     "full",
			OverallScore: float64(60 + i*10),
			RiskLevel:    []string{"low", "low", "medium", "medium", "high"}[i],
			TriggeredBy:  "admin",
			Source:       "web",
		}
		if err := RecordSecurityScan(record); err != nil {
			t.Fatalf("RecordSecurityScan() error = %v", err)
		}
	}

	stats, err := GetSecurityScanStats("stats-cluster", 30)
	if err != nil {
		t.Fatalf("GetSecurityScanStats() error = %v", err)
	}

	if stats == nil {
		t.Fatal("Expected stats, got nil")
	}

	if stats["total_scans"].(int) != 5 {
		t.Errorf("total_scans = %v, want 5", stats["total_scans"])
	}

	if stats["period_days"].(int) != 30 {
		t.Errorf("period_days = %v, want 30", stats["period_days"])
	}

	riskDist := stats["risk_distribution"].(map[string]int)
	if riskDist["low"] != 2 {
		t.Errorf("risk_distribution[low] = %d, want 2", riskDist["low"])
	}
}

func TestNilDatabaseHandling(t *testing.T) {
	// Test with nil database
	originalDB := DB
	DB = nil
	defer func() { DB = originalDB }()

	// These should not panic with nil DB
	logs, err := GetAuditLogs()
	if err != nil {
		t.Errorf("GetAuditLogs() with nil DB should not error, got %v", err)
	}
	if logs != nil {
		t.Error("GetAuditLogs() with nil DB should return nil")
	}

	scans, err := GetSecurityScans(SecurityScanFilter{})
	if err != nil {
		t.Errorf("GetSecurityScans() with nil DB should not error, got %v", err)
	}
	if scans != nil {
		t.Error("GetSecurityScans() with nil DB should return nil")
	}

	// RecordAudit with nil DB should not panic
	err = RecordAudit(AuditEntry{User: "test", Action: "test"})
	if err != nil {
		t.Errorf("RecordAudit() with nil DB should not error, got %v", err)
	}

	// RecordSecurityScan with nil DB should not panic
	err = RecordSecurityScan(SecurityScanRecord{})
	if err != nil {
		t.Errorf("RecordSecurityScan() with nil DB should not error, got %v", err)
	}
}
