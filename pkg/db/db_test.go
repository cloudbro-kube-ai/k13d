package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit_SQLite(t *testing.T) {
	// Create a temp directory for the test database
	tmpDir, err := os.MkdirTemp("", "k13d-db-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	err = Init(dbPath)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer Close()

	if DB == nil {
		t.Error("DB should not be nil after Init")
	}

	// Verify table was created
	var tableName string
	row := DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='audit_logs'")
	err = row.Scan(&tableName)
	if err != nil {
		t.Errorf("audit_logs table not created: %v", err)
	}
}

func TestInitWithConfig_SQLite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k13d-db-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := DBConfig{
		Type: DBTypeSQLite,
		Path: filepath.Join(tmpDir, "config_test.db"),
	}

	err = InitWithConfig(cfg)
	if err != nil {
		t.Fatalf("InitWithConfig() error = %v", err)
	}
	defer Close()

	if GetDBType() != DBTypeSQLite {
		t.Errorf("GetDBType() = %v, want %v", GetDBType(), DBTypeSQLite)
	}
}

func TestInitWithConfig_DefaultType(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k13d-db-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := DBConfig{
		Type: "", // Empty type should default to SQLite
		Path: filepath.Join(tmpDir, "default_test.db"),
	}

	err = InitWithConfig(cfg)
	if err != nil {
		t.Fatalf("InitWithConfig() error = %v", err)
	}
	defer Close()

	if GetDBType() != DBTypeSQLite {
		t.Errorf("GetDBType() = %v, want %v", GetDBType(), DBTypeSQLite)
	}
}

func TestInitWithConfig_UnsupportedType(t *testing.T) {
	cfg := DBConfig{
		Type: "unsupported",
	}

	err := InitWithConfig(cfg)
	if err == nil {
		t.Error("InitWithConfig() should return error for unsupported type")
		Close()
	}
}

func TestDBConfig_Fields(t *testing.T) {
	cfg := DBConfig{
		Type:     DBTypePostgres,
		Host:     "localhost",
		Port:     5432,
		Database: "k13d",
		Username: "user",
		Password: "password",
		SSLMode:  "require",
		Path:     "/path/to/db",
	}

	if cfg.Type != DBTypePostgres {
		t.Errorf("Type = %v, want %v", cfg.Type, DBTypePostgres)
	}
	if cfg.Host != "localhost" {
		t.Errorf("Host = %v, want localhost", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("Port = %v, want 5432", cfg.Port)
	}
	if cfg.Database != "k13d" {
		t.Errorf("Database = %v, want k13d", cfg.Database)
	}
	if cfg.Username != "user" {
		t.Errorf("Username = %v, want user", cfg.Username)
	}
	if cfg.SSLMode != "require" {
		t.Errorf("SSLMode = %v, want require", cfg.SSLMode)
	}
}

func TestDBType_Constants(t *testing.T) {
	tests := []struct {
		dbType   DBType
		expected string
	}{
		{DBTypeSQLite, "sqlite"},
		{DBTypePostgres, "postgres"},
		{DBTypeMariaDB, "mariadb"},
		{DBTypeMySQL, "mysql"},
	}

	for _, tt := range tests {
		if string(tt.dbType) != tt.expected {
			t.Errorf("DBType %v = %s, want %s", tt.dbType, string(tt.dbType), tt.expected)
		}
	}
}

func TestClose_NilDB(t *testing.T) {
	// Save current DB
	savedDB := DB
	DB = nil

	err := Close()
	if err != nil {
		t.Errorf("Close() with nil DB should not error: %v", err)
	}

	// Restore DB
	DB = savedDB
}

func TestCreateTables_SQLite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k13d-db-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = Init(filepath.Join(tmpDir, "tables_test.db"))
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer Close()

	// Verify audit_logs table
	rows, err := DB.Query("PRAGMA table_info(audit_logs)")
	if err != nil {
		t.Fatalf("Failed to get table info: %v", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt interface{}
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		columns[name] = true
	}

	expectedColumns := []string{
		"id", "timestamp", "user", "action", "resource", "details",
		"action_type", "k8s_user", "k8s_context", "k8s_cluster",
		"namespace", "llm_request", "llm_response", "llm_tool",
		"llm_command", "llm_approved", "source", "client_ip",
		"session_id", "success", "error_msg",
	}

	for _, col := range expectedColumns {
		if !columns[col] {
			t.Errorf("Column %s not found in audit_logs table", col)
		}
	}

	// Verify security_scans table
	rows2, err := DB.Query("PRAGMA table_info(security_scans)")
	if err != nil {
		t.Fatalf("Failed to get security_scans table info: %v", err)
	}
	defer rows2.Close()

	securityColumns := make(map[string]bool)
	for rows2.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt interface{}
		if err := rows2.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		securityColumns[name] = true
	}

	expectedSecurityColumns := []string{
		"id", "scan_time", "cluster_name", "namespace", "scan_type",
		"duration_ms", "overall_score", "risk_level", "tools_used",
		"critical_count", "high_count", "medium_count", "low_count",
	}

	for _, col := range expectedSecurityColumns {
		if !securityColumns[col] {
			t.Errorf("Column %s not found in security_scans table", col)
		}
	}
}

func TestGetIndexQueries(t *testing.T) {
	// Test SQLite indexes
	currentDBType = DBTypeSQLite
	sqliteIndexes := getIndexQueries()
	if len(sqliteIndexes) == 0 {
		t.Error("SQLite should have indexes")
	}

	// Test Postgres indexes
	currentDBType = DBTypePostgres
	pgIndexes := getIndexQueries()
	if len(pgIndexes) == 0 {
		t.Error("Postgres should have indexes")
	}

	// Test MySQL indexes
	currentDBType = DBTypeMySQL
	mysqlIndexes := getIndexQueries()
	if len(mysqlIndexes) == 0 {
		t.Error("MySQL should have indexes")
	}

	// Reset to SQLite
	currentDBType = DBTypeSQLite
}

func TestInit_DefaultPath(t *testing.T) {
	// Test that empty path uses default
	// Note: This modifies the actual config directory, so skip in CI
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that modifies config directory in CI")
	}

	err := Init("")
	if err != nil {
		// May fail due to permissions, that's OK
		t.Logf("Init with default path: %v (may be expected)", err)
		return
	}
	defer Close()

	if DB == nil {
		t.Error("DB should not be nil after Init with default path")
	}
}

func TestMigrateAuditLogsTable(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k13d-db-migrate-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "migrate_test.db")

	// First, create an old-style table with fewer columns
	err = Init(dbPath)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Drop the auto-created table and create a minimal one
	_, err = DB.Exec("DROP TABLE IF EXISTS audit_logs")
	if err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}

	// Create old-style table with fewer columns
	_, err = DB.Exec(`
		CREATE TABLE audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			user TEXT,
			action TEXT,
			resource TEXT,
			details TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create old table: %v", err)
	}

	// Insert a test record with old schema
	_, err = DB.Exec(`INSERT INTO audit_logs (user, action, resource, details) VALUES (?, ?, ?, ?)`,
		"testuser", "scale", "deployment/nginx", "scaled to 3 replicas")
	if err != nil {
		t.Fatalf("Failed to insert test record: %v", err)
	}

	Close()

	// Re-initialize - this should trigger migration
	err = Init(dbPath)
	if err != nil {
		t.Fatalf("Re-Init() error = %v", err)
	}
	defer Close()

	// Verify new columns exist by querying them
	rows, err := DB.Query("PRAGMA table_info(audit_logs)")
	if err != nil {
		t.Fatalf("Failed to get table info: %v", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt interface{}
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		columns[name] = true
	}

	// Check that new columns were added
	newColumns := []string{"action_type", "k8s_user", "namespace", "source", "success"}
	for _, col := range newColumns {
		if !columns[col] {
			t.Errorf("Migration should have added column %s", col)
		}
	}

	// Verify old data is preserved
	var count int
	err = DB.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE user = 'testuser'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count records: %v", err)
	}
	if count != 1 {
		t.Errorf("Old record should be preserved, got count = %d", count)
	}
}

func TestRecordAudit_FullFields(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k13d-db-audit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = Init(filepath.Join(tmpDir, "audit_full_test.db"))
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer Close()

	// Record an audit entry with all fields
	entry := AuditEntry{
		User:        "admin",
		Action:      "delete",
		Resource:    "pod/nginx-abc123",
		Details:     "Deleted pod in default namespace",
		ActionType:  ActionTypeMutation,
		K8sUser:     "kubernetes-admin",
		K8sContext:  "minikube",
		K8sCluster:  "minikube-cluster",
		Namespace:   "default",
		LLMRequest:  "Delete the nginx pod",
		LLMResponse: "I will delete the pod nginx-abc123",
		LLMTool:     "kubectl",
		LLMCommand:  "kubectl delete pod nginx-abc123",
		LLMApproved: true,
		Source:      "web",
		ClientIP:    "127.0.0.1",
		SessionID:   "session-123",
		Success:     true,
		ErrorMsg:    "",
	}

	err = RecordAudit(entry)
	if err != nil {
		t.Fatalf("RecordAudit() error = %v", err)
	}

	// Query and verify
	logs, err := GetAuditLogsFiltered(AuditFilter{Limit: 10})
	if err != nil {
		t.Fatalf("GetAuditLogsFiltered() error = %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(logs))
	}

	log := logs[0]
	if log["user"] != "admin" {
		t.Errorf("user = %v, want admin", log["user"])
	}
	if log["action"] != "delete" {
		t.Errorf("action = %v, want delete", log["action"])
	}
	if log["k8s_user"] != "kubernetes-admin" {
		t.Errorf("k8s_user = %v, want kubernetes-admin", log["k8s_user"])
	}
	if log["namespace"] != "default" {
		t.Errorf("namespace = %v, want default", log["namespace"])
	}
	if log["source"] != "web" {
		t.Errorf("source = %v, want web", log["source"])
	}
	if log["llm_tool"] != "kubectl" {
		t.Errorf("llm_tool = %v, want kubectl", log["llm_tool"])
	}
}

func TestGetAuditLogsFiltered_Filters(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k13d-db-filter-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = Init(filepath.Join(tmpDir, "filter_test.db"))
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer Close()

	// Insert multiple entries with different properties
	entries := []AuditEntry{
		{User: "admin", Action: "delete", Resource: "pod/nginx", ActionType: ActionTypeMutation, Source: "web", Success: true},
		{User: "admin", Action: "scale", Resource: "deployment/app", ActionType: ActionTypeMutation, Source: "tui", Success: true},
		{User: "user1", Action: "ask", Resource: "pod/nginx", ActionType: ActionTypeLLM, Source: "web", Success: true, LLMTool: "kubectl"},
		{User: "user1", Action: "delete", Resource: "service/api", ActionType: ActionTypeMutation, Source: "web", Success: false, ErrorMsg: "permission denied"},
	}

	for _, e := range entries {
		if err := RecordAudit(e); err != nil {
			t.Fatalf("RecordAudit() error = %v", err)
		}
	}

	// Test filter by user
	logs, err := GetAuditLogsFiltered(AuditFilter{User: "admin"})
	if err != nil {
		t.Fatalf("Filter by user error: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("Filter by user 'admin': got %d, want 2", len(logs))
	}

	// Test filter by action
	logs, err = GetAuditLogsFiltered(AuditFilter{Action: "delete"})
	if err != nil {
		t.Fatalf("Filter by action error: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("Filter by action 'delete': got %d, want 2", len(logs))
	}

	// Test filter by source
	logs, err = GetAuditLogsFiltered(AuditFilter{Source: "web"})
	if err != nil {
		t.Fatalf("Filter by source error: %v", err)
	}
	if len(logs) != 3 {
		t.Errorf("Filter by source 'web': got %d, want 3", len(logs))
	}

	// Test OnlyLLM filter
	logs, err = GetAuditLogsFiltered(AuditFilter{OnlyLLM: true})
	if err != nil {
		t.Fatalf("Filter OnlyLLM error: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("Filter OnlyLLM: got %d, want 1", len(logs))
	}

	// Test OnlyErrors filter
	logs, err = GetAuditLogsFiltered(AuditFilter{OnlyErrors: true})
	if err != nil {
		t.Fatalf("Filter OnlyErrors error: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("Filter OnlyErrors: got %d, want 1", len(logs))
	}

	// Test resource partial match
	logs, err = GetAuditLogsFiltered(AuditFilter{Resource: "nginx"})
	if err != nil {
		t.Fatalf("Filter by resource error: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("Filter by resource 'nginx': got %d, want 2", len(logs))
	}

	// Test limit
	logs, err = GetAuditLogsFiltered(AuditFilter{Limit: 2})
	if err != nil {
		t.Fatalf("Filter with limit error: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("Filter with limit 2: got %d, want 2", len(logs))
	}
}

func TestGetAuditLogs_NilDB(t *testing.T) {
	// Save and clear DB
	savedDB := DB
	DB = nil

	logs, err := GetAuditLogsFiltered(AuditFilter{})
	if err != nil {
		t.Errorf("GetAuditLogsFiltered with nil DB should not error: %v", err)
	}
	if logs != nil {
		t.Errorf("GetAuditLogsFiltered with nil DB should return nil, got %v", logs)
	}

	// Restore DB
	DB = savedDB
}

func TestRecordAudit_SkipsViewActions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k13d-db-view-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = Init(filepath.Join(tmpDir, "view_test.db"))
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer Close()

	// Ensure view actions are not included by default
	SetAuditConfig(AuditConfig{IncludeViews: false})

	// Record a view action
	err = RecordAudit(AuditEntry{
		User:       "admin",
		Action:     "view",
		Resource:   "pod/nginx",
		ActionType: ActionTypeView,
	})
	if err != nil {
		t.Fatalf("RecordAudit() error = %v", err)
	}

	// Record a mutation action
	err = RecordAudit(AuditEntry{
		User:       "admin",
		Action:     "delete",
		Resource:   "pod/nginx",
		ActionType: ActionTypeMutation,
	})
	if err != nil {
		t.Fatalf("RecordAudit() error = %v", err)
	}

	// Only mutation should be recorded
	logs, err := GetAuditLogsFiltered(AuditFilter{})
	if err != nil {
		t.Fatalf("GetAuditLogsFiltered() error = %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("Expected 1 log (view should be skipped), got %d", len(logs))
	}
}
