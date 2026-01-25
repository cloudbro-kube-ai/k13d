package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

// DBType represents the database type
type DBType string

const (
	DBTypeSQLite   DBType = "sqlite"
	DBTypePostgres DBType = "postgres"
	DBTypeMariaDB  DBType = "mariadb"
	DBTypeMySQL    DBType = "mysql"
)

// Current database type
var currentDBType DBType = DBTypeSQLite

// DBConfig holds database configuration
type DBConfig struct {
	Type     DBType `json:"type"`     // sqlite, postgres, mariadb, mysql
	Host     string `json:"host"`     // Database host
	Port     int    `json:"port"`     // Database port
	Database string `json:"database"` // Database name
	Username string `json:"username"` // Database username
	Password string `json:"password"` // Database password
	SSLMode  string `json:"sslMode"`  // SSL mode (for postgres)
	Path     string `json:"path"`     // SQLite file path
}

// Init initializes database with SQLite (backward compatible)
func Init(dbPath string) error {
	return InitWithConfig(DBConfig{
		Type: DBTypeSQLite,
		Path: dbPath,
	})
}

// InitWithConfig initializes database with configuration
func InitWithConfig(cfg DBConfig) error {
	var db *sql.DB
	var err error

	currentDBType = cfg.Type
	if currentDBType == "" {
		currentDBType = DBTypeSQLite
	}

	switch currentDBType {
	case DBTypeSQLite:
		db, err = initSQLite(cfg.Path)
	case DBTypePostgres:
		db, err = initPostgres(cfg)
	case DBTypeMariaDB, DBTypeMySQL:
		db, err = initMySQL(cfg)
	default:
		return fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	DB = db
	return createTables()
}

func initSQLite(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		home, _ := os.UserHomeDir()
		dbPath = filepath.Join(home, ".config", "k13d", "audit.db")
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return sql.Open("sqlite", dbPath)
}

func initPostgres(cfg DBConfig) (*sql.DB, error) {
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Database, sslMode)

	return sql.Open("postgres", dsn)
}

func initMySQL(cfg DBConfig) (*sql.DB, error) {
	// Format: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	return sql.Open("mysql", dsn)
}

// GetDBType returns the current database type
func GetDBType() DBType {
	return currentDBType
}

func createTables() error {
	// Migrate existing tables first
	if err := migrateAuditLogsTable(); err != nil {
		// Log but don't fail - we'll create new table if needed
		fmt.Printf("Warning: migration check failed: %v\n", err)
	}

	// Create audit_logs table (SQL syntax varies by DB type)
	var auditQuery string
	switch currentDBType {
	case DBTypePostgres:
		auditQuery = `
		CREATE TABLE IF NOT EXISTS audit_logs (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			"user" VARCHAR(255),
			action VARCHAR(255),
			resource VARCHAR(512),
			details TEXT,
			action_type VARCHAR(50) DEFAULT 'mutation',
			k8s_user VARCHAR(255) DEFAULT '',
			k8s_context VARCHAR(255) DEFAULT '',
			k8s_cluster VARCHAR(255) DEFAULT '',
			namespace VARCHAR(255) DEFAULT '',
			llm_request TEXT DEFAULT '',
			llm_response TEXT DEFAULT '',
			llm_tool VARCHAR(100) DEFAULT '',
			llm_command TEXT DEFAULT '',
			llm_approved BOOLEAN DEFAULT FALSE,
			source VARCHAR(50) DEFAULT '',
			client_ip VARCHAR(50) DEFAULT '',
			session_id VARCHAR(255) DEFAULT '',
			success BOOLEAN DEFAULT TRUE,
			error_msg TEXT DEFAULT ''
		);`
	case DBTypeMariaDB, DBTypeMySQL:
		auditQuery = `
		CREATE TABLE IF NOT EXISTS audit_logs (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			user VARCHAR(255),
			action VARCHAR(255),
			resource VARCHAR(512),
			details TEXT,
			action_type VARCHAR(50) DEFAULT 'mutation',
			k8s_user VARCHAR(255) DEFAULT '',
			k8s_context VARCHAR(255) DEFAULT '',
			k8s_cluster VARCHAR(255) DEFAULT '',
			namespace VARCHAR(255) DEFAULT '',
			llm_request TEXT,
			llm_response TEXT,
			llm_tool VARCHAR(100) DEFAULT '',
			llm_command TEXT,
			llm_approved TINYINT(1) DEFAULT 0,
			source VARCHAR(50) DEFAULT '',
			client_ip VARCHAR(50) DEFAULT '',
			session_id VARCHAR(255) DEFAULT '',
			success TINYINT(1) DEFAULT 1,
			error_msg TEXT
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
	default: // SQLite
		auditQuery = `
		CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			user TEXT,
			action TEXT,
			resource TEXT,
			details TEXT,
			action_type TEXT DEFAULT 'mutation',
			k8s_user TEXT DEFAULT '',
			k8s_context TEXT DEFAULT '',
			k8s_cluster TEXT DEFAULT '',
			namespace TEXT DEFAULT '',
			llm_request TEXT DEFAULT '',
			llm_response TEXT DEFAULT '',
			llm_tool TEXT DEFAULT '',
			llm_command TEXT DEFAULT '',
			llm_approved INTEGER DEFAULT 0,
			source TEXT DEFAULT '',
			client_ip TEXT DEFAULT '',
			session_id TEXT DEFAULT '',
			success INTEGER DEFAULT 1,
			error_msg TEXT DEFAULT ''
		);`
	}
	if _, err := DB.Exec(auditQuery); err != nil {
		return fmt.Errorf("failed to create audit_logs table: %w", err)
	}

	// Create security_scans table
	var securityQuery string
	switch currentDBType {
	case DBTypePostgres:
		securityQuery = `
		CREATE TABLE IF NOT EXISTS security_scans (
			id SERIAL PRIMARY KEY,
			scan_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			cluster_name VARCHAR(255),
			namespace VARCHAR(255) DEFAULT '',
			scan_type VARCHAR(50) DEFAULT 'full',
			duration_ms INTEGER,
			overall_score DECIMAL(5,2),
			risk_level VARCHAR(20),
			tools_used TEXT,
			critical_count INTEGER DEFAULT 0,
			high_count INTEGER DEFAULT 0,
			medium_count INTEGER DEFAULT 0,
			low_count INTEGER DEFAULT 0,
			pod_issues_count INTEGER DEFAULT 0,
			rbac_issues_count INTEGER DEFAULT 0,
			network_issues_count INTEGER DEFAULT 0,
			cis_pass_count INTEGER DEFAULT 0,
			cis_fail_count INTEGER DEFAULT 0,
			cis_score DECIMAL(5,2),
			scan_result JSONB,
			triggered_by VARCHAR(255) DEFAULT '',
			source VARCHAR(50) DEFAULT ''
		);`
	case DBTypeMariaDB, DBTypeMySQL:
		securityQuery = `
		CREATE TABLE IF NOT EXISTS security_scans (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			scan_time DATETIME DEFAULT CURRENT_TIMESTAMP,
			cluster_name VARCHAR(255),
			namespace VARCHAR(255) DEFAULT '',
			scan_type VARCHAR(50) DEFAULT 'full',
			duration_ms INTEGER,
			overall_score DECIMAL(5,2),
			risk_level VARCHAR(20),
			tools_used TEXT,
			critical_count INTEGER DEFAULT 0,
			high_count INTEGER DEFAULT 0,
			medium_count INTEGER DEFAULT 0,
			low_count INTEGER DEFAULT 0,
			pod_issues_count INTEGER DEFAULT 0,
			rbac_issues_count INTEGER DEFAULT 0,
			network_issues_count INTEGER DEFAULT 0,
			cis_pass_count INTEGER DEFAULT 0,
			cis_fail_count INTEGER DEFAULT 0,
			cis_score DECIMAL(5,2),
			scan_result JSON,
			triggered_by VARCHAR(255) DEFAULT '',
			source VARCHAR(50) DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
	default: // SQLite
		securityQuery = `
		CREATE TABLE IF NOT EXISTS security_scans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_time DATETIME DEFAULT CURRENT_TIMESTAMP,
			cluster_name TEXT,
			namespace TEXT DEFAULT '',
			scan_type TEXT DEFAULT 'full',
			duration_ms INTEGER,
			overall_score REAL,
			risk_level TEXT,
			tools_used TEXT,
			critical_count INTEGER DEFAULT 0,
			high_count INTEGER DEFAULT 0,
			medium_count INTEGER DEFAULT 0,
			low_count INTEGER DEFAULT 0,
			pod_issues_count INTEGER DEFAULT 0,
			rbac_issues_count INTEGER DEFAULT 0,
			network_issues_count INTEGER DEFAULT 0,
			cis_pass_count INTEGER DEFAULT 0,
			cis_fail_count INTEGER DEFAULT 0,
			cis_score REAL,
			scan_result TEXT,
			triggered_by TEXT DEFAULT '',
			source TEXT DEFAULT ''
		);`
	}
	if _, err := DB.Exec(securityQuery); err != nil {
		// Ignore if table exists
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create security_scans table: %w", err)
		}
	}

	// Create indexes
	indexQueries := getIndexQueries()
	for _, q := range indexQueries {
		DB.Exec(q) // Ignore index errors
	}

	// Create llm_usage table for token tracking
	if err := InitLLMUsageTable(); err != nil {
		// Log but don't fail - non-critical feature
		fmt.Printf("Warning: failed to create llm_usage table: %v\n", err)
	}

	return nil
}

func getIndexQueries() []string {
	switch currentDBType {
	case DBTypePostgres:
		return []string{
			"CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp DESC);",
			"CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(\"user\");",
			"CREATE INDEX IF NOT EXISTS idx_audit_action_type ON audit_logs(action_type);",
			"CREATE INDEX IF NOT EXISTS idx_audit_source ON audit_logs(source);",
			"CREATE INDEX IF NOT EXISTS idx_security_scan_time ON security_scans(scan_time DESC);",
			"CREATE INDEX IF NOT EXISTS idx_security_cluster ON security_scans(cluster_name);",
			"CREATE INDEX IF NOT EXISTS idx_security_risk ON security_scans(risk_level);",
		}
	case DBTypeMariaDB, DBTypeMySQL:
		return []string{
			"CREATE INDEX idx_audit_timestamp ON audit_logs(timestamp DESC);",
			"CREATE INDEX idx_audit_user ON audit_logs(user);",
			"CREATE INDEX idx_audit_action_type ON audit_logs(action_type);",
			"CREATE INDEX idx_audit_source ON audit_logs(source);",
			"CREATE INDEX idx_security_scan_time ON security_scans(scan_time DESC);",
			"CREATE INDEX idx_security_cluster ON security_scans(cluster_name);",
			"CREATE INDEX idx_security_risk ON security_scans(risk_level);",
		}
	default:
		return []string{
			"CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp DESC);",
			"CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user);",
			"CREATE INDEX IF NOT EXISTS idx_audit_action_type ON audit_logs(action_type);",
			"CREATE INDEX IF NOT EXISTS idx_audit_k8s_user ON audit_logs(k8s_user);",
			"CREATE INDEX IF NOT EXISTS idx_audit_source ON audit_logs(source);",
			"CREATE INDEX IF NOT EXISTS idx_security_scan_time ON security_scans(scan_time DESC);",
			"CREATE INDEX IF NOT EXISTS idx_security_cluster ON security_scans(cluster_name);",
			"CREATE INDEX IF NOT EXISTS idx_security_risk ON security_scans(risk_level);",
		}
	}
}

func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// migrateAuditLogsTable adds missing columns to existing audit_logs table
func migrateAuditLogsTable() error {
	if DB == nil {
		return nil
	}

	// Check if table exists by trying to query it
	rows, err := DB.Query("SELECT * FROM audit_logs LIMIT 0")
	if err != nil {
		// Table doesn't exist, will be created fresh
		return nil
	}
	defer rows.Close()

	// Get existing columns
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	existingCols := make(map[string]bool)
	for _, col := range columns {
		existingCols[col] = true
	}

	// Define new columns that may need to be added
	newColumns := []struct {
		name        string
		sqliteDef   string
		postgresDef string
		mysqlDef    string
	}{
		{"action_type", "TEXT DEFAULT 'mutation'", "VARCHAR(50) DEFAULT 'mutation'", "VARCHAR(50) DEFAULT 'mutation'"},
		{"k8s_user", "TEXT DEFAULT ''", "VARCHAR(255) DEFAULT ''", "VARCHAR(255) DEFAULT ''"},
		{"k8s_context", "TEXT DEFAULT ''", "VARCHAR(255) DEFAULT ''", "VARCHAR(255) DEFAULT ''"},
		{"k8s_cluster", "TEXT DEFAULT ''", "VARCHAR(255) DEFAULT ''", "VARCHAR(255) DEFAULT ''"},
		{"namespace", "TEXT DEFAULT ''", "VARCHAR(255) DEFAULT ''", "VARCHAR(255) DEFAULT ''"},
		{"llm_request", "TEXT DEFAULT ''", "TEXT DEFAULT ''", "TEXT"},
		{"llm_response", "TEXT DEFAULT ''", "TEXT DEFAULT ''", "TEXT"},
		{"llm_tool", "TEXT DEFAULT ''", "VARCHAR(100) DEFAULT ''", "VARCHAR(100) DEFAULT ''"},
		{"llm_command", "TEXT DEFAULT ''", "TEXT DEFAULT ''", "TEXT"},
		{"llm_approved", "INTEGER DEFAULT 0", "BOOLEAN DEFAULT FALSE", "TINYINT(1) DEFAULT 0"},
		{"source", "TEXT DEFAULT ''", "VARCHAR(50) DEFAULT ''", "VARCHAR(50) DEFAULT ''"},
		{"client_ip", "TEXT DEFAULT ''", "VARCHAR(50) DEFAULT ''", "VARCHAR(50) DEFAULT ''"},
		{"session_id", "TEXT DEFAULT ''", "VARCHAR(255) DEFAULT ''", "VARCHAR(255) DEFAULT ''"},
		{"success", "INTEGER DEFAULT 1", "BOOLEAN DEFAULT TRUE", "TINYINT(1) DEFAULT 1"},
		{"error_msg", "TEXT DEFAULT ''", "TEXT DEFAULT ''", "TEXT"},
	}

	// Add missing columns
	for _, col := range newColumns {
		if existingCols[col.name] {
			continue
		}

		var colDef string
		switch currentDBType {
		case DBTypePostgres:
			colDef = col.postgresDef
		case DBTypeMariaDB, DBTypeMySQL:
			colDef = col.mysqlDef
		default:
			colDef = col.sqliteDef
		}

		query := fmt.Sprintf("ALTER TABLE audit_logs ADD COLUMN %s %s", col.name, colDef)
		if _, err := DB.Exec(query); err != nil {
			// Column might already exist or other error - log but continue
			fmt.Printf("Warning: could not add column %s: %v\n", col.name, err)
		}
	}

	return nil
}
