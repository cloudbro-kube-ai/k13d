// kubectl-k13d is a kubectl plugin wrapper for k13d.
// When installed as kubectl-k13d in PATH, it becomes available as "kubectl k13d".
//
// TODO: This file duplicates significant logic from cmd/kube-ai-dashboard-cli/main.go.
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
	// Command line flags - same as main k13d binary.
	// Go's flag parser accepts both -flag and --flag forms.
	webMode := flag.Bool("web", envBoolDefault("K13D_WEB", false), "Start web server mode")
	tuiMode := flag.Bool("tui", false, "Start TUI mode (default when no mode specified)")
	mcpMode := flag.Bool("mcp", false, "Start MCP server mode (stdio transport)")
	webPort := flag.Int("port", envIntDefault("K13D_PORT", 8080), "Web server port (used with --web)")
	configPath := flag.String("config", envDefault("K13D_CONFIG", ""), "Config file path (default: platform XDG config dir + /k13d/config.yaml)")

	namespace := flag.String("namespace", envDefault("K13D_NAMESPACE", ""), "Initial namespace (use 'all' for all namespaces)")
	flag.StringVar(namespace, "n", "", "Initial namespace (short for --namespace)")
	allNamespaces := flag.Bool("all-namespaces", envBoolDefault("K13D_ALL_NAMESPACES", false), "Start with all namespaces")
	flag.BoolVar(allNamespaces, "A", false, "Start with all namespaces (short for --all-namespaces)")

	showVersion := flag.Bool("version", false, "Show version information")
	genCompletion := flag.String("completion", "", "Generate shell completion (bash, zsh, fish)")

	authMode := flag.String("auth-mode", envDefault("K13D_AUTH_MODE", "token"), "Authentication mode: token (K8s RBAC), local (username/password), ldap, oidc")
	authDisabled := flag.Bool("no-auth", envBoolDefault("K13D_NO_AUTH", false), "Disable authentication (not recommended)")
	adminUser := flag.String("admin-user", envDefault("K13D_USERNAME", ""), "Default admin username for local auth mode")
	adminPass := flag.String("admin-password", envDefault("K13D_PASSWORD", ""), "Default admin password for local auth mode")

	dbPath := flag.String("db-path", envDefault("K13D_DB_PATH", ""), "SQLite database path (default: platform XDG config dir + /k13d/audit.db)")
	disableDB := flag.Bool("no-db", envBoolDefault("K13D_NO_DB", false), "Disable database persistence entirely")
	showStorageInfo := flag.Bool("storage-info", false, "Show storage configuration and data locations")

	flag.Parse()

	if *configPath != "" {
		_ = os.Setenv("K13D_CONFIG", *configPath)
	}

	_ = tuiMode

	if *showVersion {
		fmt.Printf("kubectl-k13d version %s\n", Version)
		fmt.Printf("  Build time: %s\n", BuildTime)
		fmt.Printf("  Git commit: %s\n", GitCommit)
		return
	}

	if *genCompletion != "" {
		generateCompletion(*genCompletion)
		return
	}

	if *showStorageInfo {
		showStorageConfiguration()
		return
	}

	if err := log.Init("k13d"); err != nil {
		fmt.Printf("Warning: could not initialize logger: %v\n", err)
	}

	log.Infof("Starting kubectl-k13d plugin...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Errorf("Failed to load config: %v", err)
		cfg = config.NewDefaultConfig()
	}

	if *dbPath != "" {
		cfg.Storage.DBPath = *dbPath
	}
	if *disableDB {
		cfg.EnableAudit = false
	}

	if *mcpMode {
		runMCPServer()
		return
	}

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

	initialNS := *namespace
	if *allNamespaces {
		initialNS = ""
	}
	runTUI(cfg, initialNS)
}

func runMCPServer() {
	server := mcpserver.New("k13d", Version)
	for _, tool := range mcpserver.DefaultTools() {
		server.RegisterTool(tool)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}

func runWebServer(cfg *config.Config, port int, authOpts *web.AuthOptions) {
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

		if cfg.Storage.EnableAuditFile {
			if err := db.InitAuditFile(cfg.GetEffectiveAuditFilePath()); err != nil {
				log.Errorf("Failed to initialize audit file: %v", err)
			}
			defer db.CloseAuditFile()
		}
	}

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

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	serverErrCh := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			serverErrCh <- err
		}
		close(serverErrCh)
	}()

	select {
	case sig := <-sigCh:
		fmt.Printf("\nReceived signal %v, shutting down...\n", sig)
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

	app := ui.NewAppWithNamespace(initialNamespace)
	if err := app.Run(); err != nil {
		log.Errorf("Application exited with error: %v", err)
		os.Exit(1)
	}
	log.Infof("kubectl-k13d plugin exited cleanly.")
}

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

const bashCompletion = `# kubectl-k13d bash completion
_kubectl_k13d_completions() {
    local cur prev opts namespaces
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    opts="-n --namespace -A -web -port --version --completion"

    if [[ "${prev}" == "-n" ]] || [[ "${prev}" == "--namespace" ]]; then
        namespaces=$(kubectl get namespaces -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
        COMPREPLY=( $(compgen -W "${namespaces} all" -- ${cur}) )
        return 0
    fi

    if [[ "${prev}" == "--completion" ]]; then
        COMPREPLY=( $(compgen -W "bash zsh fish" -- ${cur}) )
        return 0
    fi

    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}
complete -F _kubectl_k13d_completions kubectl-k13d
`

const zshCompletion = `#compdef kubectl-k13d

_kubectl_k13d() {
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

_kubectl_k13d "$@"
`

const fishCompletion = `# kubectl-k13d fish completion
function __kubectl_k13d_get_namespaces
    kubectl get namespaces -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null
    echo "all"
end

complete -c kubectl-k13d -f
complete -c kubectl-k13d -s n -l namespace -d 'Initial namespace' -xa '(__kubectl_k13d_get_namespaces)'
complete -c kubectl-k13d -s A -d 'Start with all namespaces'
complete -c kubectl-k13d -l web -d 'Start web server mode'
complete -c kubectl-k13d -l port -d 'Web server port'
complete -c kubectl-k13d -l version -d 'Show version information'
complete -c kubectl-k13d -l completion -d 'Generate shell completion' -xa 'bash zsh fish'
`

func showStorageConfiguration() {
	cfg, _ := config.LoadConfig()
	if cfg == nil {
		cfg = config.NewDefaultConfig()
	}

	fmt.Println("k13d Storage Configuration")
	fmt.Println("===========================")
	fmt.Println()

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
}
