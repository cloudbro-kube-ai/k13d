package cli

import (
	"testing"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

func TestInitDB_AuditDisabled(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.EnableAudit = false

	cleanup := InitDB(cfg)
	// Should return a noop cleanup
	cleanup() // should not panic
}

func TestInitDB_PersistenceDisabled(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.EnableAudit = true
	// Disable all persistence flags
	cfg.Storage.PersistAuditLogs = false
	cfg.Storage.PersistLLMUsage = false
	cfg.Storage.PersistSecurityScans = false
	cfg.Storage.PersistMetrics = false
	cfg.Storage.PersistSessions = false

	cleanup := InitDB(cfg)
	cleanup() // should not panic
}

func TestInitDB_WithTempDB(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.EnableAudit = true
	cfg.Storage.DBType = "sqlite"
	cfg.Storage.DBPath = t.TempDir() + "/test-audit.db"

	cleanup := InitDB(cfg)
	defer cleanup()

	// Cleanup should work without panic
}

func TestInitDB_InvalidPath(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.EnableAudit = true
	cfg.Storage.DBType = "sqlite"
	cfg.Storage.DBPath = "/nonexistent/path/that/does/not/exist/db.sqlite"

	// Should not panic — returns noop on error
	cleanup := InitDB(cfg)
	cleanup()
}
