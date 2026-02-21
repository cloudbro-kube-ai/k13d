package web

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
	"github.com/cloudbro-kube-ai/k13d/pkg/ai/session"
	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/db"
	"github.com/cloudbro-kube-ai/k13d/pkg/helm"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	"github.com/cloudbro-kube-ai/k13d/pkg/mcp"
	"github.com/cloudbro-kube-ai/k13d/pkg/metrics"
	"github.com/cloudbro-kube-ai/k13d/pkg/security"
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
	authorizer       *Authorizer // RBAC authorizer (Teleport-inspired)
	reportGenerator  *ReportGenerator
	metricsCollector *metrics.Collector
	securityScanner  *security.Scanner
	sessionStore     *session.Store // AI conversation session storage
	port             int
	server           *http.Server
	embeddedLLM      bool // Using embedded LLM server
	versionInfo      *VersionInfo

	// Protects concurrent access to aiClient and cfg.LLM
	aiMu sync.RWMutex

	// Tool approval management
	pendingApprovals     map[string]*PendingToolApproval
	pendingApprovalMutex sync.RWMutex

	// Access request management (Teleport-inspired)
	accessRequestManager *AccessRequestManager

	// Rate limiters
	apiRateLimiter  *RateLimiter
	authRateLimiter *RateLimiter

	// Self-healing rules store
	healingStore *HealingStore

	// Notification manager
	notifManager *NotificationManager

	// Port forwarding sessions
	portForwardSessions map[string]*PortForwardSession
	pfMutex             sync.Mutex
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
	// Default auth config: use audit flag to control auth
	authConfig := &AuthConfig{
		Enabled:         cfg.EnableAudit,
		AuthMode:        "local",
		SessionDuration: 24 * time.Hour,
		DefaultAdmin:    "admin",
		// DefaultPassword: intentionally empty for secure random generation
	}

	return newServer(cfg, port, authConfig, false, versionInfo)
}

// NewServerWithAuth creates a new server with custom authentication options
func NewServerWithAuth(cfg *config.Config, port int, authOpts *AuthOptions, versionInfo *VersionInfo) (*Server, error) {
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

	return newServer(cfg, port, authConfig, authOpts.EmbeddedLLM, versionInfo)
}

// newServer contains the shared initialization logic for both constructors.
func newServer(cfg *config.Config, port int, authConfig *AuthConfig, embeddedLLM bool, versionInfo *VersionInfo) (*Server, error) {
	var aiClient *ai.Client
	var err error

	fmt.Printf("Starting k13d web server...\n")

	// Load LLM settings from SQLite if available (web UI takes precedence over YAML)
	if llmSettings, dbErr := db.GetWebSettingsWithPrefix("llm."); dbErr == nil && len(llmSettings) > 0 {
		if v, ok := llmSettings["llm.provider"]; ok && v != "" {
			cfg.LLM.Provider = v
		}
		if v, ok := llmSettings["llm.model"]; ok && v != "" {
			cfg.LLM.Model = v
		}
		if v, ok := llmSettings["llm.endpoint"]; ok && v != "" {
			cfg.LLM.Endpoint = v
		}
		if v, ok := llmSettings["llm.api_key"]; ok && v != "" {
			cfg.LLM.APIKey = v
		}
		if v, ok := llmSettings["llm.use_json_mode"]; ok {
			cfg.LLM.UseJSONMode = v == "true"
		}
		if v, ok := llmSettings["llm.reasoning_effort"]; ok && v != "" {
			cfg.LLM.ReasoningEffort = v
		}
		fmt.Printf("  LLM Settings: Loaded from SQLite\n")
	}

	// Load tool approval settings from SQLite if available
	if taSettings, dbErr := db.GetWebSettingsWithPrefix("tool_approval."); dbErr == nil && len(taSettings) > 0 {
		if v, ok := taSettings["tool_approval.auto_approve_read_only"]; ok && v != "" {
			cfg.Authorization.ToolApproval.AutoApproveReadOnly = v == "true"
		}
		if v, ok := taSettings["tool_approval.require_approval_for_write"]; ok && v != "" {
			cfg.Authorization.ToolApproval.RequireApprovalForWrite = v == "true"
		}
		if v, ok := taSettings["tool_approval.require_approval_for_unknown"]; ok && v != "" {
			cfg.Authorization.ToolApproval.RequireApprovalForUnknown = v == "true"
		}
		if v, ok := taSettings["tool_approval.block_dangerous"]; ok && v != "" {
			cfg.Authorization.ToolApproval.BlockDangerous = v == "true"
		}
		if v, ok := taSettings["tool_approval.approval_timeout_seconds"]; ok && v != "" {
			if seconds := parseIntSafe(v, 60); seconds > 0 && seconds <= 600 {
				cfg.Authorization.ToolApproval.ApprovalTimeoutSeconds = seconds
			}
		}
		if v, ok := taSettings["tool_approval.blocked_patterns"]; ok && v != "" {
			cfg.Authorization.ToolApproval.BlockedPatterns = strings.Split(v, "\n")
		}
		fmt.Printf("  Tool Approval Settings: Loaded from SQLite\n")
	}

	// Load agent loop settings from SQLite if available
	if agentSettings, dbErr := db.GetWebSettingsWithPrefix("agent."); dbErr == nil && len(agentSettings) > 0 {
		if v, ok := agentSettings["agent.max_iterations"]; ok && v != "" {
			if n, parseErr := strconv.Atoi(v); parseErr == nil && n >= 1 && n <= 30 {
				cfg.LLM.MaxIterations = n
			}
		}
		if v, ok := agentSettings["agent.temperature"]; ok && v != "" {
			if f, parseErr := strconv.ParseFloat(v, 64); parseErr == nil && f >= 0 && f <= 2.0 {
				cfg.LLM.Temperature = f
			}
		}
		if v, ok := agentSettings["agent.max_tokens"]; ok && v != "" {
			if n, parseErr := strconv.Atoi(v); parseErr == nil && n >= 0 {
				cfg.LLM.MaxTokens = n
			}
		}
		if v, ok := agentSettings["agent.reasoning_effort"]; ok && v != "" {
			cfg.LLM.ReasoningEffort = v
		}
		fmt.Printf("  Agent Settings: Loaded from SQLite\n")
	}

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

	// Initialize database (skip if already initialized by main)
	if db.DB == nil {
		if err := db.Init(""); err != nil {
			fmt.Printf("  Database: Failed to initialize (%v)\n", err)
		} else {
			fmt.Printf("  Database: Ready\n")
		}
	} else {
		fmt.Printf("  Database: Ready (pre-initialized)\n")
	}

	// Initialize auth manager
	authManager := NewAuthManager(authConfig)
	if authConfig.Enabled {
		fmt.Printf("  Authentication: Enabled (mode: %s)\n", authConfig.AuthMode)
	} else {
		fmt.Printf("  Authentication: Disabled\n")
	}

	if embeddedLLM {
		fmt.Printf("  Embedded LLM: Enabled (LLM settings locked)\n")
	}

	// Initialize session store for AI conversation history
	sessionStore, err := session.NewStore()
	if err != nil {
		fmt.Printf("  Session Store: Failed to initialize (%v)\n", err)
	} else {
		fmt.Printf("  Session Store: Ready\n")
	}

	// Initialize RBAC authorizer (Teleport-inspired)
	authorizer := NewAuthorizer()
	fmt.Printf("  RBAC Authorizer: Ready (roles: admin, user, viewer)\n")

	// Initialize access request manager (Teleport-inspired)
	accessReqManager := NewAccessRequestManager(30 * time.Minute)
	fmt.Printf("  Access Request Manager: Ready (TTL: 30m)\n")

	// Initialize rate limiters
	apiRateLimiter := NewRateLimiter(600, 1*time.Minute) // 600 requests per minute for API (dashboard makes many concurrent calls)
	authRateLimiter := NewRateLimiter(10, 1*time.Minute) // 10 requests per minute for auth
	fmt.Printf("  Rate Limiting: API (600/min), Auth (10/min)\n")

	server := &Server{
		cfg:                  cfg,
		aiClient:             aiClient,
		k8sClient:            k8sClient,
		helmClient:           helmClient,
		mcpClient:            mcp.NewClient(),
		authManager:          authManager,
		authorizer:           authorizer,
		accessRequestManager: accessReqManager,
		sessionStore:         sessionStore,
		port:                 port,
		embeddedLLM:          embeddedLLM,
		versionInfo:          versionInfo,
		pendingApprovals:     make(map[string]*PendingToolApproval),
		apiRateLimiter:       apiRateLimiter,
		authRateLimiter:      authRateLimiter,
		healingStore:         NewHealingStore(),
		portForwardSessions:  make(map[string]*PortForwardSession),
	}

	server.reportGenerator = NewReportGenerator(server)
	fmt.Printf("  Reports: Ready\n")

	// Initialize notification manager
	server.notifManager = NewNotificationManager(k8sClient, cfg)
	if cfg.Notifications.Enabled {
		server.notifManager.Start()
		fmt.Printf("  Notifications: Enabled (provider: %s, poll: %ds)\n",
			cfg.Notifications.Provider, cfg.Notifications.PollInterval)
	} else {
		fmt.Printf("  Notifications: Disabled\n")
	}
	// Sync in-memory notifConfig from persistent config
	notifConfigMu.Lock()
	notifConfig = &NotificationConfig{
		Enabled:    cfg.Notifications.Enabled,
		WebhookURL: cfg.Notifications.WebhookURL,
		Channel:    cfg.Notifications.Channel,
		Events:     cfg.Notifications.Events,
		Provider:   cfg.Notifications.Provider,
	}
	notifConfigMu.Unlock()

	// Initialize and start metrics collector for historical charts
	metricsCollector, err := metrics.NewCollector(k8sClient, metrics.DefaultConfig())
	if err != nil {
		fmt.Printf("  Metrics Collector: Failed to initialize (%v)\n", err)
	} else {
		server.metricsCollector = metricsCollector
		metricsCollector.Start()
		fmt.Printf("  Metrics Collector: Running (interval: 1m, retention: 7d)\n")
	}

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

	// Load persisted user locks
	authManager.LoadUserLocks()

	// Load custom roles from DB and register with authorizer
	server.loadCustomRoles()

	// Set role validator on auth manager so user creation accepts custom roles
	authManager.SetRoleValidator(func(role string) bool {
		return server.authorizer.GetRole(role) != nil
	})

	return server, nil
}

// loadCustomRoles loads custom roles from the database and registers them with the authorizer
func (s *Server) loadCustomRoles() {
	rows, err := db.ListCustomRoles()
	if err != nil {
		fmt.Printf("  Custom Roles: Failed to load (%v)\n", err)
		return
	}
	if len(rows) == 0 {
		return
	}
	for _, row := range rows {
		var role RoleDefinition
		if err := json.Unmarshal([]byte(row.Definition), &role); err != nil {
			fmt.Printf("  Custom Role %s: Failed to parse (%v)\n", row.Name, err)
			continue
		}
		role.IsCustom = true
		s.authorizer.RegisterRole(&role)
	}
	fmt.Printf("  Custom Roles: Loaded %d\n", len(rows))
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

// recoveryMiddleware wraps a handler to catch and handle panics
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				fmt.Printf("PANIC in HTTP handler: %v\nPath: %s %s\n", err, r.Method, r.URL.Path)

				// Audit the panic event
				if db.DB != nil {
					username := r.Header.Get("X-Username")
					if username == "" {
						username = "anonymous"
					}
					_ = db.RecordAudit(db.AuditEntry{
						User:       username,
						Action:     "http_panic",
						Resource:   r.URL.Path,
						Details:    fmt.Sprintf("Panic recovered: %v", err),
						ActionType: db.ActionTypeMutation,
						Source:     "web",
						Success:    false,
					})
				}

				// Return 500 error to client
				WriteError(w, NewAPIError(ErrCodeInternalError, "An unexpected error occurred"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// withRecovery wraps a HandlerFunc with panic recovery
func withRecovery(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				fmt.Printf("PANIC in HTTP handler: %v\nPath: %s %s\n", err, r.Method, r.URL.Path)

				// Audit the panic event
				if db.DB != nil {
					username := r.Header.Get("X-Username")
					if username == "" {
						username = "anonymous"
					}
					_ = db.RecordAudit(db.AuditEntry{
						User:       username,
						Action:     "http_panic",
						Resource:   r.URL.Path,
						Details:    fmt.Sprintf("Panic recovered: %v", err),
						ActionType: db.ActionTypeMutation,
						Source:     "web",
						Success:    false,
					})
				}

				// Return 500 error to client
				WriteError(w, NewAPIError(ErrCodeInternalError, "An unexpected error occurred"))
			}
		}()
		handler(w, r)
	}
}

// requestLoggingMiddleware logs all HTTP requests
func requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(rw, r)

		// Log request (exclude health checks to reduce noise)
		if r.URL.Path != "/api/health" {
			duration := time.Since(start)
			username := r.Header.Get("X-Username")
			if username == "" {
				username = "anonymous"
			}

			fmt.Printf("[%s] %s %s - %d (%s) - User: %s\n",
				start.Format("2006-01-02 15:04:05"),
				r.Method,
				r.URL.Path,
				rw.statusCode,
				duration,
				username,
			)
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Flush implements http.Flusher so SSE streaming works through the logging middleware.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker so WebSocket upgrades work through the logging middleware.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}

// doneWriter wraps http.ResponseWriter to prevent writes after timeout.
type doneWriter struct {
	http.ResponseWriter
	mu         sync.Mutex
	headerSent bool
	timedOut   bool
}

func (dw *doneWriter) WriteHeader(code int) {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	if dw.headerSent || dw.timedOut {
		return
	}
	dw.headerSent = true
	dw.ResponseWriter.WriteHeader(code)
}

func (dw *doneWriter) Write(b []byte) (int, error) {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	if dw.timedOut {
		return 0, nil
	}
	return dw.ResponseWriter.Write(b)
}

// timeoutMiddleware adds request timeouts to prevent hanging requests
func timeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip timeout for WebSocket connections and streaming endpoints
			if r.Header.Get("Upgrade") == "websocket" ||
				r.Header.Get("Accept") == "text/event-stream" ||
				r.URL.Path == "/api/chat/agentic" { // AI streaming responses
				next.ServeHTTP(w, r)
				return
			}

			// Create timeout context
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Wrap ResponseWriter to prevent writes after timeout
			dw := &doneWriter{ResponseWriter: w}

			// Channel to signal completion
			done := make(chan struct{})

			// Process request in goroutine
			go func() {
				next.ServeHTTP(dw, r.WithContext(ctx))
				close(done)
			}()

			// Wait for completion or timeout
			select {
			case <-done:
				// Request completed successfully
			case <-ctx.Done():
				// Timeout occurred — block further writes from handler goroutine
				dw.mu.Lock()
				dw.timedOut = true
				headerSent := dw.headerSent
				dw.mu.Unlock()
				if !headerSent && ctx.Err() == context.DeadlineExceeded {
					WriteError(w, NewAPIError(ErrCodeTimeout, "Request timed out"))
				}
			}
		})
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// --- Public routes (no auth required) ---
	mux.HandleFunc("/api/health", withRecovery(s.handleHealth))
	mux.HandleFunc("/api/version", s.handleVersion)

	// Authentication (public for login/logout flow)
	mux.HandleFunc("/api/auth/login", s.authManager.HandleLogin)
	mux.HandleFunc("/api/auth/logout", s.authManager.HandleLogout)
	mux.HandleFunc("/api/auth/kubeconfig", s.authManager.HandleKubeconfigLogin)
	mux.HandleFunc("/api/auth/status", s.authManager.HandleAuthStatus)
	mux.HandleFunc("/api/auth/csrf-token", s.authManager.HandleCSRFToken)

	// OIDC/SSO (public for OAuth flow)
	mux.HandleFunc("/api/auth/oidc/login", s.authManager.HandleOIDCLogin)
	mux.HandleFunc("/api/auth/oidc/callback", s.authManager.HandleOIDCCallback)
	mux.HandleFunc("/api/auth/oidc/status", s.authManager.HandleOIDCStatus)

	// Prometheus scrape endpoint (no auth for scraping)
	if s.cfg.Prometheus.ExposeMetrics {
		mux.HandleFunc("/metrics", s.handlePrometheusMetrics)
	}

	// --- Protected routes (auth required) ---

	// Auth - current user
	mux.HandleFunc("/api/auth/me", s.authManager.AuthMiddleware(s.authManager.HandleCurrentUser))

	// AI chat and tool approval (feature-gated)
	mux.HandleFunc("/api/chat/agentic", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureAIAssistant)(s.handleAgenticChat)))
	mux.HandleFunc("/api/tool/approve", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureAIAssistant)(s.handleToolApprove)))

	// AI session management
	mux.HandleFunc("/api/sessions", s.authManager.AuthMiddleware(s.handleSessions))
	mux.HandleFunc("/api/sessions/", s.authManager.AuthMiddleware(s.handleSession))

	// AI / LLM configuration and status
	mux.HandleFunc("/api/settings", s.authManager.AuthMiddleware(s.handleSettings))
	mux.HandleFunc("/api/settings/llm", s.authManager.AuthMiddleware(s.handleLLMSettings))
	mux.HandleFunc("/api/settings/agent", s.authManager.AuthMiddleware(s.handleAgentSettings))
	mux.HandleFunc("/api/llm/test", s.authManager.AuthMiddleware(s.handleLLMTest))
	mux.HandleFunc("/api/llm/status", s.authManager.AuthMiddleware(s.handleLLMStatus))
	mux.HandleFunc("/api/ai/ping", s.authManager.AuthMiddleware(s.handleAIPing))
	mux.HandleFunc("/api/llm/ollama/status", s.authManager.AuthMiddleware(s.handleOllamaStatus))
	mux.HandleFunc("/api/llm/ollama/pull", s.authManager.AuthMiddleware(s.handleOllamaPull))
	mux.HandleFunc("/api/llm/usage", s.authManager.AuthMiddleware(s.handleLLMUsage))
	mux.HandleFunc("/api/llm/usage/stats", s.authManager.AuthMiddleware(s.handleLLMUsageStats))
	mux.HandleFunc("/api/llm/available-models", s.authManager.AuthMiddleware(s.handleAvailableModels))
	mux.HandleFunc("/api/models", s.authManager.AuthMiddleware(s.handleModels))
	mux.HandleFunc("/api/models/active", s.authManager.AuthMiddleware(s.handleActiveModel))

	// MCP server management
	mux.HandleFunc("/api/mcp/servers", s.authManager.AuthMiddleware(s.handleMCPServers))
	mux.HandleFunc("/api/mcp/tools", s.authManager.AuthMiddleware(s.handleMCPTools))

	// Kubernetes resources (read-only, RBAC view)
	mux.HandleFunc("/api/k8s/apply", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("*", ActionApply)(s.handleYamlApply)))
	mux.HandleFunc("/api/k8s/", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("*", ActionView)(s.handleK8sResource)))
	mux.HandleFunc("/api/crd/", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("*", ActionView)(s.handleCustomResources)))
	mux.HandleFunc("/api/overview", s.authManager.AuthMiddleware(s.handleClusterOverview))
	mux.HandleFunc("/api/applications", s.authManager.AuthMiddleware(s.handleApplications))
	mux.HandleFunc("/api/topology/", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureTopology)(s.handleTopology)))
	mux.HandleFunc("/api/cost", s.authManager.AuthMiddleware(s.handleCostEstimate))
	mux.HandleFunc("/api/healing/rules", s.authManager.AuthMiddleware(s.handleHealingRules))
	mux.HandleFunc("/api/healing/events", s.authManager.AuthMiddleware(s.handleHealingEvents))
	mux.HandleFunc("/api/search", s.authManager.AuthMiddleware(s.handleGlobalSearch))
	mux.HandleFunc("/api/safety/analyze", s.authManager.AuthMiddleware(s.handleSafetyAnalysis))
	mux.HandleFunc("/api/settings/tool-approval", s.authManager.AuthMiddleware(s.handleToolApprovalSettings))
	mux.HandleFunc("/api/validate", s.authManager.AuthMiddleware(s.handleValidate))
	mux.HandleFunc("/api/pulse", s.authManager.AuthMiddleware(s.handlePulse))
	mux.HandleFunc("/api/xray", s.authManager.AuthMiddleware(s.handleXRay))

	// Multi-cluster context management
	mux.HandleFunc("/api/contexts", s.authManager.AuthMiddleware(s.handleContexts))
	mux.HandleFunc("/api/contexts/switch", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("*", ActionEdit)(s.handleContextSwitch)))

	// RBAC visualization
	mux.HandleFunc("/api/rbac/visualization", s.authManager.AuthMiddleware(s.handleRBACVisualization))

	// Network Policy visualization
	mux.HandleFunc("/api/netpol/visualization", s.authManager.AuthMiddleware(s.handleNetworkPolicyVisualization))

	// GitOps status (ArgoCD / Flux)
	mux.HandleFunc("/api/gitops/status", s.authManager.AuthMiddleware(s.handleGitOpsStatus))

	// Resource templates (feature-gated)
	mux.HandleFunc("/api/templates", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureTemplates)(s.handleTemplates)))
	mux.HandleFunc("/api/templates/apply", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureTemplates)(s.authorizer.AuthzMiddleware("*", ActionApply)(s.handleTemplateApply))))

	// Notification webhook configuration
	mux.HandleFunc("/api/notifications/config", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("*", ActionEdit)(s.handleNotificationConfig)))
	mux.HandleFunc("/api/notifications/test", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("*", ActionEdit)(s.handleNotificationTest)))
	mux.HandleFunc("/api/notifications/history", s.authManager.AuthMiddleware(s.handleNotificationHistory))
	mux.HandleFunc("/api/notifications/status", s.authManager.AuthMiddleware(s.handleNotificationStatus))

	// AI troubleshooting
	mux.HandleFunc("/api/troubleshoot", s.authManager.AuthMiddleware(s.handleTroubleshoot))

	// Velero backup status
	mux.HandleFunc("/api/velero/backups", s.authManager.AuthMiddleware(s.handleVeleroBackups))
	mux.HandleFunc("/api/velero/schedules", s.authManager.AuthMiddleware(s.handleVeleroSchedules))

	// Resource diff
	mux.HandleFunc("/api/diff", s.authManager.AuthMiddleware(s.handleResourceDiff))

	// Event timeline (feature-gated)
	mux.HandleFunc("/api/events/timeline", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureEventTimeline)(s.handleEventTimeline)))

	// Pod operations
	mux.HandleFunc("/api/pods/", s.authManager.AuthMiddleware(s.handlePodLogs))
	mux.HandleFunc("/api/workload/pods", s.authManager.AuthMiddleware(s.handleWorkloadPods))

	// WebSocket terminal (feature-gated)
	terminalHandler := NewTerminalHandler(s.k8sClient)
	mux.HandleFunc("/api/terminal/", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureTerminal)(terminalHandler.HandleTerminal)))

	// Workload operations (RBAC-protected)
	mux.HandleFunc("/api/deployment/scale", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("deployments", ActionScale)(s.handleDeploymentScale)))
	mux.HandleFunc("/api/deployment/restart", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("deployments", ActionRestart)(s.handleDeploymentRestart)))
	mux.HandleFunc("/api/deployment/pause", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("deployments", ActionEdit)(s.handleDeploymentPause)))
	mux.HandleFunc("/api/deployment/resume", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("deployments", ActionEdit)(s.handleDeploymentResume)))
	mux.HandleFunc("/api/deployment/rollback", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("deployments", ActionEdit)(s.handleDeploymentRollback)))
	mux.HandleFunc("/api/deployment/history", s.authManager.AuthMiddleware(s.handleDeploymentHistory))
	mux.HandleFunc("/api/statefulset/scale", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("statefulsets", ActionScale)(s.handleStatefulSetScale)))
	mux.HandleFunc("/api/statefulset/restart", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("statefulsets", ActionRestart)(s.handleStatefulSetRestart)))
	mux.HandleFunc("/api/daemonset/restart", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("daemonsets", ActionRestart)(s.handleDaemonSetRestart)))
	mux.HandleFunc("/api/cronjob/trigger", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("cronjobs", ActionCreate)(s.handleCronJobTrigger)))
	mux.HandleFunc("/api/cronjob/suspend", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("cronjobs", ActionEdit)(s.handleCronJobSuspend)))

	// Node operations (RBAC-protected)
	mux.HandleFunc("/api/node/cordon", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("nodes", ActionEdit)(s.handleNodeCordon)))
	mux.HandleFunc("/api/node/drain", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("nodes", ActionEdit)(s.handleNodeDrain)))
	mux.HandleFunc("/api/node/pods", s.authManager.AuthMiddleware(s.handleNodePods))

	// Port forwarding (RBAC-protected)
	mux.HandleFunc("/api/portforward/start", s.authManager.AuthMiddleware(s.authorizer.AuthzMiddleware("pods", ActionPortForward)(s.handlePortForwardStart)))
	mux.HandleFunc("/api/portforward/list", s.authManager.AuthMiddleware(s.handlePortForwardList))
	mux.HandleFunc("/api/portforward/", s.authManager.AuthMiddleware(s.handlePortForwardStop))

	// Helm operations (RBAC-protected for mutations, feature-gated)
	mux.HandleFunc("/api/helm/releases", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureHelmManagement)(s.handleHelmReleases)))
	mux.HandleFunc("/api/helm/release/", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureHelmManagement)(s.handleHelmRelease)))
	mux.HandleFunc("/api/helm/install", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureHelmManagement)(s.authorizer.AuthzMiddleware("helm", ActionCreate)(s.handleHelmInstall))))
	mux.HandleFunc("/api/helm/upgrade", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureHelmManagement)(s.authorizer.AuthzMiddleware("helm", ActionEdit)(s.handleHelmUpgrade))))
	mux.HandleFunc("/api/helm/uninstall", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureHelmManagement)(s.authorizer.AuthzMiddleware("helm", ActionDelete)(s.handleHelmUninstall))))
	mux.HandleFunc("/api/helm/rollback", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureHelmManagement)(s.authorizer.AuthzMiddleware("helm", ActionEdit)(s.handleHelmRollback))))
	mux.HandleFunc("/api/helm/repos", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureHelmManagement)(s.handleHelmRepos)))
	mux.HandleFunc("/api/helm/search", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureHelmManagement)(s.handleHelmSearch)))

	// Metrics (real-time and historical, feature-gated)
	mux.HandleFunc("/api/metrics/pods", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureMetrics)(s.handlePodMetrics)))
	mux.HandleFunc("/api/metrics/nodes", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureMetrics)(s.handleNodeMetrics)))
	mux.HandleFunc("/api/metrics/history/cluster", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureMetrics)(s.handleClusterMetricsHistory)))
	mux.HandleFunc("/api/metrics/history/nodes", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureMetrics)(s.handleNodeMetricsHistory)))
	mux.HandleFunc("/api/metrics/history/pods", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureMetrics)(s.handlePodMetricsHistory)))
	mux.HandleFunc("/api/metrics/history/summary", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureMetrics)(s.handleMetricsSummary)))
	mux.HandleFunc("/api/metrics/history/aggregated", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureMetrics)(s.handleAggregatedMetrics)))
	mux.HandleFunc("/api/metrics/collect", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureMetrics)(s.handleMetricsCollectNow)))
	mux.HandleFunc("/api/prometheus/settings", s.authManager.AuthMiddleware(s.handlePrometheusSettings))
	mux.HandleFunc("/api/prometheus/test", s.authManager.AuthMiddleware(s.handlePrometheusTest))
	mux.HandleFunc("/api/prometheus/query", s.authManager.AuthMiddleware(s.handlePrometheusQuery))

	// Security scanning (feature-gated)
	mux.HandleFunc("/api/security/scan", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureSecurityScan)(s.handleSecurityScan)))
	mux.HandleFunc("/api/security/scan/quick", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureSecurityScan)(s.handleSecurityQuickScan)))
	mux.HandleFunc("/api/security/scans", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureSecurityScan)(s.handleSecurityScanHistory)))
	mux.HandleFunc("/api/security/scans/stats", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureSecurityScan)(s.handleSecurityScanStats)))
	mux.HandleFunc("/api/security/scan/", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureSecurityScan)(s.handleSecurityScanDetail)))
	mux.HandleFunc("/api/security/trivy/status", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureSecurityScan)(s.handleTrivyStatus)))
	mux.HandleFunc("/api/security/trivy/install", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureSecurityScan)(s.handleTrivyInstall)))
	mux.HandleFunc("/api/security/trivy/instructions", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureSecurityScan)(s.handleTrivyInstructions)))

	// Audit and reports (feature-gated)
	mux.HandleFunc("/api/audit", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureAuditLogs)(s.handleAuditLogs)))
	mux.HandleFunc("/api/reports", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureReports)(s.reportGenerator.HandleReports)))
	mux.HandleFunc("/api/reports/preview", s.authManager.AuthMiddleware(s.authorizer.FeatureMiddleware(FeatureReports)(s.reportGenerator.HandleReportPreview)))

	// Role management (admin-only for mutations, auth for read)
	mux.HandleFunc("/api/roles", s.authManager.AuthMiddleware(s.handleRoles))
	mux.HandleFunc("/api/roles/", s.authManager.AuthMiddleware(s.authManager.AdminMiddleware(s.handleRoleByName)))
	mux.HandleFunc("/api/auth/permissions", s.authManager.AuthMiddleware(s.handleUserPermissions))

	// --- Admin-only routes ---
	mux.HandleFunc("/api/admin/users", s.authManager.AuthMiddleware(s.authManager.AdminMiddleware(s.handleAdminUsers)))
	mux.HandleFunc("/api/admin/users/", s.authManager.AuthMiddleware(s.authManager.AdminMiddleware(s.handleAdminUserAction)))
	mux.HandleFunc("/api/admin/reset-password", s.authManager.AuthMiddleware(s.authManager.AdminMiddleware(s.authManager.HandleResetPassword)))
	mux.HandleFunc("/api/admin/status", s.authManager.AuthMiddleware(s.authManager.AdminMiddleware(s.authManager.HandleAuthStatus)))
	mux.HandleFunc("/api/admin/lock", s.authManager.AuthMiddleware(s.authManager.AdminMiddleware(s.authManager.HandleLockUser)))
	mux.HandleFunc("/api/admin/unlock", s.authManager.AuthMiddleware(s.authManager.AdminMiddleware(s.authManager.HandleUnlockUser)))

	// Access request workflow (Teleport-inspired)
	mux.HandleFunc("/api/access/request", s.authManager.AuthMiddleware(s.accessRequestManager.HandleCreateAccessRequest))
	mux.HandleFunc("/api/access/requests", s.authManager.AuthMiddleware(s.accessRequestManager.HandleListAccessRequests))
	mux.HandleFunc("/api/access/approve/", s.authManager.AuthMiddleware(s.authManager.AdminMiddleware(s.accessRequestManager.HandleApproveAccessRequest)))
	mux.HandleFunc("/api/access/deny/", s.authManager.AuthMiddleware(s.authManager.AdminMiddleware(s.accessRequestManager.HandleDenyAccessRequest)))

	// Static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		mux.Handle("/", http.FileServer(http.Dir("pkg/web/static")))
	} else {
		mux.Handle("/", http.FileServer(http.FS(staticFS)))
	}

	// Apply middleware chain: recovery -> request logging -> rate limiting -> body limit -> timeout -> security headers -> CORS -> CSRF -> handler
	handler := recoveryMiddleware(
		requestLoggingMiddleware(
			RateLimitMiddleware(s.apiRateLimiter, s.authRateLimiter)(
				maxBodyMiddleware(1 << 20)( // 1 MB max body size
					timeoutMiddleware(60 * time.Second)(
						securityHeadersMiddleware(
							corsMiddleware(
								s.authManager.CSRFMiddleware(mux),
							),
						),
					),
				),
			),
		),
	)

	s.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		// ReadTimeout and WriteTimeout are intentionally 0 (no limit) to support
		// WebSocket terminals and SSE streaming. Per-request timeouts are enforced
		// by timeoutMiddleware instead.
		IdleTimeout: 120 * time.Second,
	}

	fmt.Printf("\n  Web server started at http://localhost:%d\n", s.port)
	return s.server.ListenAndServe()
}

func (s *Server) Stop() error {
	// Stop metrics collector
	if s.metricsCollector != nil {
		s.metricsCollector.Stop()
	}

	// Stop notification manager
	if s.notifManager != nil {
		s.notifManager.Stop()
	}

	// Stop rate limiter cleanup goroutines
	if s.apiRateLimiter != nil {
		s.apiRateLimiter.Stop()
	}
	if s.authRateLimiter != nil {
		s.authRateLimiter.Stop()
	}

	// Stop brute-force protector cleanup goroutine
	if s.authManager != nil && s.authManager.bruteForce != nil {
		s.authManager.bruteForce.Stop()
	}

	// Stop CSRF/session cleanup goroutine
	if s.authManager != nil {
		s.authManager.StopCleanup()
	}

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

// maxBodyMiddleware limits request body size to prevent memory exhaustion
func maxBodyMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil && r.ContentLength != 0 {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Only allow same-origin requests or explicitly trusted origins
		allowedOrigins := map[string]bool{
			"":                       true, // Same origin (no Origin header)
			"http://localhost:8080":  true,
			"http://localhost:3000":  true,
			"http://127.0.0.1:8080":  true,
			"https://localhost:8080": true,
		}

		// Support configurable CORS origins via K13D_CORS_ALLOWED_ORIGINS env var
		if extraOrigins := os.Getenv("K13D_CORS_ALLOWED_ORIGINS"); extraOrigins != "" {
			for _, o := range strings.Split(extraOrigins, ",") {
				o = strings.TrimSpace(o)
				if o != "" {
					allowedOrigins[o] = true
				}
			}
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

		// HTTP Strict Transport Security (only for HTTPS)
		if r.TLS != nil {
			// max-age=31536000 (1 year), includeSubDomains
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Content Security Policy (all assets vendored locally for air-gapped support)
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"font-src 'self'; "+
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
	_ = json.NewEncoder(w).Encode(status)
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
	_ = json.NewEncoder(w).Encode(info)
}

// parseIntSafe parses a string to int, returning defaultVal on error.
func parseIntSafe(s string, defaultVal int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
