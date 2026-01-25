package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gdamore/tcell/v2"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/i18n"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
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

	// Navigation history (protected by navMx)
	navMx           sync.Mutex
	navigationStack []navHistory

	// Atomic guards (k9s pattern for lock-free update deduplication)
	inUpdate    int32
	running     int32 // 1 after Application.Run() starts
	stopping    int32 // 1 when Stop() is called (set immediately, before tview processes)
	hasToolCall int32 // 1 if there's a pending tool call (atomic for lock-free check)
	cancelFn    context.CancelFunc
	cancelLock  sync.Mutex

	// AI tool approval state (protected by aiMx)
	aiMx                sync.RWMutex
	pendingDecisions    []PendingDecision
	pendingToolApproval chan bool
	currentToolCallInfo struct {
		Name    string
		Args    string
		Command string
	}

	// Logger
	logger *slog.Logger
}

// NewApp creates a new TUI application with default (all) namespace
func NewApp() *App {
	return NewAppWithNamespace("")
}

// NewAppWithNamespace creates a new TUI application with initial namespace
// Pass "" for all namespaces, or a specific namespace name
func NewAppWithNamespace(initialNamespace string) *App {
	// Setup structured logging (k9s pattern)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Warn("Failed to load config, using defaults", "error", err)
		cfg = config.NewDefaultConfig()
	}
	i18n.SetLanguage(cfg.Language)

	// K8s client with graceful degradation (k9s pattern)
	k8sClient, err := k8s.NewClient()
	if err != nil {
		logger.Warn("K8s client initialization failed", "error", err)
	} else {
		logger.Info("K8s connectivity OK")
	}

	// AI client (optional)
	aiClient, _ := ai.NewClient(&cfg.LLM)

	// Handle "all" as empty string (all namespaces)
	if initialNamespace == "all" {
		initialNamespace = ""
	}

	app := &App{
		Application:         tview.NewApplication(),
		config:              cfg,
		k8s:                 k8sClient,
		aiClient:            aiClient,
		currentResource:     "pods",
		currentNamespace:    initialNamespace,
		namespaces:          []string{""},
		recentNamespaces:    make([]string, 0),
		maxRecentNamespaces: 9, // Track up to 9 recent namespaces (for 1-9 keys)
		showAIPanel:         true,
		selectedRows:        make(map[int]bool),
		sortColumn:          -1, // No sort initially
		sortAscending:       true,
		pendingToolApproval: make(chan bool, 1),
		logger:              logger,
	}

	app.setupUI()
	app.setupKeybindings()

	// Load API resources in background (for autocomplete)
	go app.loadAPIResources()

	// Load namespaces in background (for header preview)
	go app.loadNamespaces()

	return app
}

// loadAPIResources fetches available API resources from the cluster
func (a *App) loadAPIResources() {
	if a.k8s == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resources, err := a.k8s.GetAPIResources(ctx)
	if err != nil {
		a.logger.Warn("Failed to load API resources", "error", err)
		// Use common resources as fallback
		resources = a.k8s.GetCommonResources()
	}

	a.mx.Lock()
	a.apiResources = resources
	a.mx.Unlock()

	a.logger.Info("Loaded API resources", "count", len(resources))
}

// loadNamespaces fetches namespaces for header preview
func (a *App) loadNamespaces() {
	if a.k8s == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nss, err := a.k8s.ListNamespaces(ctx)
	if err != nil {
		a.logger.Warn("Failed to load namespaces", "error", err)
		return
	}

	// Build namespace list
	namespaceList := make([]string, 0, len(nss)+1)
	namespaceList = append(namespaceList, "") // Empty string for "all namespaces"
	for _, n := range nss {
		namespaceList = append(namespaceList, n.Name)
	}

	a.mx.Lock()
	a.namespaces = namespaceList
	a.mx.Unlock()

	// Reorder by recent usage
	reordered := a.reorderNamespacesByRecent()
	a.mx.Lock()
	a.namespaces = reordered
	a.mx.Unlock()

	// Update header to show namespaces
	a.updateHeader()

	a.logger.Info("Loaded namespaces", "count", len(nss))
}

// setupUI initializes all UI components
func (a *App) setupUI() {
	// Header
	a.header = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	a.header.SetBackgroundColor(tcell.ColorDarkBlue)

	// Main table with fixed header row
	a.table = tview.NewTable().
		SetSelectable(true, false).
		SetFixed(1, 0)
	a.table.SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)
	a.table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkCyan).
		Foreground(tcell.ColorWhite))

	// AI Panel (output area)
	a.aiPanel = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	a.aiPanel.SetText("[gray]Press [yellow]Tab[gray] to ask AI\n\n" +
		"[white]Examples:\n" +
		"[darkgray]- Why is this pod failing?\n" +
		"- How do I scale this deployment?\n" +
		"- Explain this resource")

	// AI Input field
	a.aiInput = tview.NewInputField().
		SetLabel(" 🤖 ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetPlaceholder("Ask AI a question...")
	a.aiInput.SetPlaceholderStyle(tcell.StyleDefault.Foreground(tcell.ColorDarkGray))
	a.setupAIInput()

	// Flash message area (k9s pattern)
	a.flash = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	// Briefing panel (natural language cluster summary)
	a.briefing = NewBriefingPanel(a)

	// Status bar
	a.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	a.statusBar.SetBackgroundColor(tcell.ColorDarkGreen)

	// Command input with autocomplete
	a.cmdInput = tview.NewInputField().
		SetLabel(" : ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorDefault)

	// Autocomplete hint (dimmed text showing suggestion)
	a.cmdHint = tview.NewTextView().
		SetDynamicColors(true)

	// Autocomplete dropdown
	a.cmdDropdown = tview.NewList().
		ShowSecondaryText(true).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorDarkCyan).
		SetSelectedTextColor(tcell.ColorWhite)
	a.cmdDropdown.SetBorder(true).SetTitle(" Commands ")

	// Setup autocomplete behavior
	a.setupAutocomplete()

	// AI Panel container (output + input)
	aiContainer := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.aiPanel, 0, 1, false).
		AddItem(a.aiInput, 1, 0, true)
	aiContainer.SetBorder(true).
		SetTitle(" AI Assistant ").
		SetBorderColor(tcell.ColorDarkMagenta)

	// Content area (table + AI panel)
	contentFlex := tview.NewFlex()
	contentFlex.AddItem(a.table, 0, 3, true)
	if a.showAIPanel {
		contentFlex.AddItem(aiContainer, 45, 0, false)
	}

	// Command bar with hint overlay
	cmdFlex := tview.NewFlex().
		AddItem(a.cmdInput, 0, 1, true).
		AddItem(a.cmdHint, 0, 2, false)

	// Main layout with briefing panel
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.header, 4, 0, false). // 4 lines: title, context info, namespace preview
		AddItem(a.flash, 1, 0, false).
		AddItem(a.briefing, 5, 0, false). // 3 lines content + 2 border
		AddItem(contentFlex, 0, 1, true).
		AddItem(a.statusBar, 1, 0, false).
		AddItem(cmdFlex, 1, 0, false)

	// Pages
	a.pages = tview.NewPages().
		AddPage("main", mainFlex, true, true)

	a.SetRoot(a.pages, true)

	// Initial UI state
	a.updateHeader()
	a.updateStatusBar()
}

// setupAIInput configures the AI input field
func (a *App) setupAIInput() {
	a.aiInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			question := a.aiInput.GetText()
			if question != "" {
				a.aiInput.SetText("")
				go a.askAI(question)
			}
		}
	})

	a.aiInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc, tcell.KeyTab:
			a.SetFocus(a.table)
			return nil
		}
		return event
	})
}

// PendingDecision represents a command awaiting user approval
type PendingDecision struct {
	Command     string
	Description string
	IsDangerous bool
	Warnings    []string
	// For MCP tool execution
	ToolName   string
	ToolArgs   string
	IsToolCall bool
}

// Note: pendingDecisions, pendingToolApproval, and currentToolCallInfo
// are now fields of App struct (protected by aiMx mutex)

// askAI sends a question to the AI and displays the response
func (a *App) askAI(question string) {
	// Show loading state
	a.QueueUpdateDraw(func() {
		a.aiPanel.SetText(fmt.Sprintf("[yellow]Question:[white] %s\n\n[gray]Thinking...", question))
	})

	// Get current context
	a.mx.RLock()
	resource := a.currentResource
	ns := a.currentNamespace
	a.mx.RUnlock()

	// Get selected resource info if available
	var selectedInfo string
	row, _ := a.table.GetSelection()
	if row > 0 {
		var parts []string
		for c := 0; c < a.table.GetColumnCount(); c++ {
			cell := a.table.GetCell(row, c)
			if cell != nil {
				parts = append(parts, cell.Text)
			}
		}
		selectedInfo = strings.Join(parts, " | ")
	}

	// Build context for AI
	ctx := context.Background()
	prompt := fmt.Sprintf(`User is viewing Kubernetes %s`, resource)
	if ns != "" {
		prompt += fmt.Sprintf(` in namespace "%s"`, ns)
	}
	if selectedInfo != "" {
		prompt += fmt.Sprintf(`. Selected: %s`, selectedInfo)
	}
	prompt += fmt.Sprintf(`

User question: %s

Please provide a concise, helpful answer. If you suggest kubectl commands, wrap them in code blocks.`, question)

	// Call AI
	if a.aiClient == nil || !a.aiClient.IsReady() {
		a.QueueUpdateDraw(func() {
			a.aiPanel.SetText(fmt.Sprintf("[yellow]Q:[white] %s\n\n[red]AI is not available.[white]\n\nConfigure LLM in config file:\n[gray]~/.kube-ai-dashboard/config.yaml", question))
		})
		return
	}

	// Check if AI supports tool calling (agentic mode)
	var fullResponse strings.Builder
	var err error

	if a.aiClient.SupportsTools() {
		// Use agentic mode with tool calling
		a.QueueUpdateDraw(func() {
			a.aiPanel.SetText(fmt.Sprintf("[yellow]Q:[white] %s\n\n[cyan]🤖 Agentic Mode[white] - AI can execute kubectl commands\n\n[gray]Thinking...", question))
		})

		err = a.aiClient.AskWithTools(ctx, prompt, func(chunk string) {
			fullResponse.WriteString(chunk)
			response := fullResponse.String()
			a.QueueUpdateDraw(func() {
				a.aiPanel.SetText(fmt.Sprintf("[yellow]Q:[white] %s\n\n[cyan]🤖 Agentic Mode[white]\n\n[green]A:[white] %s", question, response))
			})
		}, func(toolName string, args string) bool {
			// Tool approval callback - kubectl-ai style Decision Required
			a.logger.Info("Tool callback invoked", "tool", toolName, "args", args)

			filter := ai.NewCommandFilter()

			// Parse command from args
			var cmdArgs struct {
				Command   string `json:"command"`
				Namespace string `json:"namespace,omitempty"`
			}
			if err := parseJSON(args, &cmdArgs); err != nil {
				a.logger.Error("Failed to parse tool args", "error", err, "args", args)
			}

			fullCmd := ""
			if toolName == "kubectl" {
				fullCmd = "kubectl " + cmdArgs.Command
				if cmdArgs.Namespace != "" && !strings.Contains(cmdArgs.Command, "-n ") {
					fullCmd = "kubectl -n " + cmdArgs.Namespace + " " + cmdArgs.Command
				}
			} else if toolName == "bash" {
				fullCmd = cmdArgs.Command
			}

			a.logger.Info("Analyzed command", "fullCmd", fullCmd)

			// Analyze command safety
			report := filter.AnalyzeCommand(fullCmd)

			// Read-only commands: auto-approve
			if report.Type == ai.CommandTypeReadOnly {
				return true
			}

			// Store current tool info for approval (deadlock-safe)
			a.setToolCallState(toolName, args, fullCmd)

			// Show Decision Required UI
			a.QueueUpdateDraw(func() {
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("[yellow]Q:[white] %s\n\n", question))
				sb.WriteString(fullResponse.String())
				sb.WriteString("\n\n[yellow::b]━━━ DECISION REQUIRED ━━━[white::-]\n\n")

				if report.IsDangerous {
					sb.WriteString("[red]⚠ DANGEROUS COMMAND[white]\n")
				} else if report.Type == ai.CommandTypeWrite {
					sb.WriteString("[yellow]? WRITE OPERATION[white]\n")
				} else {
					sb.WriteString("[gray]? COMMAND APPROVAL[white]\n")
				}

				sb.WriteString(fmt.Sprintf("\n[cyan]%s[white]\n\n", fullCmd))

				for _, w := range report.Warnings {
					sb.WriteString(fmt.Sprintf("[red]• %s[white]\n", w))
				}

				sb.WriteString("\n[gray]Press [green]Y[gray] or [green]Enter[gray] to approve, [red]N[gray] or [red]Esc[gray] to cancel[white]")
				a.aiPanel.SetText(sb.String())

				// Focus AI panel for key input
				a.SetFocus(a.aiPanel)
			})

			// Wait for user decision (blocking)
			// Clear any pending approvals first
			select {
			case <-a.pendingToolApproval:
			default:
			}

			// Wait for approval with timeout
			select {
			case approved := <-a.pendingToolApproval:
				if approved {
					a.QueueUpdateDraw(func() {
						currentText := a.aiPanel.GetText(false)
						a.aiPanel.SetText(currentText + "\n\n[green]✓ Approved - Executing...[white]")
					})
				} else {
					a.QueueUpdateDraw(func() {
						currentText := a.aiPanel.GetText(false)
						a.aiPanel.SetText(currentText + "\n\n[red]✗ Cancelled by user[white]")
					})
				}
				return approved
			case <-ctx.Done():
				return false
			}
		})
	} else {
		// Fallback to regular streaming
		err = a.aiClient.Ask(ctx, prompt, func(chunk string) {
			fullResponse.WriteString(chunk)
			response := fullResponse.String()
			a.QueueUpdateDraw(func() {
				a.aiPanel.SetText(fmt.Sprintf("[yellow]Q:[white] %s\n\n[green]A:[white] %s", question, response))
			})
		})
	}

	if err != nil {
		a.QueueUpdateDraw(func() {
			a.aiPanel.SetText(fmt.Sprintf("[yellow]Q:[white] %s\n\n[red]Error:[white] %v", question, err))
		})
		return
	}

	// After response complete, analyze for commands that need approval (fallback mode)
	if !a.aiClient.SupportsTools() {
		finalResponse := fullResponse.String()
		a.analyzeAndShowDecisions(question, finalResponse)
	}
}

// parseJSON is a helper to parse JSON arguments
func parseJSON(jsonStr string, v interface{}) error {
	return jsonUnmarshal([]byte(jsonStr), v)
}

// jsonUnmarshal wraps json.Unmarshal
var jsonUnmarshal = func(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// analyzeAndShowDecisions extracts commands from AI response and shows decision UI
func (a *App) analyzeAndShowDecisions(question, response string) {
	// Extract kubectl commands from response
	commands := ai.ExtractKubectlCommands(response)
	if len(commands) == 0 {
		return
	}

	// Analyze commands for safety
	filter := ai.NewCommandFilter()

	a.aiMx.Lock()
	a.pendingDecisions = nil

	var hasDecisions bool
	for _, cmd := range commands {
		report := filter.AnalyzeCommand(cmd)
		if report.RequiresConfirmation || report.IsDangerous {
			hasDecisions = true
			a.pendingDecisions = append(a.pendingDecisions, PendingDecision{
				Command:     cmd,
				Description: getCommandDescription(cmd),
				IsDangerous: report.IsDangerous,
				Warnings:    report.Warnings,
			})
		}
	}

	if !hasDecisions {
		a.aiMx.Unlock()
		return
	}

	// Copy for UI update
	decisions := make([]PendingDecision, len(a.pendingDecisions))
	copy(decisions, a.pendingDecisions)
	a.aiMx.Unlock()

	// Update AI panel with decision prompt
	a.QueueUpdateDraw(func() {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("[yellow]Q:[white] %s\n\n", question))
		sb.WriteString(fmt.Sprintf("[green]A:[white] %s\n\n", response))
		sb.WriteString("[yellow::b]━━━ DECISION REQUIRED ━━━[white::-]\n\n")

		for i, decision := range decisions {
			if decision.IsDangerous {
				sb.WriteString(fmt.Sprintf("[red]⚠ [%d] DANGEROUS:[white] ", i+1))
			} else {
				sb.WriteString(fmt.Sprintf("[yellow]? [%d] Confirm:[white] ", i+1))
			}
			sb.WriteString(fmt.Sprintf("[cyan]%s[white]\n", decision.Command))

			for _, warning := range decision.Warnings {
				sb.WriteString(fmt.Sprintf("   [red]• %s[white]\n", warning))
			}
			sb.WriteString("\n")
		}

		sb.WriteString("[gray]Press [yellow]1-9[gray] to execute, [yellow]A[gray] to execute all, [yellow]Esc[gray] to cancel[white]")
		a.aiPanel.SetText(sb.String())
	})
}

// getCommandDescription returns a brief description of the command
func getCommandDescription(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) < 2 {
		return cmd
	}

	// Skip "kubectl" if present
	if parts[0] == "kubectl" {
		parts = parts[1:]
	}

	if len(parts) == 0 {
		return cmd
	}

	switch parts[0] {
	case "delete":
		return "Delete resource"
	case "apply":
		return "Apply configuration"
	case "create":
		return "Create resource"
	case "scale":
		return "Scale resource"
	case "rollout":
		return "Rollout operation"
	case "patch":
		return "Patch resource"
	case "edit":
		return "Edit resource"
	case "drain":
		return "Drain node"
	case "cordon":
		return "Cordon node"
	case "uncordon":
		return "Uncordon node"
	default:
		return parts[0]
	}
}

// executeDecision executes a specific pending decision by index
func (a *App) executeDecision(idx int) {
	a.aiMx.Lock()
	if idx < 0 || idx >= len(a.pendingDecisions) {
		a.aiMx.Unlock()
		return
	}

	decision := a.pendingDecisions[idx]
	// Remove executed decision
	a.pendingDecisions = append(a.pendingDecisions[:idx], a.pendingDecisions[idx+1:]...)
	a.aiMx.Unlock()

	a.flashMsg(fmt.Sprintf("Executing: %s", decision.Command), false)

	// Execute the command
	cmd := exec.Command("bash", "-c", decision.Command)
	output, err := cmd.CombinedOutput()

	// Update AI panel with result
	a.QueueUpdateDraw(func() {
		var result string
		if err != nil {
			result = fmt.Sprintf("[red]Error:[white] %v\n%s", err, string(output))
		} else {
			result = fmt.Sprintf("[green]Success:[white]\n%s", string(output))
		}

		// Show execution result
		currentText := a.aiPanel.GetText(false)
		a.aiPanel.SetText(currentText + "\n\n[yellow]━━━ EXECUTION RESULT ━━━[white]\n" +
			fmt.Sprintf("[cyan]%s[white]\n%s", decision.Command, result))
	})

	// Refresh if it was a modifying command
	go a.refresh()
}

// executeAllDecisions executes all pending decisions
func (a *App) executeAllDecisions() {
	a.aiMx.RLock()
	if len(a.pendingDecisions) == 0 {
		a.aiMx.RUnlock()
		return
	}

	// Show confirmation for dangerous commands
	hasDangerous := false
	for _, d := range a.pendingDecisions {
		if d.IsDangerous {
			hasDangerous = true
			break
		}
	}
	a.aiMx.RUnlock()

	if hasDangerous {
		modal := tview.NewModal().
			SetText("[red]WARNING:[white] Some commands are dangerous!\n\nAre you sure you want to execute ALL commands?").
			AddButtons([]string{"Cancel", "Execute All"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				a.pages.RemovePage("confirm-all")
				if buttonLabel == "Execute All" {
					go a.doExecuteAll()
				}
			})
		modal.SetBackgroundColor(tcell.ColorDarkRed)
		a.pages.AddPage("confirm-all", modal, true, true)
	} else {
		go a.doExecuteAll()
	}
}

// doExecuteAll actually executes all pending decisions
func (a *App) doExecuteAll() {
	a.aiMx.Lock()
	decisions := make([]PendingDecision, len(a.pendingDecisions))
	copy(decisions, a.pendingDecisions)
	a.pendingDecisions = nil
	a.aiMx.Unlock()

	var results strings.Builder
	results.WriteString("\n\n[yellow]━━━ BATCH EXECUTION RESULTS ━━━[white]\n")

	for _, decision := range decisions {
		a.flashMsg(fmt.Sprintf("Executing: %s", decision.Command), false)

		cmd := exec.Command("bash", "-c", decision.Command)
		output, err := cmd.CombinedOutput()

		results.WriteString(fmt.Sprintf("\n[cyan]%s[white]\n", decision.Command))
		if err != nil {
			results.WriteString(fmt.Sprintf("[red]Error:[white] %v\n%s\n", err, string(output)))
		} else {
			results.WriteString(fmt.Sprintf("[green]Success:[white] %s\n", strings.TrimSpace(string(output))))
		}
	}

	a.QueueUpdateDraw(func() {
		currentText := a.aiPanel.GetText(false)
		a.aiPanel.SetText(currentText + results.String())
	})

	a.flashMsg(fmt.Sprintf("Executed %d commands", len(decisions)), false)
	go a.refresh()
}

// setupAutocomplete configures the command input with autocomplete
func (a *App) setupAutocomplete() {
	// Track current suggestions
	var suggestions []string
	var selectedIdx int

	// Update hint as user types
	a.cmdInput.SetChangedFunc(func(text string) {
		suggestions = a.getCompletions(text)
		selectedIdx = 0

		if len(suggestions) > 0 && text != "" {
			// Show dimmed hint for first suggestion
			hint := suggestions[0]
			if strings.HasPrefix(hint, text) {
				remaining := hint[len(text):]
				a.cmdHint.SetText("[gray]" + remaining)
			} else {
				a.cmdHint.SetText("[gray] → " + hint)
			}
		} else {
			a.cmdHint.SetText("")
		}
	})

	// Handle special keys
	a.cmdInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		text := a.cmdInput.GetText()

		switch event.Key() {
		case tcell.KeyTab:
			// Accept current suggestion
			if len(suggestions) > 0 {
				selected := suggestions[selectedIdx]
				// If it's a namespace command, add space
				if selected == "ns" || strings.HasPrefix(selected, "ns ") {
					a.cmdInput.SetText(selected + " ")
				} else {
					a.cmdInput.SetText(selected)
				}
				a.cmdHint.SetText("")
			}
			return nil

		case tcell.KeyDown:
			// Cycle through suggestions
			if len(suggestions) > 1 {
				selectedIdx = (selectedIdx + 1) % len(suggestions)
				hint := suggestions[selectedIdx]
				if strings.HasPrefix(hint, text) {
					remaining := hint[len(text):]
					a.cmdHint.SetText("[gray]" + remaining)
				} else {
					a.cmdHint.SetText("[gray] → " + hint)
				}
			}
			return nil

		case tcell.KeyUp:
			// Cycle through suggestions backwards
			if len(suggestions) > 1 {
				selectedIdx--
				if selectedIdx < 0 {
					selectedIdx = len(suggestions) - 1
				}
				hint := suggestions[selectedIdx]
				if strings.HasPrefix(hint, text) {
					remaining := hint[len(text):]
					a.cmdHint.SetText("[gray]" + remaining)
				} else {
					a.cmdHint.SetText("[gray] → " + hint)
				}
			}
			return nil

		case tcell.KeyEnter:
			cmd := text
			// If hint is showing and user didn't type full command, use suggestion
			if len(suggestions) > 0 && cmd != suggestions[selectedIdx] {
				// Check if input matches number for namespace selection
				if num, ok := a.parseNamespaceNumber(cmd); ok {
					a.selectNamespaceByNumber(num)
					a.cmdInput.SetText("")
					a.cmdHint.SetText("")
					a.cmdInput.SetLabel(" : ")
					a.SetFocus(a.table)
					return nil
				}
			}
			a.cmdInput.SetText("")
			a.cmdHint.SetText("")
			a.cmdInput.SetLabel(" : ")
			a.handleCommand(cmd)
			a.SetFocus(a.table)
			return nil

		case tcell.KeyEsc:
			a.cmdInput.SetText("")
			a.cmdHint.SetText("")
			a.cmdInput.SetLabel(" : ")
			a.SetFocus(a.table)
			return nil

		case tcell.KeyRune:
			// Check for number input (1-9) to select namespace
			if event.Rune() >= '0' && event.Rune() <= '9' && text == "" {
				// Show namespace hint
				a.showNamespaceHint()
			}
		}

		return event
	})

	a.cmdInput.SetDoneFunc(func(key tcell.Key) {
		// Already handled in InputCapture
	})
}

// getCompletions returns matching commands for the input
func (a *App) getCompletions(input string) []string {
	if input == "" {
		return nil
	}

	inputLower := strings.ToLower(input)
	var matches []string

	// Check for namespace command (ns <namespace>)
	if strings.HasPrefix(inputLower, "ns ") || strings.HasPrefix(inputLower, "namespace ") {
		prefix := strings.TrimPrefix(inputLower, "ns ")
		prefix = strings.TrimPrefix(prefix, "namespace ")

		a.mx.RLock()
		namespaces := a.namespaces
		a.mx.RUnlock()

		for _, ns := range namespaces {
			if ns == "" {
				continue
			}
			if strings.HasPrefix(ns, prefix) {
				matches = append(matches, "ns "+ns)
			}
		}
		return matches
	}

	// Check for resource command with -n flag (e.g., "pods -n kube")
	if strings.Contains(inputLower, " -n ") {
		parts := strings.Split(input, " -n ")
		if len(parts) == 2 {
			resourcePart := strings.TrimSpace(parts[0])
			nsPrefix := strings.TrimSpace(parts[1])

			a.mx.RLock()
			namespaces := a.namespaces
			a.mx.RUnlock()

			for _, ns := range namespaces {
				if ns == "" {
					continue
				}
				if strings.HasPrefix(ns, nsPrefix) {
					matches = append(matches, resourcePart+" -n "+ns)
				}
			}
			return matches
		}
	}

	// Check if input ends with "-n " - suggest namespaces
	if strings.HasSuffix(inputLower, "-n ") || strings.HasSuffix(inputLower, "-n") {
		basePart := strings.TrimSuffix(input, " ")
		if !strings.HasSuffix(basePart, " ") {
			basePart = strings.TrimSuffix(basePart, "-n") + "-n "
		}

		a.mx.RLock()
		namespaces := a.namespaces
		a.mx.RUnlock()

		for _, ns := range namespaces {
			if ns == "" {
				matches = append(matches, basePart+"all")
			} else {
				matches = append(matches, basePart+ns)
			}
		}
		// Limit suggestions
		if len(matches) > 10 {
			matches = matches[:10]
		}
		return matches
	}

	// Match built-in commands first
	for _, cmd := range commands {
		if strings.HasPrefix(cmd.name, inputLower) || strings.HasPrefix(cmd.alias, inputLower) {
			matches = append(matches, cmd.name)
		}
	}

	// Also match API resources from cluster (including CRDs)
	a.mx.RLock()
	apiResources := a.apiResources
	a.mx.RUnlock()

	seen := make(map[string]bool)
	for _, m := range matches {
		seen[m] = true
	}

	for _, res := range apiResources {
		if seen[res.Name] {
			continue
		}
		if strings.HasPrefix(res.Name, inputLower) {
			matches = append(matches, res.Name)
			seen[res.Name] = true
		}
		// Check short names
		for _, short := range res.ShortNames {
			if strings.HasPrefix(short, inputLower) && !seen[res.Name] {
				matches = append(matches, res.Name)
				seen[res.Name] = true
				break
			}
		}
	}

	return matches
}

// showNamespaceHint shows numbered namespace list in hint
func (a *App) showNamespaceHint() {
	a.mx.RLock()
	namespaces := a.namespaces
	a.mx.RUnlock()

	if len(namespaces) <= 1 {
		return
	}

	var hints []string
	for i, ns := range namespaces {
		if i == 0 {
			hints = append(hints, fmt.Sprintf("[gray]0[darkgray]:all"))
		} else if i <= 9 {
			hints = append(hints, fmt.Sprintf("[gray]%d[darkgray]:%s", i, ns))
		}
	}

	a.cmdHint.SetText(strings.Join(hints, " "))
}

// parseNamespaceNumber parses input as namespace number
func (a *App) parseNamespaceNumber(input string) (int, bool) {
	if len(input) != 1 {
		return 0, false
	}
	if input[0] >= '0' && input[0] <= '9' {
		return int(input[0] - '0'), true
	}
	return 0, false
}

// switchToAllNamespaces switches to all namespaces (k9s style: 0 key)
func (a *App) switchToAllNamespaces() {
	a.mx.Lock()
	a.currentNamespace = ""
	// Clear filter when switching namespace to avoid stale highlighting
	a.filterText = ""
	a.filterRegex = false
	a.mx.Unlock()

	a.flashMsg("Switched to: all namespaces", false)
	a.updateHeader()
	a.refresh()
}

// selectNamespaceByNumber selects namespace by number (for command mode)
func (a *App) selectNamespaceByNumber(num int) {
	a.mx.Lock()

	if num >= len(a.namespaces) {
		a.mx.Unlock()
		a.flashMsg(fmt.Sprintf("Namespace %d not available (max: %d)", num, len(a.namespaces)-1), true)
		return
	}

	selectedNs := a.namespaces[num]
	a.currentNamespace = selectedNs
	// Clear filter when switching namespace to avoid stale highlighting
	a.filterText = ""
	a.filterRegex = false
	nsName := selectedNs
	if nsName == "" {
		nsName = "all"
	}

	// Track recently used namespace (skip "all")
	if selectedNs != "" {
		a.addRecentNamespace(selectedNs)
	}
	a.mx.Unlock()

	a.flashMsg(fmt.Sprintf("Switched to namespace: %s", nsName), false)
	a.updateHeader()
	a.refresh()
}

// addRecentNamespace adds a namespace to the recent list (must be called with lock held)
func (a *App) addRecentNamespace(ns string) {
	if ns == "" {
		return
	}

	// Remove if already exists
	newRecent := make([]string, 0, a.maxRecentNamespaces)
	for _, r := range a.recentNamespaces {
		if r != ns {
			newRecent = append(newRecent, r)
		}
	}

	// Add to front
	newRecent = append([]string{ns}, newRecent...)

	// Trim to max size
	if len(newRecent) > a.maxRecentNamespaces {
		newRecent = newRecent[:a.maxRecentNamespaces]
	}

	a.recentNamespaces = newRecent
}

// reorderNamespacesByRecent reorders namespaces list by recent usage
// Returns a new slice with recent namespaces first, then others alphabetically
func (a *App) reorderNamespacesByRecent() []string {
	a.mx.RLock()
	allNamespaces := make([]string, len(a.namespaces))
	copy(allNamespaces, a.namespaces)
	recent := make([]string, len(a.recentNamespaces))
	copy(recent, a.recentNamespaces)
	a.mx.RUnlock()

	// Build result: "" (all) first, then recent, then others
	result := make([]string, 0, len(allNamespaces))

	// Add "all" (empty string) first if present
	hasAll := false
	for _, ns := range allNamespaces {
		if ns == "" {
			hasAll = true
			break
		}
	}
	if hasAll {
		result = append(result, "")
	}

	// Add recent namespaces (that still exist)
	nsSet := make(map[string]bool)
	for _, ns := range allNamespaces {
		nsSet[ns] = true
	}

	addedSet := make(map[string]bool)
	addedSet[""] = true // Already added "all"

	for _, ns := range recent {
		if nsSet[ns] && !addedSet[ns] {
			result = append(result, ns)
			addedSet[ns] = true
		}
	}

	// Add remaining namespaces (not in recent)
	remaining := make([]string, 0)
	for _, ns := range allNamespaces {
		if !addedSet[ns] {
			remaining = append(remaining, ns)
		}
	}
	// Sort remaining alphabetically
	sort.Strings(remaining)
	result = append(result, remaining...)

	return result
}

// startFilter activates filter mode
func (a *App) startFilter() {
	a.cmdInput.SetLabel(" / ")
	a.cmdHint.SetText("[gray]Type to filter (use /regex/ for regex), Enter to confirm, Esc to clear")
	a.cmdInput.SetText(a.filterText)
	a.SetFocus(a.cmdInput)

	// Debounce filter application to prevent UI lag/deadlock on rapid input
	var filterTimer *time.Timer
	var filterMu sync.Mutex

	a.cmdInput.SetChangedFunc(func(text string) {
		filterMu.Lock()
		if filterTimer != nil {
			filterTimer.Stop()
		}
		// Debounce: wait 100ms after last keystroke before applying filter
		filterTimer = time.AfterFunc(100*time.Millisecond, func() {
			a.applyFilterText(text)
		})
		filterMu.Unlock()
	})

	a.cmdInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			text := a.cmdInput.GetText()
			a.mx.Lock()
			// Check for regex pattern /pattern/
			if strings.HasPrefix(text, "/") && strings.HasSuffix(text, "/") && len(text) > 2 {
				a.filterText = text[1 : len(text)-1]
				a.filterRegex = true
			} else {
				a.filterText = text
				a.filterRegex = false
			}
			a.mx.Unlock()
			a.cmdInput.SetLabel(" : ")
			a.cmdHint.SetText("")
			a.restoreAutocompleteHandler()
			a.SetFocus(a.table)
			return nil

		case tcell.KeyEsc:
			a.mx.Lock()
			a.filterText = ""
			a.filterRegex = false
			a.mx.Unlock()
			a.cmdInput.SetText("")
			a.cmdInput.SetLabel(" : ")
			a.cmdHint.SetText("")
			a.applyFilterText("")
			a.restoreAutocompleteHandler()
			a.SetFocus(a.table)
			return nil

		case tcell.KeyRune:
			// Block special characters that might trigger other handlers
			// These should just be typed as filter text, not trigger commands
			switch event.Rune() {
			case '?', ':', '/':
				// Let these be typed as filter text (return event to continue processing)
				return event
			}
		}
		return event
	})
}

// applyFilterText filters the table based on the given text with regex support (k9s style)
func (a *App) applyFilterText(filter string) {
	// Read all needed state upfront to avoid lock inside QueueUpdateDraw
	a.mx.RLock()
	headers := a.tableHeaders
	rows := a.tableRows
	resource := a.currentResource
	a.mx.RUnlock()

	if len(headers) == 0 || len(rows) == 0 {
		return
	}

	// Check for regex pattern /pattern/
	isRegex := false
	filterPattern := filter
	if strings.HasPrefix(filter, "/") && strings.HasSuffix(filter, "/") && len(filter) > 2 {
		filterPattern = filter[1 : len(filter)-1]
		isRegex = true
	}

	// Compile regex if needed
	var re *regexp.Regexp
	var err error
	if isRegex && filterPattern != "" {
		re, err = regexp.Compile("(?i)" + filterPattern) // Case insensitive
		if err != nil {
			// Invalid regex, treat as plain text
			isRegex = false
			filterPattern = filter
		}
	}

	filterLower := strings.ToLower(filterPattern)

	a.QueueUpdateDraw(func() {
		a.table.Clear()

		// Set headers
		for i, h := range headers {
			cell := tview.NewTableCell(h).
				SetTextColor(tcell.ColorYellow).
				SetAttributes(tcell.AttrBold).
				SetSelectable(false).
				SetExpansion(1)
			a.table.SetCell(0, i, cell)
		}

		// Filter and set rows
		rowIdx := 1
		for _, row := range rows {
			if filterPattern != "" {
				match := false
				for _, cell := range row {
					if isRegex && re != nil {
						if re.MatchString(cell) {
							match = true
							break
						}
					} else {
						if strings.Contains(strings.ToLower(cell), filterLower) {
							match = true
							break
						}
					}
				}
				if !match {
					continue
				}
			}

			for c, text := range row {
				color := tcell.ColorWhite
				if c == 2 { // Usually status column
					color = a.statusColor(text)
				}
				// Highlight matching text
				displayText := text
				if filterPattern != "" {
					if isRegex && re != nil {
						displayText = a.highlightRegexMatch(text, re)
					} else if strings.Contains(strings.ToLower(text), filterLower) {
						displayText = a.highlightMatch(text, filterLower)
					}
				}
				cell := tview.NewTableCell(displayText).
					SetTextColor(color).
					SetExpansion(1)
				a.table.SetCell(rowIdx, c, cell)
			}
			rowIdx++
		}

		// Note: resource was read upfront before QueueUpdateDraw to avoid deadlock
		filterInfo := ""
		if filter != "" {
			if isRegex {
				filterInfo = fmt.Sprintf(" [regex: %s]", filterPattern)
			} else {
				filterInfo = fmt.Sprintf(" [filter: %s]", filter)
			}
		}
		a.table.SetTitle(fmt.Sprintf(" %s (%d/%d)%s ", resource, rowIdx-1, len(rows), filterInfo))

		if rowIdx > 1 {
			a.table.Select(1, 0)
		}
	})
}

// highlightMatch wraps matching text with color tags
func (a *App) highlightMatch(text, filter string) string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, filter)
	if idx < 0 {
		return text
	}

	before := text[:idx]
	match := text[idx : idx+len(filter)]
	after := text[idx+len(filter):]

	return before + "[yellow]" + match + "[white]" + after
}

// highlightRegexMatch wraps regex-matching text with color tags (k9s style)
func (a *App) highlightRegexMatch(text string, re *regexp.Regexp) string {
	if re == nil {
		return text
	}
	matches := re.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	var result strings.Builder
	lastEnd := 0
	for _, match := range matches {
		start, end := match[0], match[1]
		result.WriteString(text[lastEnd:start])
		result.WriteString("[yellow]")
		result.WriteString(text[start:end])
		result.WriteString("[white]")
		lastEnd = end
	}
	result.WriteString(text[lastEnd:])
	return result.String()
}

// restoreAutocompleteHandler restores the default autocomplete behavior
func (a *App) restoreAutocompleteHandler() {
	a.setupAutocomplete()
}

// setupKeybindings configures keyboard shortcuts (k9s compatible)
func (a *App) setupKeybindings() {
	a.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				a.Stop()
				return nil
			case ':':
				a.SetFocus(a.cmdInput)
				return nil
			case '/':
				a.startFilter()
				return nil
			case '?':
				a.showHelp()
				return nil
			case 'r':
				go a.refresh()
				return nil
			case 'n':
				a.cycleNamespace()
				return nil
			// k9s style: 0 = all namespaces, 1-9 = select namespace by number
			case '0':
				go a.switchToAllNamespaces()
				return nil
			case '1', '2', '3', '4', '5', '6', '7', '8', '9':
				num := int(event.Rune() - '0')
				go a.selectNamespaceByNumber(num)
				return nil
			case 'l':
				a.showLogs()
				return nil
			case 'p':
				a.showLogsPrevious()
				return nil
			case 'd':
				a.showDescribe() // k9s: d = describe
				return nil
			case 'y':
				a.showYAML() // k9s: y = yaml
				return nil
			case 'e':
				a.editResource() // k9s: e = edit
				return nil
			case 's':
				a.execShell() // k9s: s = shell
				return nil
			case 'a':
				a.attachContainer() // k9s: a = attach
				return nil
			case 'c':
				a.showContextSwitcher() // context switcher
				return nil
			case 'g':
				a.table.Select(1, 0) // go to top
				return nil
			case 'G':
				a.table.Select(a.table.GetRowCount()-1, 0) // go to bottom
				return nil
			case 'u':
				a.useNamespace() // k9s: u = use namespace
				return nil
			case 'o':
				a.showNode() // k9s: o = show node (for pods)
				return nil
			case 'O':
				a.showSettings() // Shift+O = settings/options
				return nil
			case 'k':
				a.killPod() // k9s: k or Ctrl+K = kill pod
				return nil
			case 'b':
				a.showBenchmark() // k9s: b = benchmark (services)
				return nil
			case 't':
				a.triggerCronJob() // k9s: t = trigger (cronjobs)
				return nil
			case 'z':
				a.showRelatedResource() // k9s: z = zoom (show related)
				return nil
			case 'F':
				a.portForward() // k9s: Shift+F = port-forward
				return nil
			case 'S':
				a.scaleResource() // k9s: Shift+S = scale
				return nil
			case 'R':
				a.restartResource() // k9s: Shift+R = restart
				return nil
			case 'B':
				a.toggleBriefing() // Shift+B = toggle briefing panel
				return nil
			// k9s-style column sorting (Shift + column key)
			case 'N':
				a.sortByColumnName("NAME") // Shift+N = sort by Name
				return nil
			case 'A':
				a.sortByColumnName("AGE") // Shift+A = sort by Age
				return nil
			case 'T':
				a.sortByColumnName("STATUS") // Shift+T = sort by sTatus
				return nil
			case 'P':
				a.sortByColumnName("NAMESPACE") // Shift+P = sort by namespace (ns)
				return nil
			case 'C':
				a.sortByColumnName("RESTARTS") // Shift+C = sort by restart Count
				return nil
			case 'D':
				a.sortByColumnName("READY") // Shift+D = sort by reaDy
				return nil
			case '!':
				a.sortByColumn(0) // Shift+1 = sort by column 1
				return nil
			case '@':
				a.sortByColumn(1) // Shift+2 = sort by column 2
				return nil
			case '#':
				a.sortByColumn(2) // Shift+3 = sort by column 3
				return nil
			case '$':
				a.sortByColumn(3) // Shift+4 = sort by column 4
				return nil
			case '%':
				a.sortByColumn(4) // Shift+5 = sort by column 5
				return nil
			case '^':
				a.sortByColumn(5) // Shift+6 = sort by column 6
				return nil
			case ' ':
				a.toggleSelection() // k9s: Space = toggle selection (multi-select)
				return nil
			}
		case tcell.KeyTab:
			if a.showAIPanel {
				a.SetFocus(a.aiInput)
			}
			return nil
		case tcell.KeyEnter:
			a.drillDown() // k9s: Enter = drill down to related resource
			return nil
		case tcell.KeyEsc:
			a.goBack() // k9s: Esc = go back
			return nil
		case tcell.KeyCtrlD:
			a.confirmDelete() // k9s: Ctrl+D = delete
			return nil
		case tcell.KeyCtrlK:
			a.killPod() // k9s: Ctrl+K = kill pod
			return nil
		case tcell.KeyCtrlU:
			a.pageUp() // k9s: Ctrl+U = page up
			return nil
		case tcell.KeyCtrlF:
			a.pageDown() // k9s: Ctrl+F = page down (vim style)
			return nil
		case tcell.KeyCtrlB:
			a.pageUp() // k9s: Ctrl+B = page up (vim style)
			return nil
		case tcell.KeyCtrlC:
			a.Stop()
			return nil
		case tcell.KeyCtrlI:
			a.aiBriefing() // Ctrl+I = AI-generated briefing
			return nil
		}
		return event
	})

	a.aiPanel.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			a.SetFocus(a.aiInput)
			return nil
		case tcell.KeyEnter:
			// Approve pending MCP tool call (use atomic for lock-free check)
			if atomic.LoadInt32(&a.hasToolCall) == 1 {
				a.approveToolCall(true)
				return nil
			}
		case tcell.KeyEsc:
			// Cancel pending MCP tool call (use atomic for lock-free check)
			if atomic.LoadInt32(&a.hasToolCall) == 1 {
				a.approveToolCall(false)
				a.SetFocus(a.table)
				return nil
			}
			// Clear pending decisions when escaping
			a.clearPendingDecisions()
			a.SetFocus(a.table)
			return nil
		case tcell.KeyRune:
			// Handle Y/N for MCP tool approval (kubectl-ai style)
			if atomic.LoadInt32(&a.hasToolCall) == 1 {
				switch event.Rune() {
				case 'y', 'Y':
					a.approveToolCall(true)
					return nil
				case 'n', 'N':
					a.approveToolCall(false)
					a.SetFocus(a.table)
					return nil
				}
			}
			// Handle decision input (1-9 to execute command, A to execute all)
			a.aiMx.RLock()
			numDecisions := len(a.pendingDecisions)
			a.aiMx.RUnlock()
			if numDecisions > 0 {
				switch event.Rune() {
				case '1', '2', '3', '4', '5', '6', '7', '8', '9':
					idx := int(event.Rune() - '1')
					if idx < numDecisions {
						go a.executeDecision(idx)
					}
					return nil
				case 'a', 'A':
					go a.executeAllDecisions()
					return nil
				}
			}
		}
		return event
	})
}

// flash displays a temporary message (k9s pattern)
func (a *App) flashMsg(msg string, isError bool) {
	color := "[green]"
	if isError {
		color = "[red]"
	}
	a.QueueUpdateDraw(func() {
		// Clear before setting to prevent ghosting
		a.flash.Clear()
		a.flash.SetText(color + msg + "[white]")
	})

	// Clear after 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		a.QueueUpdateDraw(func() {
			a.flash.Clear()
		})
	}()
}

// updateHeader updates the header text (thread-safe)
func (a *App) updateHeader() {
	ctxName := "N/A"
	cluster := "N/A"
	if a.k8s != nil {
		var err error
		ctxName, cluster, _, err = a.k8s.GetContextInfo()
		if err != nil {
			ctxName = "N/A"
			cluster = "N/A"
		}
	}

	a.mx.RLock()
	ns := a.currentNamespace
	resource := a.currentResource
	namespaces := a.namespaces
	a.mx.RUnlock()

	currentNsDisplay := "[green]all[white]"
	if ns != "" {
		currentNsDisplay = "[green]" + ns + "[white]"
	}

	aiStatus := "[red]Offline[white]"
	if a.aiClient != nil && a.aiClient.IsReady() {
		aiStatus = "[green]Online[white]"
	}

	// Build namespace quick-select preview (show first 9 namespaces with numbers)
	nsPreview := ""
	if len(namespaces) > 1 {
		var nsParts []string
		maxShow := 9
		if len(namespaces) < maxShow {
			maxShow = len(namespaces)
		}
		for i := 0; i < maxShow; i++ {
			nsName := namespaces[i]
			if nsName == "" {
				nsName = "all"
			}
			// Highlight current namespace
			if (ns == "" && nsName == "all") || ns == nsName {
				nsParts = append(nsParts, fmt.Sprintf("[yellow]%d[white]:[green::b]%s[white::-]", i, truncateNsName(nsName, 12)))
			} else {
				nsParts = append(nsParts, fmt.Sprintf("[yellow]%d[white]:[gray]%s[white]", i, truncateNsName(nsName, 12)))
			}
		}
		if len(namespaces) > maxShow {
			nsParts = append(nsParts, fmt.Sprintf("[gray]+%d more[white]", len(namespaces)-maxShow))
		}
		nsPreview = " " + strings.Join(nsParts, " ")
	}

	header := fmt.Sprintf(
		" [yellow::b]k13d[white::-] - Kubernetes AI Dashboard                                    AI: %s\n"+
			" [gray]Context:[white] %s  [gray]Cluster:[white] %s  [gray]NS:[white] %s  [gray]Resource:[white] [cyan]%s[white]\n"+
			" [gray]Namespaces:[white]%s",
		aiStatus, ctxName, cluster, currentNsDisplay, resource, nsPreview,
	)

	// Use QueueUpdateDraw only after Application.Run() has started (k9s pattern)
	if atomic.LoadInt32(&a.running) == 1 {
		a.QueueUpdateDraw(func() {
			// Clear before setting new text to prevent ghosting artifacts
			a.header.Clear()
			a.header.SetText(header)
		})
	} else {
		// Direct update during initialization (before Run())
		a.header.Clear()
		a.header.SetText(header)
	}
}

// truncateNsName truncates namespace name for display
func truncateNsName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}
	return name[:maxLen-1] + "…"
}

// updateStatusBar updates the status bar (k9s style)
func (a *App) updateStatusBar() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	// k9s style status bar: show key shortcuts
	shortcuts := "[yellow]<n>[white]NS [yellow]<0>[white]All [yellow]</>[white]Filter [yellow]<:>[white]Cmd [yellow]<?>[white]Help [yellow]<q>[white]Quit"

	// Add resource-specific shortcuts
	switch resource {
	case "pods", "po":
		shortcuts = "[yellow]<l>[white]Logs [yellow]<s>[white]Shell [yellow]<d>[white]Describe " + shortcuts
	case "deployments", "deploy", "statefulsets", "sts", "daemonsets", "ds":
		shortcuts = "[yellow]<S>[white]Scale [yellow]<R>[white]Restart [yellow]<d>[white]Describe " + shortcuts
	case "namespaces", "ns":
		shortcuts = "[yellow]<u>[white]Use " + shortcuts
	default:
		shortcuts = "[yellow]<d>[white]Describe [yellow]<y>[white]YAML " + shortcuts
	}

	// Clear before setting to prevent ghosting
	a.statusBar.Clear()
	a.statusBar.SetText(shortcuts)
}

// prepareContext cancels previous operations and creates new context (k9s pattern)
func (a *App) prepareContext() context.Context {
	a.cancelLock.Lock()
	defer a.cancelLock.Unlock()

	if a.cancelFn != nil {
		a.cancelFn() // Cancel previous operation
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.cancelFn = cancel
	return ctx
}

// refresh reloads the current resource list with atomic guard (k9s pattern)
func (a *App) refresh() {
	// Atomic guard to prevent concurrent updates (k9s pattern)
	if !atomic.CompareAndSwapInt32(&a.inUpdate, 0, 1) {
		a.logger.Debug("Dropping refresh - update already in progress")
		return
	}
	defer atomic.StoreInt32(&a.inUpdate, 0)

	ctx := a.prepareContext()

	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	// Show loading state
	a.QueueUpdateDraw(func() {
		a.table.Clear()
		a.table.SetTitle(fmt.Sprintf(" %s - Loading... ", resource))
		a.table.SetCell(0, 0, tview.NewTableCell("Loading...").SetTextColor(tcell.ColorYellow))
	})

	// Fetch with exponential backoff (k9s pattern)
	var headers []string
	var rows [][]string
	var fetchErr error

	bf := backoff.NewExponentialBackOff()
	bf.InitialInterval = 300 * time.Millisecond
	bf.MaxElapsedTime = 10 * time.Second

	err := backoff.Retry(func() error {
		select {
		case <-ctx.Done():
			return backoff.Permanent(ctx.Err())
		default:
		}

		headers, rows, fetchErr = a.fetchResources(ctx)
		if fetchErr != nil {
			a.logger.Warn("Fetch failed, retrying", "error", fetchErr, "resource", resource)
			return fetchErr
		}
		return nil
	}, backoff.WithContext(bf, ctx))

	if err != nil {
		a.logger.Error("Fetch failed after retries", "error", err, "resource", resource)
		a.flashMsg(fmt.Sprintf("Error: %v", err), true)
		a.QueueUpdateDraw(func() {
			a.table.Clear()
			a.table.SetTitle(fmt.Sprintf(" %s - Error ", resource))
			a.table.SetCell(0, 0, tview.NewTableCell(fmt.Sprintf("Error: %v", err)).SetTextColor(tcell.ColorRed))
		})
		return
	}

	// Store original data for filtering
	a.mx.Lock()
	a.tableHeaders = headers
	a.tableRows = rows
	currentFilter := a.filterText
	a.mx.Unlock()

	// Apply filter if active, otherwise show all
	if currentFilter != "" {
		a.applyFilterText(currentFilter)
	} else {
		// Update UI (k9s pattern: queue all UI updates)
		a.QueueUpdateDraw(func() {
			a.table.Clear()

			// Set headers
			for i, h := range headers {
				cell := tview.NewTableCell(h).
					SetTextColor(tcell.ColorYellow).
					SetAttributes(tcell.AttrBold).
					SetSelectable(false).
					SetExpansion(1)
				a.table.SetCell(0, i, cell)
			}

			// Set rows
			for r, row := range rows {
				for c, text := range row {
					color := tcell.ColorWhite
					if c == 2 { // Usually status column
						color = a.statusColor(text)
					}
					cell := tview.NewTableCell(text).
						SetTextColor(color).
						SetExpansion(1)
					a.table.SetCell(r+1, c, cell)
				}
			}

			count := len(rows)
			a.table.SetTitle(fmt.Sprintf(" %s (%d) ", resource, count))

			if count > 0 {
				a.table.Select(1, 0)
			}
		})
	}

	// Update status bar for resource-specific shortcuts
	a.QueueUpdateDraw(func() {
		a.updateStatusBar()
	})

	// Update briefing panel if visible
	if a.briefing != nil && a.briefing.IsVisible() {
		go a.briefing.Update(ctx)
	}

	a.logger.Info("Refresh completed", "resource", resource, "count", len(rows))
}

// QueueUpdateDraw queues up a UI action and redraws.
// Note: Unlike k9s, we don't wrap in goroutine here because tview's
// QueueUpdateDraw is already thread-safe and non-blocking when called
// from outside the main goroutine. The k9s pattern caused timing issues
// where UI updates happened after function returns.
//
// IMPORTANT: This method checks if the app is running before queuing.
// This prevents goroutines from blocking forever after app.Stop() is called.
// It uses a timeout to prevent indefinite blocking on the update channel.
func (a *App) QueueUpdateDraw(f func()) {
	if a.Application == nil {
		return
	}
	// Check if app is stopping or not running - skip to avoid blocking
	if atomic.LoadInt32(&a.stopping) == 1 || atomic.LoadInt32(&a.running) == 0 {
		return
	}
	// Use a timeout wrapper to prevent indefinite blocking
	// This can happen if too many updates are queued or the app is shutting down
	done := make(chan struct{})
	go func() {
		defer close(done)
		a.Application.QueueUpdateDraw(f)
	}()
	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		// Timeout - app might be shutting down or overloaded
		return
	}
}

// IsRunning returns true if the application is currently running and not stopping.
func (a *App) IsRunning() bool {
	return atomic.LoadInt32(&a.running) == 1 && atomic.LoadInt32(&a.stopping) == 0
}

// Stop stops the application and prevents any further UI updates.
// This is safe to call from any goroutine.
func (a *App) Stop() {
	// Set stopping flag immediately to prevent new QueueUpdateDraw calls
	atomic.StoreInt32(&a.stopping, 1)

	// Stop briefing pulse animation to prevent goroutine leaks
	if a.briefing != nil {
		a.briefing.stopPulseAnimation()
	}

	// Now stop the tview application
	if a.Application != nil {
		a.Application.Stop()
	}
}

// Run starts the application with panic recovery (k9s pattern)
func (a *App) Run() error {
	// Top-level panic recovery (k9s pattern)
	defer func() {
		if err := recover(); err != nil {
			a.logger.Error("PANIC RECOVERED", "error", err, "stack", string(debug.Stack()))
			fmt.Fprintf(os.Stderr, "\n[FATAL] k13d crashed: %v\n", err)
		}
	}()

	// Ensure running flag is reset when app stops
	defer func() {
		// Set stopping flag and stop pulse animation (in case Stop() wasn't called)
		atomic.StoreInt32(&a.stopping, 1)
		if a.briefing != nil {
			a.briefing.stopPulseAnimation()
		}
		atomic.StoreInt32(&a.running, 0)
	}()

	// Mark as running and trigger initial refresh after first draw
	a.SetAfterDrawFunc(func(screen tcell.Screen) {
		a.SetAfterDrawFunc(nil) // Only run once
		atomic.StoreInt32(&a.running, 1)
		go a.refresh()

		// Start briefing pulse animation if visible
		if a.briefing != nil && a.briefing.IsVisible() {
			a.briefing.startPulse()
		}
	})

	a.logger.Info("Starting k13d TUI")
	return a.Application.Run()
}

// approveToolCall handles tool call approval/rejection (deadlock-safe)
func (a *App) approveToolCall(approved bool) {
	// Use non-blocking send to avoid potential deadlock
	select {
	case a.pendingToolApproval <- approved:
		// Clear tool call state atomically first, then update struct
		atomic.StoreInt32(&a.hasToolCall, 0)
		a.aiMx.Lock()
		a.currentToolCallInfo = struct {
			Name    string
			Args    string
			Command string
		}{}
		a.aiMx.Unlock()
	default:
		// Channel full or no receiver, ignore
	}
}

// setToolCallState sets the current tool call info (deadlock-safe)
func (a *App) setToolCallState(name, args, command string) {
	a.aiMx.Lock()
	a.currentToolCallInfo.Name = name
	a.currentToolCallInfo.Args = args
	a.currentToolCallInfo.Command = command
	a.aiMx.Unlock()
	// Set atomic flag after lock release
	atomic.StoreInt32(&a.hasToolCall, 1)
}

// clearToolCallState clears the tool call state (deadlock-safe)
func (a *App) clearToolCallState() {
	atomic.StoreInt32(&a.hasToolCall, 0)
	a.aiMx.Lock()
	a.currentToolCallInfo = struct {
		Name    string
		Args    string
		Command string
	}{}
	a.aiMx.Unlock()
}

// clearPendingDecisions clears pending decisions and shows message (deadlock-safe)
func (a *App) clearPendingDecisions() {
	a.aiMx.Lock()
	hadDecisions := len(a.pendingDecisions) > 0
	a.pendingDecisions = nil
	a.aiMx.Unlock()

	// Show message after lock release to avoid deadlock
	if hadDecisions {
		a.flashMsg("Cancelled pending commands", false)
	}
}
