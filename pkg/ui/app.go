package ui

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/i18n"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Command definitions for autocomplete (k9s-style comprehensive list)
var commands = []struct {
	name     string
	alias    string
	desc     string
	category string
}{
	// Core Resources
	{"pods", "po", "List pods", "resource"},
	{"deployments", "deploy", "List deployments", "resource"},
	{"services", "svc", "List services", "resource"},
	{"nodes", "no", "List nodes", "resource"},
	{"namespaces", "ns", "List namespaces", "resource"},
	{"events", "ev", "List events", "resource"},

	// Config & Storage
	{"configmaps", "cm", "List configmaps", "resource"},
	{"secrets", "sec", "List secrets", "resource"},
	{"persistentvolumes", "pv", "List persistent volumes", "resource"},
	{"persistentvolumeclaims", "pvc", "List persistent volume claims", "resource"},
	{"storageclasses", "sc", "List storage classes", "resource"},

	// Workloads
	{"replicasets", "rs", "List replicasets", "resource"},
	{"daemonsets", "ds", "List daemonsets", "resource"},
	{"statefulsets", "sts", "List statefulsets", "resource"},
	{"jobs", "job", "List jobs", "resource"},
	{"cronjobs", "cj", "List cronjobs", "resource"},
	{"replicationcontrollers", "rc", "List replication controllers", "resource"},

	// Networking
	{"ingresses", "ing", "List ingresses", "resource"},
	{"endpoints", "ep", "List endpoints", "resource"},
	{"networkpolicies", "netpol", "List network policies", "resource"},
	{"ingressclasses", "ic", "List ingress classes", "resource"},

	// RBAC
	{"serviceaccounts", "sa", "List service accounts", "resource"},
	{"roles", "role", "List roles", "resource"},
	{"rolebindings", "rb", "List role bindings", "resource"},
	{"clusterroles", "cr", "List cluster roles", "resource"},
	{"clusterrolebindings", "crb", "List cluster role bindings", "resource"},

	// Policy
	{"poddisruptionbudgets", "pdb", "List pod disruption budgets", "resource"},
	{"podsecuritypolicies", "psp", "List pod security policies", "resource"},
	{"limitranges", "limits", "List limit ranges", "resource"},
	{"resourcequotas", "quota", "List resource quotas", "resource"},
	{"horizontalpodautoscalers", "hpa", "List horizontal pod autoscalers", "resource"},

	// CRDs
	{"customresourcedefinitions", "crd", "List custom resource definitions", "resource"},

	// Other
	{"leases", "lease", "List leases", "resource"},
	{"priorityclasses", "pc", "List priority classes", "resource"},
	{"runtimeclasses", "rtc", "List runtime classes", "resource"},
	{"volumeattachments", "va", "List volume attachments", "resource"},
	{"csidrivers", "csidriver", "List CSI drivers", "resource"},
	{"csinodes", "csinode", "List CSI nodes", "resource"},

	// Actions
	{"quit", "q", "Exit application", "action"},
	{"health", "status", "Show cluster health", "action"},
	{"context", "ctx", "Switch context", "action"},
	{"help", "?", "Show help", "action"},
	{"model", "models", "Switch AI model", "action"},
	{"alias", "aliases", "Show command aliases", "action"},
	{"plugins", "plugin", "Show plugins", "action"},
	{"pulse", "pu", "Cluster health pulse", "action"},
	{"xray", "xr", "XRay resource hierarchy", "action"},
	{"applications", "app", "Application-centric view", "action"},
}

// App is the main TUI application with k9s-style stability patterns
type App struct {
	*tview.Application

	// Core
	config   *config.Config
	k8s      *k8s.Client
	aiClient *ai.Client

	// UI components
	pages       *tview.Pages
	header      *tview.TextView
	briefing    *BriefingPanel // Natural language cluster briefing
	table       *tview.Table
	statusBar   *tview.TextView
	flash       *tview.TextView
	cmdInput    *tview.InputField
	cmdHint     *tview.TextView // Autocomplete hint (dimmed)
	cmdDropdown *tview.List     // Autocomplete dropdown
	aiPanel     *tview.TextView
	aiInput     *tview.InputField // AI question input

	// State (protected by mutex)
	mx                  sync.RWMutex
	currentResource     string
	currentNamespace    string
	namespaces          []string
	recentNamespaces    []string // Recently used namespaces (most recent first)
	maxRecentNamespaces int      // Max number of recent namespaces to track
	showAIPanel         bool
	filterText          string            // Current filter text
	filterRegex         bool              // True if filter is regex (e.g., /pattern/)
	tableHeaders        []string          // Original headers
	tableRows           [][]string        // Original rows (unfiltered)
	apiResources        []k8s.APIResource // Cached API resources from cluster
	selectedRows        map[int]bool      // Multi-select: selected row indices (k9s Space key)
	sortColumn          int               // Current sort column index (-1 = none)
	sortAscending       bool              // Sort direction (true = ascending, false = descending)

	// Command history
	cmdHistory    []string
	cmdHistoryIdx int // -1 = not browsing history

	// Port-forward tracking (protected by pfMx)
	pfMx         sync.Mutex
	portForwards []*portForwardInfo

	// Navigation history (protected by navMx)
	navMx           sync.Mutex
	navigationStack []navHistory

	// Atomic guards (k9s pattern for lock-free update deduplication)
	inUpdate    int32
	running     int32 // 1 after Application.Run() starts
	stopping    int32 // 1 when Stop() is called (set immediately, before tview processes)
	hasToolCall int32 // 1 if there's a pending tool call (atomic for lock-free check)
	needsSync   int32 // 1 when a full terminal sync is needed (namespace/resource switch)
	lastAIDraw  int64 // Unix nanos of last AI streaming draw (throttle rapid updates)
	lastSync    int64 // Unix nanos of last screen.Sync() (for periodic safety sync)
	flashSeq    int64 // Monotonic sequence for flash messages (prevents stale clears)

	// Loading state for UX
	loadingCount int32  // Number of active background tasks
	spinnerIdx   uint32 // Current spinner animation frame

	// Context management (k9s pattern for graceful shutdown)
	appCtx     context.Context    // Root application context
	appCancel  context.CancelFunc // Cancels all operations on Stop()
	cancelFn   context.CancelFunc // Refresh-specific cancellation
	cancelLock sync.Mutex         // Protects cancelFn updates

	// AI tool approval state (protected by aiMx)
	aiMx                sync.RWMutex
	pendingDecisions    []PendingDecision
	pendingToolApproval chan bool
	currentToolCallInfo struct {
		Name    string
		Args    string
		Command string
	}

	// Watch state (protected by watchMu)
	watcher     *k8s.ResourceWatcher // Active resource watcher (nil when inactive)
	watchCancel context.CancelFunc   // Cancel function for watcher context
	watchMu     sync.Mutex           // Protects watcher lifecycle operations

	// Logger
	logger *slog.Logger

	// Test mode flags
	skipBriefing bool // Skip briefing panel in test mode to prevent pulse animation blocking

	// Extensibility configs (k9s pattern)
	customAliases *config.AliasConfig // User-defined resource aliases
	viewsConfig   *config.ViewConfig  // Per-resource view settings (sort defaults)
	plugins       *config.PluginsFile // Plugin definitions
	styles        *config.StyleConfig // Per-context skin/theme

	// RBAC authorization (Teleport-inspired)
	tuiRole      string // TUI user role (default: "admin" for backward compatibility)
	authorizer   TUIAuthorizer
	warnedNoRBAC bool // True after first "no RBAC" warning has been shown
}

// TUIAuthorizer interface for RBAC authorization in TUI
type TUIAuthorizer interface {
	IsAllowed(role, resource string, action string, namespace string) (bool, string)
}

// PendingDecision represents a command awaiting user approval
type PendingDecision struct {
	Command     string
	Description string
	IsDangerous bool
	Warnings    []string
	ToolName    string
	ToolArgs    string
	IsToolCall  bool
}

// safeGo wraps goroutines with panic recovery
func (a *App) safeGo(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.logger.Error("goroutine panic recovered", "name", name, "error", r, "stack", string(debug.Stack()))
				a.flashMsg(fmt.Sprintf("Internal error in %s. Recovered.", name), true)
			}
		}()
		fn()
	}()
}

// getAppContext returns the root app context
func (a *App) getAppContext() context.Context {
	if a.appCtx != nil {
		return a.appCtx
	}
	return context.Background()
}

// NewApp creates a new TUI application
func NewApp() *App {
	return NewAppWithNamespace("")
}

// NewAppWithNamespace creates a new TUI application with initial namespace
func NewAppWithNamespace(initialNamespace string) *App {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Warn("Failed to load config, using defaults", "error", err)
		cfg = config.NewDefaultConfig()
	}
	i18n.SetLanguage(cfg.Language)

	k8sClient, err := k8s.NewClient()
	if err != nil {
		logger.Warn("K8s client initialization failed", "error", err)
	}

	aiClient, aiErr := ai.NewClient(&cfg.LLM)
	if aiErr != nil {
		logger.Warn("AI client initialization failed", "error", aiErr)
	}

	if initialNamespace == "all" {
		initialNamespace = ""
	}

	appCtx, appCancel := context.WithCancel(context.Background())

	app := &App{
		Application:         tview.NewApplication(),
		config:              cfg,
		k8s:                 k8sClient,
		aiClient:            aiClient,
		currentResource:     "pods",
		currentNamespace:    initialNamespace,
		namespaces:          []string{""},
		recentNamespaces:    make([]string, 0),
		maxRecentNamespaces: 9,
		showAIPanel:         true,
		selectedRows:        make(map[int]bool),
		sortColumn:          -1,
		sortAscending:       true,
		cmdHistoryIdx:       -1,
		pendingToolApproval: make(chan bool, 1),
		logger:              logger,
		appCtx:              appCtx,
		appCancel:           appCancel,
	}

	if aliases, err := config.LoadAliases(); err == nil {
		app.customAliases = aliases
	}
	if views, err := config.LoadViews(); err == nil {
		app.viewsConfig = views
	}
	if plugins, err := config.LoadPlugins(); err == nil {
		app.plugins = plugins
	}

	if k8sClient != nil {
		if ctxName, err := k8sClient.GetCurrentContext(); err == nil && ctxName != "" {
			if styles, err := config.LoadStylesForContext(ctxName); err == nil {
				app.styles = styles
			}
		}
	}
	if app.styles == nil {
		app.styles = config.DefaultStyles()
	}

	app.setupUI()
	app.setupKeybindings()

	app.safeGo("loadAPIResources", app.loadAPIResources)
	app.safeGo("loadNamespaces", app.loadNamespaces)

	return app
}

// Run starts the application with a global panic handler
func (a *App) Run() error {
	// Global robust panic handler to ensure terminal reset
	defer func() {
		if r := recover(); r != nil {
			// Ensure tview stops to reset terminal
			if a.Application != nil {
				a.Application.Stop()
			}

			// Log panic details
			stack := string(debug.Stack())
			fmt.Fprintf(os.Stderr, "\n[FATAL ERROR] k13d encountered a panic and had to exit.\n")
			fmt.Fprintf(os.Stderr, "Error: %v\n", r)
			fmt.Fprintf(os.Stderr, "\nPlease report this issue with the following stack trace:\n%s\n", stack)

			if a.logger != nil {
				a.logger.Error("GLOBAL TUI PANIC", "error", r, "stack", stack)
			}

			os.Exit(1)
		}
	}()

	defer func() {
		atomic.StoreInt32(&a.stopping, 1)
		if a.briefing != nil {
			a.briefing.stopPulseAnimation()
		}
		atomic.StoreInt32(&a.running, 0)
	}()

	a.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Clear()
		now := time.Now().UnixNano()
		if atomic.CompareAndSwapInt32(&a.needsSync, 1, 0) {
			screen.Sync()
			atomic.StoreInt64(&a.lastSync, now)
		} else if now-atomic.LoadInt64(&a.lastSync) > 2_000_000_000 {
			screen.Sync()
			atomic.StoreInt64(&a.lastSync, now)
		}
		return false
	})

	go func() {
		ticker := time.NewTicker(150 * time.Millisecond)
		defer ticker.Stop()
		for {
			if atomic.LoadInt32(&a.stopping) == 1 {
				return
			}
			<-ticker.C
			if atomic.LoadInt32(&a.loadingCount) > 0 {
				atomic.AddUint32(&a.spinnerIdx, 1)
				a.QueueUpdateDraw(func() { a.updateStatusBar() })
			}
		}
	}()

	a.SetAfterDrawFunc(func(screen tcell.Screen) {
		a.SetAfterDrawFunc(nil)
		atomic.StoreInt32(&a.running, 1)
		time.AfterFunc(50*time.Millisecond, func() {
			a.refresh()
			a.startWatch()
		})
		if a.briefing != nil && a.briefing.IsVisible() {
			a.briefing.startPulse()
		}
	})

	a.logger.Info("Starting k13d TUI")
	return a.Application.Run()
}

// Stop stops the application gracefully
func (a *App) Stop() {
	atomic.StoreInt32(&a.stopping, 1)
	if a.appCancel != nil {
		a.appCancel()
	}
	a.cancelLock.Lock()
	if a.cancelFn != nil {
		a.cancelFn()
	}
	a.cancelLock.Unlock()
	a.stopWatch()
	if a.briefing != nil {
		a.briefing.stopPulseAnimation()
	}
	a.cleanupPortForwards()
	if a.Application != nil {
		a.Application.Stop()
	}
}

// IsRunning returns true if the application is active
func (a *App) IsRunning() bool {
	return atomic.LoadInt32(&a.running) == 1 && atomic.LoadInt32(&a.stopping) == 0
}
