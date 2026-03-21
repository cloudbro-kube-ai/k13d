package cli

import (
	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/db"
	"github.com/cloudbro-kube-ai/k13d/pkg/log"
)

// InitDB initializes the audit database and optional file-based audit log
// based on the current config. Returns a cleanup function that should be
// deferred by the caller.
func InitDB(cfg *config.Config) func() {
	noop := func() {}

	if !cfg.EnableAudit || !cfg.IsPersistenceEnabled() {
		return noop
	}

	dbCfg := db.DBConfig{
		Type:     db.DBType(cfg.Storage.DBType),
		Path:     cfg.GetEffectiveDBPath(),
		Host:     cfg.Storage.DBHost,
		Port:     cfg.Storage.DBPort,
		Database: cfg.Storage.DBName,
		Username: cfg.Storage.DBUser,
		Password: cfg.Storage.DBPassword,
		SSLMode:  cfg.Storage.DBSSLMode,
	}
	if err := db.InitWithConfig(dbCfg); err != nil {
		log.Errorf("Failed to initialize audit database: %v", err)
		return noop
	}

	cleanup := func() { db.Close() }

	if cfg.Storage.EnableAuditFile {
		if err := db.InitAuditFile(cfg.GetEffectiveAuditFilePath()); err != nil {
			log.Errorf("Failed to initialize audit file: %v", err)
		} else {
			cleanup = func() {
				db.CloseAuditFile()
				db.Close()
			}
		}
	}

	return cleanup
}
