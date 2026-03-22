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
	Context   string `json:"context,omitempty"`    // Selected resource context for the AI prompt only
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

	return newServer(cfg, port, authConfig, versionInfo)
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

	return newServer(cfg, port, authConfig, versionInfo)
}

// newServer contains the shared initialization logic for both constructors.
func newServer(cfg *config.Config, port int, authConfig *AuthConfig, versionInfo *VersionInfo) (*Server, error) {
	var aiClient *ai.Client
	var err error
	runtimeInfo := config.GetRuntimeSourceInfo()

	fmt.Printf("Starting k13d web server...\n")
	fmt.Printf("  Config File: %s (%s)\n", runtimeInfo.ConfigPath, describeConfigFileStatus(runtimeInfo))
	fmt.Printf("  Config Path Source: %s\n", runtimeInfo.ConfigPathSource)
	if runtimeInfo.XDGConfigHome != "" {
		fmt.Printf("  XDG_CONFIG_HOME: %s\n", runtimeInfo.XDGConfigHome)
	}
	if len(runtimeInfo.EnvOverrides) > 0 {
		fmt.Printf("  Env Overrides: %s\n", strings.Join(runtimeInfo.EnvOverrides, ", "))
	} else {
		fmt.Printf("  Env Overrides: none\n")
	}
	fmt.Printf("  LLM Settings: %s\n", describeLLMSource(runtimeInfo))
	fmt.Printf("  LLM Provider: %s, Model: %s\n", cfg.LLM.Provider, cfg.LLM.Model)
	fmt.Printf("  Login UI: %s\n", describeLoginUI(authConfig))

	aiClient, ready, err := createUsableAIClient(&cfg.LLM)
	if err != nil {
		fmt.Printf("  AI client creation failed: %v\n", err)
		aiClient = nil
	} else if ready {
		fmt.Printf("  AI client: Ready\n")
	} else {
		aiClient = nil
		fmt.Printf("  AI client: Not configured (missing endpoint or credentials)\n")
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

	// Set MCP reconnect callback to re-register tools when connection is restored
	server.mcpClient.OnReconnect = func(serverName string) {
		server.registerMCPTools(serverName)
	}

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

func describeConfigFileStatus(info config.RuntimeSourceInfo) string {
	if info.ConfigFileExists {
		return "found"
	}
	return "missing -> defaults/env only"
}

func describeLLMSource(info config.RuntimeSourceInfo) string {
	if len(info.LLMEnvOverrides) == 0 {
		if info.ConfigFileExists {
			return "config file only"
		}
		return "defaults only"
	}
	if info.ConfigFileExists {
		return "config file + env overrides"
	}
	return "defaults + env overrides"
}

func describeLoginUI(authConfig *AuthConfig) string {
	if authConfig == nil || !authConfig.Enabled {
		return "no login screen (authentication disabled)"
	}

	switch authConfig.AuthMode {
	case "local":
		return "username/password form (token form hidden)"
	case "token", "":
		return "Kubernetes token form"
	case "ldap":
		return "username/password form (LDAP backend)"
	case "oidc":
		return "OIDC / SSO flow"
	default:
		return fmt.Sprintf("auth mode %q", authConfig.AuthMode)
	}
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

// withRecovery wraps a HandlerFunc with panic recovery.
// Delegates to recoveryMiddleware to avoid duplicating recovery logic.
func withRecovery(handler http.HandlerFunc) http.HandlerFunc {
	return recoveryMiddleware(handler).ServeHTTP
}

// requestLoggingMiddleware logs all HTTP requests
func requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(rw, r)

		// Log request (exclude health checks and successful GETs to reduce noise)
		if r.URL.Path != "/api/health" && (r.Method != http.MethodGet || rw.statusCode >= 400) {
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
				if !headerSent && ctx.Err() == context.DeadlineExceeded {
					dw.ResponseWriter.Header().Set("Content-Type", "application/json")
					dw.ResponseWriter.WriteHeader(http.StatusGatewayTimeout)
					_ = json.NewEncoder(dw.ResponseWriter).Encode(NewAPIError(ErrCodeTimeout, "Request timed out"))
				}
				dw.mu.Unlock()
			}
		})
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register routes by domain (see individual methods for details)
	s.registerPublicRoutes(mux)
	s.registerAuthRoutes(mux)
	s.registerAIRoutes(mux)
	s.registerK8sRoutes(mux)
	s.registerWorkloadOperationRoutes(mux)
	s.registerHelmRoutes(mux)
	s.registerMetricsRoutes(mux)
	s.registerSecurityRoutes(mux)
	s.registerVisualizationRoutes(mux)
	s.registerAdminRoutes(mux)

	// Static files - serve index.html with auth mode injected
	staticFS, err := fs.Sub(staticFiles, "static")
	var staticHandler http.Handler
	if err != nil {
		staticHandler = http.FileServer(http.Dir("pkg/web/static"))
	} else {
		staticHandler = http.FileServer(http.FS(staticFS))
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// For index.html (root path), inject auth mode as inline script
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			var indexData []byte
			if err != nil {
				indexData, _ = os.ReadFile("pkg/web/static/index.html")
			} else {
				indexData, _ = fs.ReadFile(staticFiles, "static/index.html")
			}
			if indexData != nil {
				authMode := s.authManager.GetAuthMode()
				html := string(indexData)
				// Inject auth mode as JS variable
				injection := fmt.Sprintf(`<script>window.__AUTH_MODE__=%q;</script>`, authMode)
				html = strings.Replace(html, "</head>", injection+"</head>", 1)
				// Directly show the correct login form via inline style (no JS dependency)
				if authMode == "local" {
					html = strings.Replace(html, `id="password-login-form" class="login-form"`, `id="password-login-form" class="login-form" style="display:block"`, 1)
				} else {
					html = strings.Replace(html, `id="token-login-form" class="login-form"`, `id="token-login-form" class="login-form" style="display:block"`, 1)
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				_, _ = w.Write([]byte(html))
				return
			}
		}
		staticHandler.ServeHTTP(w, r)
	})

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

	// Stop all active port forward sessions
	s.pfMutex.Lock()
	for id, sess := range s.portForwardSessions {
		sess.closeOnce.Do(func() {
			close(sess.stopChan)
		})
		delete(s.portForwardSessions, id)
	}
	s.pfMutex.Unlock()

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
