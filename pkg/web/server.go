package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai/session"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/helm"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/mcp"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/metrics"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/security"
)

//go:embed static/*
var staticFiles embed.FS

// VersionInfo holds build version information
type VersionInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GitCommit string `json:"git_commit"`
}

type Server struct {
	cfg              *config.Config
	aiClient         *ai.Client
	k8sClient        *k8s.Client
	helmClient       *helm.Client
	mcpClient        *mcp.Client
	authManager      *AuthManager
	reportGenerator  *ReportGenerator
	metricsCollector *metrics.Collector
	securityScanner  *security.Scanner
	sessionStore     *session.Store // AI conversation session storage
	port             int
	server           *http.Server
	embeddedLLM      bool // Using embedded LLM server
	versionInfo      *VersionInfo

	// Tool approval management
	pendingApprovals     map[string]*PendingToolApproval
	pendingApprovalMutex sync.RWMutex
}

// PendingToolApproval represents a tool call waiting for user approval
type PendingToolApproval struct {
	ID        string    `json:"id"`
	ToolName  string    `json:"tool_name"`
	Command   string    `json:"command"`
	Category  string    `json:"category"` // read-only, write, dangerous
	Timestamp time.Time `json:"timestamp"`
	Response  chan bool `json:"-"`
}

type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"` // Session ID for conversation history
	Language  string `json:"language,omitempty"`   // Display language preference (e.g., "ko", "en")
}

type ChatResponse struct {
	Response string `json:"response"`
	Command  string `json:"command,omitempty"`
	Error    string `json:"error,omitempty"`
}

type K8sResourceResponse struct {
	Kind      string                   `json:"kind"`
	Items     []map[string]interface{} `json:"items"`
	Error     string                   `json:"error,omitempty"`
	Timestamp time.Time                `json:"timestamp"`
}

type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	mu      sync.Mutex
}

func (s *SSEWriter) Write(data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := fmt.Fprintf(s.w, "data: %s\n\n", data)
	if err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}

func NewServer(cfg *config.Config, port int, versionInfo *VersionInfo) (*Server, error) {
	var aiClient *ai.Client
	var err error

	fmt.Printf("Starting k13d web server...\n")
	fmt.Printf("  LLM Provider: %s, Model: %s\n", cfg.LLM.Provider, cfg.LLM.Model)

	if cfg.LLM.Endpoint != "" {
		aiClient, err = ai.NewClient(&cfg.LLM)
		if err != nil {
			fmt.Printf("  AI client creation failed: %v\n", err)
			aiClient = nil
		} else {
			fmt.Printf("  AI client: Ready\n")
		}
	} else {
		fmt.Printf("  AI client: Not configured\n")
	}

	k8sClient, err := k8s.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}
	fmt.Printf("  K8s client: Ready\n")

	// Initialize Helm client (uses default kubeconfig)
	helmClient := helm.NewClient("", "")
	fmt.Printf("  Helm client: Ready\n")

	// Initialize database
	if err := db.Init(""); err != nil {
		fmt.Printf("  Database: Failed to initialize (%v)\n", err)
	} else {
		fmt.Printf("  Database: Ready\n")
	}

	// Initialize auth manager
	// Note: DefaultPassword left empty so NewAuthManager generates a secure random password
	authConfig := &AuthConfig{
		Enabled:         cfg.EnableAudit, // Use audit flag to control auth for now
		AuthMode:        "local",         // Use local authentication
		SessionDuration: 24 * time.Hour,
		DefaultAdmin:    "admin",
		// DefaultPassword: intentionally empty for secure random generation
	}
	authManager := NewAuthManager(authConfig)
	fmt.Printf("  Authentication: %s\n", map[bool]string{true: "Enabled", false: "Disabled"}[authConfig.Enabled])

	// Initialize session store for AI conversation history
	sessionStore, err := session.NewStore()
	if err != nil {
		fmt.Printf("  Session Store: Failed to initialize (%v)\n", err)
	} else {
		fmt.Printf("  Session Store: Ready\n")
	}

	server := &Server{
		cfg:              cfg,
		aiClient:         aiClient,
		k8sClient:        k8sClient,
		helmClient:       helmClient,
		mcpClient:        mcp.NewClient(),
		authManager:      authManager,
		sessionStore:     sessionStore,
		port:             port,
		versionInfo:      versionInfo,
		pendingApprovals: make(map[string]*PendingToolApproval),
	}

	server.reportGenerator = NewReportGenerator(server)
	fmt.Printf("  Reports: Ready\n")

	// Metrics collector is disabled by default to avoid performance issues
	// It can still collect on-demand via /api/metrics/collect
	fmt.Printf("  Metrics Collector: Disabled (on-demand collection available)\n")

	// Initialize security scanner
	server.securityScanner = security.NewScanner(k8sClient)
	scannerInfo := "Basic checks"
	if server.securityScanner.TrivyAvailable() {
		scannerInfo += ", Trivy"
	}
	if server.securityScanner.KubeBenchAvailable() {
		scannerInfo += ", kube-bench"
	}
	fmt.Printf("  Security Scanner: Ready (%s)\n", scannerInfo)

	// Initialize MCP servers
	server.initMCPServers()

	return server, nil
}

// NewServerWithAuth creates a new server with custom authentication options
func NewServerWithAuth(cfg *config.Config, port int, authOpts *AuthOptions, versionInfo *VersionInfo) (*Server, error) {
	var aiClient *ai.Client
	var err error

	fmt.Printf("Starting k13d web server...\n")
	fmt.Printf("  LLM Provider: %s, Model: %s\n", cfg.LLM.Provider, cfg.LLM.Model)

	if cfg.LLM.Endpoint != "" {
		aiClient, err = ai.NewClient(&cfg.LLM)
		if err != nil {
			fmt.Printf("  AI client creation failed: %v\n", err)
			aiClient = nil
		} else {
			fmt.Printf("  AI client: Ready\n")
		}
	} else {
		fmt.Printf("  AI client: Not configured\n")
	}

	k8sClient, err := k8s.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}
	fmt.Printf("  K8s client: Ready\n")

	// Initialize Helm client (uses default kubeconfig)
	helmClient := helm.NewClient("", "")
	fmt.Printf("  Helm client: Ready\n")

	// Initialize database
	if err := db.Init(""); err != nil {
		fmt.Printf("  Database: Failed to initialize (%v)\n", err)
	} else {
		fmt.Printf("  Database: Ready\n")
	}

	// Initialize auth manager based on CLI options
	authConfig := &AuthConfig{
		Enabled:         !authOpts.Disabled,
		AuthMode:        authOpts.Mode,
		SessionDuration: 24 * time.Hour,
	}

	// Set default admin credentials
	if authOpts.DefaultAdmin != "" {
		authConfig.DefaultAdmin = authOpts.DefaultAdmin
	} else {
		authConfig.DefaultAdmin = "admin"
	}
	// Only set password if explicitly provided; otherwise leave empty
	// so that auth.go generates a secure random password
	if authOpts.DefaultPassword != "" {
		authConfig.DefaultPassword = authOpts.DefaultPassword
	}
	// Note: If DefaultPassword is empty, NewAuthManager will generate a secure random password

	authManager := NewAuthManager(authConfig)

	if authOpts.Disabled {
		fmt.Printf("  Authentication: Disabled (WARNING: not recommended for production)\n")
	} else {
		fmt.Printf("  Authentication: Enabled (mode: %s)\n", authOpts.Mode)
	}

	if authOpts.EmbeddedLLM {
		fmt.Printf("  Embedded LLM: Enabled (LLM settings locked)\n")
	}

	// Initialize session store for AI conversation history
	sessionStore, err := session.NewStore()
	if err != nil {
		fmt.Printf("  Session Store: Failed to initialize (%v)\n", err)
	} else {
		fmt.Printf("  Session Store: Ready\n")
	}

	server := &Server{
		cfg:              cfg,
		aiClient:         aiClient,
		k8sClient:        k8sClient,
		helmClient:       helmClient,
		mcpClient:        mcp.NewClient(),
		authManager:      authManager,
		sessionStore:     sessionStore,
		port:             port,
		embeddedLLM:      authOpts.EmbeddedLLM,
		versionInfo:      versionInfo,
		pendingApprovals: make(map[string]*PendingToolApproval),
	}

	server.reportGenerator = NewReportGenerator(server)
	fmt.Printf("  Reports: Ready\n")

	// Metrics collector is disabled by default to avoid performance issues
	// It can still collect on-demand via /api/metrics/collect
	fmt.Printf("  Metrics Collector: Disabled (on-demand collection available)\n")

	// Initialize security scanner
	server.securityScanner = security.NewScanner(k8sClient)
	scannerInfo := "Basic checks"
	if server.securityScanner.TrivyAvailable() {
		scannerInfo += ", Trivy"
	}
	if server.securityScanner.KubeBenchAvailable() {
		scannerInfo += ", kube-bench"
	}
	fmt.Printf("  Security Scanner: Ready (%s)\n", scannerInfo)

	// Initialize MCP servers
	server.initMCPServers()

	return server, nil
}

// initMCPServers connects to all enabled MCP servers
func (s *Server) initMCPServers() {
	enabledServers := s.cfg.GetEnabledMCPServers()
	if len(enabledServers) == 0 {
		fmt.Printf("  MCP Servers: None configured\n")
		return
	}

	fmt.Printf("  MCP Servers: Connecting...\n")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, serverCfg := range enabledServers {
		if err := s.mcpClient.Connect(ctx, serverCfg); err != nil {
			fmt.Printf("    - %s: Failed (%v)\n", serverCfg.Name, err)
		} else {
			fmt.Printf("    - %s: Connected\n", serverCfg.Name)
			// Register MCP tools with AI client
			s.registerMCPTools(serverCfg.Name)
		}
	}
}

// registerMCPTools registers tools from an MCP server with the AI client
func (s *Server) registerMCPTools(serverName string) {
	if s.aiClient == nil {
		return
	}

	mcpTools := s.mcpClient.GetAllTools()
	registry := s.aiClient.GetToolRegistry()

	// Set the MCP executor if not already set
	registry.SetMCPExecutor(mcp.NewMCPToolExecutor(s.mcpClient))

	for _, tool := range mcpTools {
		if tool.ServerName == serverName {
			registry.RegisterMCPTool(tool.Name, tool.Description, tool.ServerName, tool.InputSchema)
		}
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Public routes (no auth required)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/version", s.handleVersion)
	mux.HandleFunc("/api/auth/login", s.authManager.HandleLogin)
	mux.HandleFunc("/api/auth/logout", s.authManager.HandleLogout)
	mux.HandleFunc("/api/auth/kubeconfig", s.authManager.HandleKubeconfigLogin)
	mux.HandleFunc("/api/auth/status", s.authManager.HandleAuthStatus)
	mux.HandleFunc("/api/auth/csrf-token", s.authManager.HandleCSRFToken)
	// OIDC/SSO routes (public for OAuth flow)
	mux.HandleFunc("/api/auth/oidc/login", s.authManager.HandleOIDCLogin)
	mux.HandleFunc("/api/auth/oidc/callback", s.authManager.HandleOIDCCallback)
	mux.HandleFunc("/api/auth/oidc/status", s.authManager.HandleOIDCStatus)

	// Protected routes
	mux.HandleFunc("/api/auth/me", s.authManager.AuthMiddleware(s.authManager.HandleCurrentUser))
	// All chat requests use agentic mode
	mux.HandleFunc("/api/chat/agentic", s.authManager.AuthMiddleware(s.handleAgenticChat))
	mux.HandleFunc("/api/tool/approve", s.authManager.AuthMiddleware(s.handleToolApprove))

	// Session management for conversation history
	mux.HandleFunc("/api/sessions", s.authManager.AuthMiddleware(s.handleSessions))
	mux.HandleFunc("/api/sessions/", s.authManager.AuthMiddleware(s.handleSession))
	mux.HandleFunc("/api/k8s/", s.authManager.AuthMiddleware(s.handleK8sResource))
	mux.HandleFunc("/api/crd/", s.authManager.AuthMiddleware(s.handleCustomResources))
	mux.HandleFunc("/api/audit", s.authManager.AuthMiddleware(s.handleAuditLogs))
	mux.HandleFunc("/api/reports", s.authManager.AuthMiddleware(s.reportGenerator.HandleReports))
	mux.HandleFunc("/api/reports/preview", s.authManager.AuthMiddleware(s.reportGenerator.HandleReportPreview))
	mux.HandleFunc("/api/settings", s.authManager.AuthMiddleware(s.handleSettings))
	mux.HandleFunc("/api/settings/llm", s.authManager.AuthMiddleware(s.handleLLMSettings))

	// LLM connection test endpoint
	mux.HandleFunc("/api/llm/test", s.authManager.AuthMiddleware(s.handleLLMTest))
	mux.HandleFunc("/api/llm/status", s.authManager.AuthMiddleware(s.handleLLMStatus))
	mux.HandleFunc("/api/ai/ping", s.authManager.AuthMiddleware(s.handleAIPing))

	// Ollama helper endpoints
	mux.HandleFunc("/api/llm/ollama/status", s.authManager.AuthMiddleware(s.handleOllamaStatus))
	mux.HandleFunc("/api/llm/ollama/pull", s.authManager.AuthMiddleware(s.handleOllamaPull))

	// K8s safety analysis endpoint (guardrails)
	mux.HandleFunc("/api/safety/analyze", s.authManager.AuthMiddleware(s.handleSafetyAnalysis))

	// LLM usage tracking endpoints
	mux.HandleFunc("/api/llm/usage", s.authManager.AuthMiddleware(s.handleLLMUsage))
	mux.HandleFunc("/api/llm/usage/stats", s.authManager.AuthMiddleware(s.handleLLMUsageStats))

	// Model management endpoints
	mux.HandleFunc("/api/models", s.authManager.AuthMiddleware(s.handleModels))
	mux.HandleFunc("/api/models/active", s.authManager.AuthMiddleware(s.handleActiveModel))

	// MCP server management endpoints
	mux.HandleFunc("/api/mcp/servers", s.authManager.AuthMiddleware(s.handleMCPServers))
	mux.HandleFunc("/api/mcp/tools", s.authManager.AuthMiddleware(s.handleMCPTools))

	// WebSocket terminal handler
	terminalHandler := NewTerminalHandler(s.k8sClient)
	mux.HandleFunc("/api/terminal/", s.authManager.AuthMiddleware(terminalHandler.HandleTerminal))

	// Metrics endpoints (real-time)
	mux.HandleFunc("/api/metrics/pods", s.authManager.AuthMiddleware(s.handlePodMetrics))
	mux.HandleFunc("/api/metrics/nodes", s.authManager.AuthMiddleware(s.handleNodeMetrics))

	// Time-series metrics endpoints (historical)
	mux.HandleFunc("/api/metrics/history/cluster", s.authManager.AuthMiddleware(s.handleClusterMetricsHistory))
	mux.HandleFunc("/api/metrics/history/nodes", s.authManager.AuthMiddleware(s.handleNodeMetricsHistory))
	mux.HandleFunc("/api/metrics/history/pods", s.authManager.AuthMiddleware(s.handlePodMetricsHistory))
	mux.HandleFunc("/api/metrics/history/summary", s.authManager.AuthMiddleware(s.handleMetricsSummary))
	mux.HandleFunc("/api/metrics/history/aggregated", s.authManager.AuthMiddleware(s.handleAggregatedMetrics))
	mux.HandleFunc("/api/metrics/collect", s.authManager.AuthMiddleware(s.handleMetricsCollectNow))

	// Prometheus integration endpoints
	if s.cfg.Prometheus.ExposeMetrics {
		mux.HandleFunc("/metrics", s.handlePrometheusMetrics) // No auth for Prometheus scraping
	}
	mux.HandleFunc("/api/prometheus/settings", s.authManager.AuthMiddleware(s.handlePrometheusSettings))
	mux.HandleFunc("/api/prometheus/test", s.authManager.AuthMiddleware(s.handlePrometheusTest))
	mux.HandleFunc("/api/prometheus/query", s.authManager.AuthMiddleware(s.handlePrometheusQuery))

	// Security scanning endpoints
	mux.HandleFunc("/api/security/scan", s.authManager.AuthMiddleware(s.handleSecurityScan))
	mux.HandleFunc("/api/security/scan/quick", s.authManager.AuthMiddleware(s.handleSecurityQuickScan))
	mux.HandleFunc("/api/security/scans", s.authManager.AuthMiddleware(s.handleSecurityScanHistory))
	mux.HandleFunc("/api/security/scans/stats", s.authManager.AuthMiddleware(s.handleSecurityScanStats))
	mux.HandleFunc("/api/security/scan/", s.authManager.AuthMiddleware(s.handleSecurityScanDetail))

	// Trivy CVE scanner management
	mux.HandleFunc("/api/security/trivy/status", s.authManager.AuthMiddleware(s.handleTrivyStatus))
	mux.HandleFunc("/api/security/trivy/install", s.authManager.AuthMiddleware(s.handleTrivyInstall))
	mux.HandleFunc("/api/security/trivy/instructions", s.authManager.AuthMiddleware(s.handleTrivyInstructions))

	// Port forwarding endpoints
	mux.HandleFunc("/api/portforward/start", s.authManager.AuthMiddleware(s.handlePortForwardStart))
	mux.HandleFunc("/api/portforward/list", s.authManager.AuthMiddleware(s.handlePortForwardList))
	mux.HandleFunc("/api/portforward/", s.authManager.AuthMiddleware(s.handlePortForwardStop))

	// Deployment operations
	mux.HandleFunc("/api/deployment/scale", s.authManager.AuthMiddleware(s.handleDeploymentScale))
	mux.HandleFunc("/api/deployment/restart", s.authManager.AuthMiddleware(s.handleDeploymentRestart))
	mux.HandleFunc("/api/deployment/pause", s.authManager.AuthMiddleware(s.handleDeploymentPause))
	mux.HandleFunc("/api/deployment/resume", s.authManager.AuthMiddleware(s.handleDeploymentResume))
	mux.HandleFunc("/api/deployment/rollback", s.authManager.AuthMiddleware(s.handleDeploymentRollback))
	mux.HandleFunc("/api/deployment/history", s.authManager.AuthMiddleware(s.handleDeploymentHistory))

	// StatefulSet operations
	mux.HandleFunc("/api/statefulset/scale", s.authManager.AuthMiddleware(s.handleStatefulSetScale))
	mux.HandleFunc("/api/statefulset/restart", s.authManager.AuthMiddleware(s.handleStatefulSetRestart))

	// DaemonSet operations
	mux.HandleFunc("/api/daemonset/restart", s.authManager.AuthMiddleware(s.handleDaemonSetRestart))

	// CronJob operations
	mux.HandleFunc("/api/cronjob/trigger", s.authManager.AuthMiddleware(s.handleCronJobTrigger))
	mux.HandleFunc("/api/cronjob/suspend", s.authManager.AuthMiddleware(s.handleCronJobSuspend))

	// Node operations
	mux.HandleFunc("/api/node/cordon", s.authManager.AuthMiddleware(s.handleNodeCordon))
	mux.HandleFunc("/api/node/drain", s.authManager.AuthMiddleware(s.handleNodeDrain))
	mux.HandleFunc("/api/node/pods", s.authManager.AuthMiddleware(s.handleNodePods))

	// Pod logs endpoint
	mux.HandleFunc("/api/pods/", s.authManager.AuthMiddleware(s.handlePodLogs))

	// Workload pods endpoint (get pods for deployment, daemonset, statefulset, replicaset)
	mux.HandleFunc("/api/workload/pods", s.authManager.AuthMiddleware(s.handleWorkloadPods))

	// Cluster overview endpoint
	mux.HandleFunc("/api/overview", s.authManager.AuthMiddleware(s.handleClusterOverview))

	// Global search endpoint
	mux.HandleFunc("/api/search", s.authManager.AuthMiddleware(s.handleGlobalSearch))

	// YAML apply endpoint
	mux.HandleFunc("/api/k8s/apply", s.authManager.AuthMiddleware(s.handleYamlApply))

	// Helm operations
	mux.HandleFunc("/api/helm/releases", s.authManager.AuthMiddleware(s.handleHelmReleases))
	mux.HandleFunc("/api/helm/release/", s.authManager.AuthMiddleware(s.handleHelmRelease))
	mux.HandleFunc("/api/helm/install", s.authManager.AuthMiddleware(s.handleHelmInstall))
	mux.HandleFunc("/api/helm/upgrade", s.authManager.AuthMiddleware(s.handleHelmUpgrade))
	mux.HandleFunc("/api/helm/uninstall", s.authManager.AuthMiddleware(s.handleHelmUninstall))
	mux.HandleFunc("/api/helm/rollback", s.authManager.AuthMiddleware(s.handleHelmRollback))
	mux.HandleFunc("/api/helm/repos", s.authManager.AuthMiddleware(s.handleHelmRepos))
	mux.HandleFunc("/api/helm/search", s.authManager.AuthMiddleware(s.handleHelmSearch))

	// Admin-only endpoints (user management)
	mux.HandleFunc("/api/admin/users", s.authManager.AuthMiddleware(s.authManager.AdminMiddleware(s.handleAdminUsers)))
	mux.HandleFunc("/api/admin/users/", s.authManager.AuthMiddleware(s.authManager.AdminMiddleware(s.handleAdminUserAction)))
	mux.HandleFunc("/api/admin/reset-password", s.authManager.AuthMiddleware(s.authManager.AdminMiddleware(s.authManager.HandleResetPassword)))
	mux.HandleFunc("/api/admin/status", s.authManager.AuthMiddleware(s.authManager.AdminMiddleware(s.authManager.HandleAuthStatus)))

	// Static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		mux.Handle("/", http.FileServer(http.Dir("pkg/web/static")))
	} else {
		mux.Handle("/", http.FileServer(http.FS(staticFS)))
	}

	// Apply middleware chain: security headers -> CORS -> CSRF -> handler
	handler := securityHeadersMiddleware(corsMiddleware(s.authManager.CSRFMiddleware(mux)))

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: handler,
	}

	fmt.Printf("\n  Web server started at http://localhost:%d\n", s.port)
	return s.server.ListenAndServe()
}

func (s *Server) Stop() error {
	// Disconnect all MCP servers
	if s.mcpClient != nil {
		s.mcpClient.DisconnectAll()
	}

	db.Close()
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Only allow same-origin requests or explicitly trusted origins
		// In production, configure allowed origins via environment variable
		allowedOrigins := map[string]bool{
			"":                       true, // Same origin (no Origin header)
			"http://localhost:8080":  true,
			"http://localhost:3000":  true,
			"http://127.0.0.1:8080":  true,
			"https://localhost:8080": true,
		}

		if origin != "" {
			if allowedOrigins[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			// If origin not allowed, don't set CORS headers (browser will block)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// securityHeadersMiddleware adds security headers to all responses
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// XSS protection (legacy but still useful)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
				"font-src 'self' https://fonts.gstatic.com; "+
				"img-src 'self' data:; "+
				"connect-src 'self' ws: wss:; "+
				"frame-ancestors 'none'")

		// Permissions Policy (formerly Feature-Policy)
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	version := "dev"
	if s.versionInfo != nil && s.versionInfo.Version != "" {
		version = s.versionInfo.Version
	}

	status := map[string]interface{}{
		"status":       "ok",
		"timestamp":    time.Now(),
		"ai_ready":     s.aiClient != nil && s.aiClient.IsReady(),
		"k8s_ready":    s.k8sClient != nil,
		"db_ready":     db.DB != nil,
		"auth_enabled": s.authManager.config.Enabled,
		"auth_mode":    s.authManager.GetAuthMode(),
		"version":      version,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"version":    "dev",
		"build_time": "unknown",
		"git_commit": "unknown",
		"go_version": "",
	}

	if s.versionInfo != nil {
		if s.versionInfo.Version != "" {
			info["version"] = s.versionInfo.Version
		}
		if s.versionInfo.BuildTime != "" {
			info["build_time"] = s.versionInfo.BuildTime
		}
		if s.versionInfo.GitCommit != "" {
			info["git_commit"] = s.versionInfo.GitCommit
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
