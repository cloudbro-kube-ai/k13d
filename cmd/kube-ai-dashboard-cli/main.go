// TODO: This file duplicates significant logic from cmd/kubectl-k13d/main.go.
// Extract shared startup logic (flag parsing, DB init, embedded LLM, web/TUI runner)
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
	"syscall"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/db"
	"github.com/cloudbro-kube-ai/k13d/pkg/llm/embedded"
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

func main() {
	// Command line flags - using consistent --long-form style
	// Flags fall back to K13D_* environment variables for Docker/K8s compatibility
	// Mode flags
	webMode := flag.Bool("web", false, "Start web server mode")
	tuiMode := flag.Bool("tui", false, "Start TUI mode (default when no mode specified)")
	mcpMode := flag.Bool("mcp", false, "Start MCP server mode (stdio transport)")
	webPort := flag.Int("port", 8080, "Web server port (used with --web)")

	// Namespace flags (k9s compatible)
	namespace := flag.String("namespace", "", "Initial namespace (use 'all' for all namespaces)")
	flag.StringVar(namespace, "n", "", "Initial namespace (short for --namespace)")
	allNamespaces := flag.Bool("all-namespaces", false, "Start with all namespaces")
	flag.BoolVar(allNamespaces, "A", false, "Start with all namespaces (short for --all-namespaces)")

	// Info flags
	showVersion := flag.Bool("version", false, "Show version information")
	genCompletion := flag.String("completion", "", "Generate shell completion (bash, zsh, fish)")

	// Web server auth flags (env: K13D_AUTH_MODE, K13D_USERNAME, K13D_PASSWORD)
	authMode := flag.String("auth-mode", envDefault("K13D_AUTH_MODE", "token"), "Authentication mode: token (K8s RBAC), local (username/password), ldap")
	authDisabled := flag.Bool("no-auth", false, "Disable authentication (not recommended)")
	adminUser := flag.String("admin-user", envDefault("K13D_USERNAME", ""), "Default admin username for local auth mode")
	adminPass := flag.String("admin-password", envDefault("K13D_PASSWORD", ""), "Default admin password for local auth mode")

	// Storage flags
	dbPath := flag.String("db-path", "", "SQLite database path (default: ~/.config/k13d/audit.db)")
	disableDB := flag.Bool("no-db", false, "Disable database persistence entirely")
	showStorageInfo := flag.Bool("storage-info", false, "Show storage configuration and data locations")

	// Embedded LLM flags
	embeddedLLM := flag.Bool("embedded-llm", false, "Start embedded LLM server (llama.cpp)")
	embeddedLLMPort := flag.Int("embedded-llm-port", 8081, "Embedded LLM server port")
	embeddedLLMModel := flag.String("embedded-llm-model", "", "Path to custom GGUF model file")
	embeddedLLMContext := flag.Int("embedded-llm-context", 0, "Context size (0 = auto-detect based on model)")
	downloadModel := flag.Bool("download-model", false, "Download the default model (Qwen2.5-0.5B-Instruct)")
	embeddedLLMStatus := flag.Bool("embedded-llm-status", false, "Show embedded LLM status")

	flag.Parse()

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

	// Embedded LLM operations
	if *downloadModel {
		downloadEmbeddedModel()
		return
	}

	if *embeddedLLMStatus {
		showEmbeddedLLMStatus()
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

	// Override storage settings from CLI flags
	if *dbPath != "" {
		cfg.Storage.DBPath = *dbPath
	}
	if *disableDB {
		cfg.EnableAudit = false
	}

	// Start embedded LLM server if requested
	var embeddedServer *embedded.Server
	if *embeddedLLM {
		var err error
		embeddedServer, err = startEmbeddedLLM(*embeddedLLMPort, *embeddedLLMModel, *embeddedLLMContext)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to start embedded LLM: %v\n", err)
			os.Exit(1)
		}
		defer embeddedServer.Stop()

		// Update config to use embedded LLM
		cfg.LLM.Provider = "embedded"
		cfg.LLM.Endpoint = embeddedServer.Endpoint()
		cfg.LLM.Model = "qwen2.5-0.5b-instruct"
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
			EmbeddedLLM:     *embeddedLLM,
		}
		runWebServer(cfg, *webPort, authOpts, embeddedServer)
		return
	}

	// TUI mode with optional namespace
	initialNS := *namespace
	if *allNamespaces {
		initialNS = "" // empty means all namespaces
	}
	runTUI(cfg, initialNS, embeddedServer)
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

func runWebServer(cfg *config.Config, port int, authOpts *web.AuthOptions, embeddedServer *embedded.Server) {
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

		// Stop embedded LLM server if running
		if embeddedServer != nil {
			fmt.Println("Stopping embedded LLM server...")
			if err := embeddedServer.Stop(); err != nil {
				log.Errorf("Error stopping embedded LLM: %v", err)
			}
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

func runTUI(cfg *config.Config, initialNamespace string, embeddedServer *embedded.Server) {
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

	// Stop embedded LLM server on exit
	if embeddedServer != nil {
		defer func() {
			fmt.Println("Stopping embedded LLM server...")
			if err := embeddedServer.Stop(); err != nil {
				log.Errorf("Error stopping embedded LLM: %v", err)
			}
		}()
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
    opts="-n --namespace -A -web -port --version --completion"

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
        '-web[Start web server mode]'
        '-port[Web server port]:port:'
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

// Embedded LLM functions

func startEmbeddedLLM(port int, modelPath string, contextSize int) (*embedded.Server, error) {
	cfg := embedded.DefaultConfig()
	cfg.Port = port
	if modelPath != "" {
		cfg.ModelPath = modelPath
	}

	// Auto-detect context size based on model if not specified
	if contextSize > 0 {
		cfg.ContextSize = contextSize
	} else if modelPath != "" {
		// Get recommended context size for the model
		modelInfo := embedded.GetModelContextInfo(modelPath)
		cfg.ContextSize = modelInfo.RecommendedCtx
		fmt.Printf("  Auto-detected context size: %d (max: %d)\n", modelInfo.RecommendedCtx, modelInfo.MaxContext)
	}

	server, err := embedded.NewServer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedded LLM server: %w", err)
	}

	// Check if binary exists
	if err := server.EnsureBinary(); err != nil {
		return nil, err
	}

	// Check if model exists
	if _, err := os.Stat(server.ModelPath()); err != nil {
		return nil, fmt.Errorf("model not found at %s. Run 'k13d --download-model' to download", server.ModelPath())
	}

	// Use background context - signal handling is done in runWebServer or runTUI
	ctx := context.Background()

	fmt.Printf("Starting embedded LLM server...\n")
	fmt.Printf("  Model: %s\n", server.ModelPath())
	fmt.Printf("  Port: %d\n", port)

	if err := server.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	fmt.Printf("Embedded LLM server running at %s\n", server.Endpoint())
	return server, nil
}

func downloadEmbeddedModel() {
	cfg := embedded.DefaultConfig()
	server, err := embedded.NewServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Downloading model to: %s\n", server.ModelPath())
	fmt.Printf("Model URL: %s\n", embedded.ModelURL)
	fmt.Println("This may take a few minutes...")

	ctx := context.Background()
	err = server.EnsureModel(ctx, func(downloaded, total int64) {
		if total > 0 {
			pct := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rDownloading: %.1f%% (%d / %d MB)", pct, downloaded/1024/1024, total/1024/1024)
		}
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError downloading model: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nModel downloaded successfully!")
	fmt.Printf("Path: %s\n", server.ModelPath())
}

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

func showEmbeddedLLMStatus() {
	cfg := embedded.DefaultConfig()
	server, err := embedded.NewServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	status := server.Status()

	fmt.Println("Embedded LLM Status")
	fmt.Println("===================")
	fmt.Printf("Data Directory: %s\n", server.DataDir())
	fmt.Printf("Model: %s\n", status.Model)
	fmt.Printf("Model Exists: %v\n", status.ModelExists)

	// Show model context info if model exists
	if status.ModelExists {
		modelInfo := embedded.GetModelContextInfo(server.ModelPath())
		fmt.Printf("Max Context: %d tokens\n", modelInfo.MaxContext)
		fmt.Printf("Recommended Context: %d tokens\n", modelInfo.RecommendedCtx)
		fmt.Printf("Min RAM: %dGB\n", modelInfo.MinRAM)
	}

	fmt.Printf("Server Binary: %s\n", server.ServerBinaryPath())

	if _, err := os.Stat(server.ServerBinaryPath()); err == nil {
		fmt.Printf("Binary Exists: true\n")
	} else {
		fmt.Printf("Binary Exists: false\n")
	}

	fmt.Printf("Default Port: %d\n", status.Port)

	if !status.ModelExists {
		fmt.Println("\nTo download the model, run:")
		fmt.Println("  k13d --download-model")
	}

	if _, err := os.Stat(server.ServerBinaryPath()); err != nil {
		fmt.Println("\nNote: The llama-server binary is bundled with k13d-llm releases.")
		fmt.Println("Download from: https://github.com/cloudbro-kube-ai/k13d/releases")
	}
}
