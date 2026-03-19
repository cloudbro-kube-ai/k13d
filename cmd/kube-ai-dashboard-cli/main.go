// TODO: This file duplicates significant logic from cmd/kubectl-k13d/main.go.
// Extract shared startup logic (flag parsing, DB init, web/TUI runner)
// into an internal/cli package and keep both main.go files as thin entry points.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/db"
	"github.com/cloudbro-kube-ai/k13d/pkg/log"
	mcpserver "github.com/cloudbro-kube-ai/k13d/pkg/mcp/server"
	"github.com/cloudbro-kube-ai/k13d/pkg/ui"
	"github.com/cloudbro-kube-ai/k13d/pkg/web"
)

// Version info (set by ldflags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// envDefault returns the environment variable value if set, otherwise the default.
func envDefault(envKey, defaultVal string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return defaultVal
}

func envBoolDefault(envKey string, defaultVal bool) bool {
	v := os.Getenv(envKey)
	if v == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return parsed
}

func envIntDefault(envKey string, defaultVal int) int {
	v := os.Getenv(envKey)
	if v == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return parsed
}

func main() {
	// Command line flags - Go's flag parser accepts both -flag and --flag forms.
	// Selected flags also fall back to K13D_* environment variables for Docker/K8s compatibility.
	// Mode flags
	webMode := flag.Bool("web", envBoolDefault("K13D_WEB", false), "Start web server mode")
	tuiMode := flag.Bool("tui", false, "Start TUI mode (default when no mode specified)")
	mcpMode := flag.Bool("mcp", false, "Start MCP server mode (stdio transport)")
	webPort := flag.Int("port", envIntDefault("K13D_PORT", 8080), "Web server port (used with --web)")
	configPath := flag.String("config", envDefault("K13D_CONFIG", ""), "Config file path (default: platform XDG config dir + /k13d/config.yaml)")

	// Namespace flags (k9s compatible)
	namespace := flag.String("namespace", envDefault("K13D_NAMESPACE", ""), "Initial namespace (use 'all' for all namespaces)")
	flag.StringVar(namespace, "n", "", "Initial namespace (short for --namespace)")
	allNamespaces := flag.Bool("all-namespaces", envBoolDefault("K13D_ALL_NAMESPACES", false), "Start with all namespaces")
	flag.BoolVar(allNamespaces, "A", false, "Start with all namespaces (short for --all-namespaces)")

	// Info flags
	showVersion := flag.Bool("version", false, "Show version information")
	genCompletion := flag.String("completion", "", "Generate shell completion (bash, zsh, fish)")

	// Web server auth flags (env: K13D_AUTH_MODE, K13D_USERNAME, K13D_PASSWORD)
	authMode := flag.String("auth-mode", envDefault("K13D_AUTH_MODE", "token"), "Authentication mode: token (K8s RBAC), local (username/password), ldap, oidc")
	authDisabled := flag.Bool("no-auth", envBoolDefault("K13D_NO_AUTH", false), "Disable authentication (not recommended)")
	adminUser := flag.String("admin-user", envDefault("K13D_USERNAME", ""), "Default admin username for local auth mode")
	adminPass := flag.String("admin-password", envDefault("K13D_PASSWORD", ""), "Default admin password for local auth mode")

	// Storage flags
	dbPath := flag.String("db-path", envDefault("K13D_DB_PATH", ""), "SQLite database path (default: platform XDG config dir + /k13d/audit.db)")
	disableDB := flag.Bool("no-db", envBoolDefault("K13D_NO_DB", false), "Disable database persistence entirely")
	showStorageInfo := flag.Bool("storage-info", false, "Show storage configuration and data locations")

	flag.Parse()

	if *configPath != "" {
		_ = os.Setenv("K13D_CONFIG", *configPath)
	}

	// -tui flag is explicit TUI mode (useful for Docker)
	_ = tuiMode // TUI is default when -web is not specified

	// Show version
	if *showVersion {
		fmt.Printf("k13d version %s\n", Version)
		fmt.Printf("  Build time: %s\n", BuildTime)
		fmt.Printf("  Git commit: %s\n", GitCommit)
		return
	}

	// Generate shell completion
	if *genCompletion != "" {
		generateCompletion(*genCompletion)
		return
	}

	// Show storage info
	if *showStorageInfo {
		showStorageConfiguration()
		return
	}

	// Initialize enterprise logger
	if err := log.Init("k13d"); err != nil {
		fmt.Printf("Warning: could not initialize logger: %v\n", err)
	}

	log.Infof("Starting k13d application...")

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Errorf("Failed to load config: %v", err)
		cfg = config.NewDefaultConfig()
	}

	// Apply log level from config
	log.SetLevel(cfg.LogLevel)

	// Apply timezone from config
	if cfg.Timezone != "" && cfg.Timezone != "auto" {
		loc, err := time.LoadLocation(cfg.Timezone)
		if err != nil {
			log.Errorf("Failed to load timezone %s: %v", cfg.Timezone, err)
		} else {
			time.Local = loc
			log.Infof("Timezone set to %s", cfg.Timezone)
		}
	}

	// Override storage settings from CLI flags
	if *dbPath != "" {
		cfg.Storage.DBPath = *dbPath
	}
	if *disableDB {
		cfg.EnableAudit = false
	}

	// MCP server mode
	if *mcpMode {
		runMCPServer()
		return
	}

	// Web mode
	if *webMode {
		authOpts := &web.AuthOptions{
			Mode:            *authMode,
			Disabled:        *authDisabled,
			DefaultAdmin:    *adminUser,
			DefaultPassword: *adminPass,
		}
		runWebServer(cfg, *webPort, authOpts)
		return
	}

	// TUI mode with optional namespace
	initialNS := *namespace
	if *allNamespaces {
		initialNS = "" // empty means all namespaces
	}
	runTUI(cfg, initialNS)
}

func runMCPServer() {
	// Create MCP server
	server := mcpserver.New("k13d", Version)

	// Register default tools
	for _, tool := range mcpserver.DefaultTools() {
		server.RegisterTool(tool)
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	// Run server (blocks until context is cancelled or EOF)
	if err := server.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}

func runWebServer(cfg *config.Config, port int, authOpts *web.AuthOptions) {
	// Initialize audit database and file for web mode
	if cfg.EnableAudit && cfg.IsPersistenceEnabled() {
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
		}
		defer db.Close()

		// Initialize audit file if enabled
		if cfg.Storage.EnableAuditFile {
			if err := db.InitAuditFile(cfg.GetEffectiveAuditFilePath()); err != nil {
				log.Errorf("Failed to initialize audit file: %v", err)
			}
			defer db.CloseAuditFile()
		}
	}

	// Pass version info to web server
	versionInfo := &web.VersionInfo{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
	}

	server, err := web.NewServerWithAuth(cfg, port, authOpts, versionInfo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create web server: %v\n", err)
		log.Errorf("Failed to create web server: %v", err)
		os.Exit(1)
	}

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start web server in goroutine
	serverErrCh := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			serverErrCh <- err
		}
		close(serverErrCh)
	}()

	// Wait for signal or server error
	select {
	case sig := <-sigCh:
		fmt.Printf("\nReceived signal %v, shutting down...\n", sig)

		// Stop web server
		if err := server.Stop(); err != nil {
			log.Errorf("Error stopping web server: %v", err)
		}

		fmt.Println("Shutdown complete.")
	case err := <-serverErrCh:
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Web server stopped: %v\n", err)
			log.Errorf("Web server error: %v", err)
			os.Exit(1)
		}
	}
}

func runTUI(cfg *config.Config, initialNamespace string) {
	// Initialize audit database if enabled in config
	if cfg.EnableAudit && cfg.IsPersistenceEnabled() {
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
		}
		defer db.Close()

		// Initialize audit file logging if enabled
		if cfg.Storage.EnableAuditFile {
			if err := db.InitAuditFile(cfg.GetEffectiveAuditFilePath()); err != nil {
				log.Errorf("Failed to initialize audit file: %v", err)
			}
			defer db.CloseAuditFile()
		}
	}

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("PANIC RECOVERED: %v\n%s", r, debug.Stack())
			fmt.Fprintf(os.Stderr, "k13d crashed due to a panic. Details have been logged.\n")
			os.Exit(1)
		}
	}()

	ui.Version = Version
	app := ui.NewAppWithNamespace(initialNamespace)
	if err := app.Run(); err != nil {
		log.Errorf("Application exited with error: %v", err)
		os.Exit(1)
	}
	log.Infof("k13d application exited cleanly.")
}

// generateCompletion outputs shell completion script
func generateCompletion(shell string) {
	switch shell {
	case "bash":
		fmt.Print(bashCompletion)
	case "zsh":
		fmt.Print(zshCompletion)
	case "fish":
		fmt.Print(fishCompletion)
	default:
		fmt.Fprintf(os.Stderr, "Unknown shell: %s. Supported: bash, zsh, fish\n", shell)
		os.Exit(1)
	}
}

const bashCompletion = `# k13d bash completion
_k13d_completions() {
    local cur prev opts namespaces
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Main options
    opts="-n --namespace -A --web --port --version --completion"

    # Complete namespace after -n or --namespace
    if [[ "${prev}" == "-n" ]] || [[ "${prev}" == "--namespace" ]]; then
        namespaces=$(kubectl get namespaces -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
        COMPREPLY=( $(compgen -W "${namespaces} all" -- ${cur}) )
        return 0
    fi

    # Complete shell after --completion
    if [[ "${prev}" == "--completion" ]]; then
        COMPREPLY=( $(compgen -W "bash zsh fish" -- ${cur}) )
        return 0
    fi

    # Default to options
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}
complete -F _k13d_completions k13d

# To enable: source <(k13d --completion bash)
# Or add to ~/.bashrc: eval "$(k13d --completion bash)"
`

const zshCompletion = `#compdef k13d

_k13d() {
    local -a opts namespaces

    opts=(
        '-n[Initial namespace]:namespace:->namespaces'
        '--namespace[Initial namespace]:namespace:->namespaces'
        '-A[Start with all namespaces]'
        '--web[Start web server mode]'
        '--port[Web server port]:port:'
        '--version[Show version information]'
        '--completion[Generate shell completion]:shell:(bash zsh fish)'
    )

    _arguments -s $opts

    case "$state" in
        namespaces)
            namespaces=(${(f)"$(kubectl get namespaces -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null)"})
            namespaces+=("all")
            _describe 'namespace' namespaces
            ;;
    esac
}

_k13d "$@"

# To enable: source <(k13d --completion zsh)
# Or add to ~/.zshrc: eval "$(k13d --completion zsh)"
`

const fishCompletion = `# k13d fish completion
function __k13d_get_namespaces
    kubectl get namespaces -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null
    echo "all"
end

complete -c k13d -f
complete -c k13d -s n -l namespace -d 'Initial namespace' -xa '(__k13d_get_namespaces)'
complete -c k13d -s A -d 'Start with all namespaces'
complete -c k13d -l web -d 'Start web server mode'
complete -c k13d -l port -d 'Web server port'
complete -c k13d -l version -d 'Show version information'
complete -c k13d -l completion -d 'Generate shell completion' -xa 'bash zsh fish'

# To enable: k13d --completion fish | source
# Or add to ~/.config/fish/config.fish: k13d --completion fish | source
`

func showStorageConfiguration() {
	cfg, _ := config.LoadConfig()
	if cfg == nil {
		cfg = config.NewDefaultConfig()
	}

	fmt.Println("k13d Storage Configuration")
	fmt.Println("===========================")
	fmt.Println()

	// Database configuration
	fmt.Println("[Database]")
	fmt.Printf("  Type:     %s\n", cfg.Storage.DBType)
	if cfg.Storage.DBType == "sqlite" || cfg.Storage.DBType == "" {
		dbPath := cfg.GetEffectiveDBPath()
		fmt.Printf("  Path:     %s\n", dbPath)
		if _, err := os.Stat(dbPath); err == nil {
			info, _ := os.Stat(dbPath)
			fmt.Printf("  Size:     %.2f MB\n", float64(info.Size())/1024/1024)
			fmt.Printf("  Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("  Status:   Not created yet\n")
		}
	} else {
		fmt.Printf("  Host:     %s:%d\n", cfg.Storage.DBHost, cfg.Storage.DBPort)
		fmt.Printf("  Database: %s\n", cfg.Storage.DBName)
		fmt.Printf("  User:     %s\n", cfg.Storage.DBUser)
	}
	fmt.Println()

	// Data persistence settings
	fmt.Println("[Data Persistence]")
	fmt.Printf("  Audit Logs:     %v\n", cfg.Storage.PersistAuditLogs)
	fmt.Printf("  LLM Usage:      %v\n", cfg.Storage.PersistLLMUsage)
	fmt.Printf("  Security Scans: %v\n", cfg.Storage.PersistSecurityScans)
	fmt.Printf("  Metrics:        %v\n", cfg.Storage.PersistMetrics)
	fmt.Printf("  AI Sessions:    %v\n", cfg.Storage.PersistSessions)
	fmt.Println()

	// File-based audit log
	fmt.Println("[Audit File]")
	fmt.Printf("  Enabled:  %v\n", cfg.Storage.EnableAuditFile)
	if cfg.Storage.EnableAuditFile {
		auditPath := cfg.GetEffectiveAuditFilePath()
		fmt.Printf("  Path:     %s\n", auditPath)
		if info, err := os.Stat(auditPath); err == nil {
			fmt.Printf("  Size:     %.2f MB\n", float64(info.Size())/1024/1024)
		}
	}
	fmt.Println()

	// Data retention
	fmt.Println("[Data Retention]")
	if cfg.Storage.AuditRetentionDays == 0 {
		fmt.Printf("  Audit Logs:     Forever\n")
	} else {
		fmt.Printf("  Audit Logs:     %d days\n", cfg.Storage.AuditRetentionDays)
	}
	fmt.Printf("  Metrics:        %d days\n", cfg.Storage.MetricsRetentionDays)
	fmt.Printf("  LLM Usage:      %d days\n", cfg.Storage.LLMUsageRetentionDays)
	fmt.Println()

	// Sessions directory
	sessionsDir := config.DefaultSessionsPath()
	fmt.Println("[AI Sessions]")
	fmt.Printf("  Path:     %s\n", sessionsDir)
	if entries, err := os.ReadDir(sessionsDir); err == nil {
		fmt.Printf("  Sessions: %d\n", len(entries))
	} else {
		fmt.Printf("  Status:   Not created yet\n")
	}
	fmt.Println()

	// Configuration file
	fmt.Println("[Configuration File]")
	fmt.Printf("  Path:     %s\n", config.GetConfigPath())
	if _, err := os.Stat(config.GetConfigPath()); err == nil {
		fmt.Printf("  Status:   Exists\n")
	} else {
		fmt.Printf("  Status:   Using defaults\n")
	}
	fmt.Println()

	// Data stored summary
	fmt.Println("[Data Stored in SQLite]")
	fmt.Println("  - audit_logs:      User actions, K8s operations, LLM interactions")
	fmt.Println("  - security_scans:  Security assessment results")
	fmt.Println("  - llm_usage:       LLM token usage and API call tracking")
	fmt.Println("  - cluster_metrics: Time-series cluster resource metrics")
	fmt.Println("  - node_metrics:    Per-node CPU/memory metrics")
	fmt.Println("  - pod_metrics:     Per-pod resource usage")
	fmt.Println()

	fmt.Println("[Data Stored in Files]")
	fmt.Printf("  - Sessions:        %s/*.json\n", sessionsDir)
	fmt.Printf("  - Audit Log:       %s (if enabled)\n", cfg.GetEffectiveAuditFilePath())
	fmt.Printf("  - Config:          %s\n", config.GetConfigPath())
}
