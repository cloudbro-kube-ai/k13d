package cli

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	"github.com/cloudbro-kube-ai/k13d/pkg/log"
)

// ──────────────────────────────────────────────
// tview-compatible logo rendering (opencode style)
// ──────────────────────────────────────────────

// LogoLinesTview returns the k13d logo in tview-compatible [color] format.
func LogoLinesTview() string {
	lines := []string{
		"██   ██   █████   ██████   ██████            ██████   ██       ██████  ",
		"██  ██      ██        ██   ██   ██           ██       ██         ██    ",
		"██ ██       ██    ██████   ██   ██           ██       ██         ██    ",
		"████        ██        ██   ██   ██           ██       ██         ██    ",
		"██ ██       ██        ██   ██   ██           ██       ██         ██    ",
		"██  ██      ██    ██████   ██████            ██████   ██████   ██████  ",
	}
	colors := []string{
		"[#00ffff]", "[#00ddff]", "[#00bbff]",
		"[#0099ff]", "[#0077ff]", "[#55aaff]",
	}
	var buf strings.Builder
	for i, line := range lines {
		color := colors[i]
		if i >= len(colors) {
			color = colors[len(colors)-1]
		}
		buf.WriteString(color)
		buf.WriteString(line)
		buf.WriteString("[-]")
		if i < len(lines)-1 {
			buf.WriteString("\n")
		}
	}
	return buf.String()
}

// HelpText returns the help screen as a tview-compatible string.
func HelpText() string {
	return strings.TrimLeft(`
[#00ddff]Built-in Commands:[-]
  [white]:help[-]                   Show this help message
  [white]:quit[-], [white]:exit[-]            Exit CLI mode
  [white]:clear[-]                  Clear output screen
  [white]:version[-]                Show k13d version information
  [white]:namespace <name>[-]       Set default namespace
  [white]:context <name>[-]         Switch Kubernetes context
  [white]:history[-]                Show command history
  [white]:ai <question>[-]          Ask AI about your cluster

[#00ddff]Any other input is executed as a kubectl command.[-]

[#00ddff]Examples:[-]
  [white]get pods[-]
  [white]get pods -n kube-system[-]
  [white]get deployments[-]
  [white]describe pod nginx-xxx[-]
  [white]logs pod/nginx-xxx[-]
  [white]get nodes[-]

[#00ddff]Navigation:[-]
  [white]Up/Down arrows[-]    Navigate command history
  [white]Tab[-]               Auto-complete commands
  [white]Ctrl+C[-]            Exit
  [white]Ctrl+L[-]            Clear output
`, "\n")
}

// ──────────────────────────────────────────────
// TUI REPL — opencode-style split layout
// ──────────────────────────────────────────────

// StartTUI starts the enhanced tview-based CLI REPL.
//
// Layout (opencode style):
//
//	┌─────────────────────────────────────┐
//	│  Logo (first launch only)           │
//	├─────────────────────────────────────┤
//	│  Output / Results (scrollable)      │
//	│                                     │
//	├─────────────────────────────────────┤
//	│  ▶ Input (fixed at bottom)          │
//	└─────────────────────────────────────┘
//
// After the first command, the logo area is hidden
// and only Output + Input remain visible.
func (c *CLI) StartTUI() error {
	// ── Initialize clients ──
	var err error
	c.client, err = k8s.NewClient()
	if err != nil {
		log.Warnf("Failed to create Kubernetes client: %v", err)
	}
	if c.namespace == "" {
		c.namespace = "default"
	}
	if c.cfg.LLM.Provider != "" {
		ac, err := ai.NewClient(&c.cfg.LLM)
		if err == nil {
			c.aiClient = ac
		}
	}

	app := tview.NewApplication()

	// ── Logo view (shown on first launch, hidden after first command) ──
	logoView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	logoView.SetText(LogoLinesTview())
	logoView.SetBackgroundColor(tcell.ColorDefault)

	versionDisplay := c.version.Version
	if versionDisplay == "dev" || versionDisplay == "" {
		versionDisplay = ""
	} else {
		versionDisplay = "v" + versionDisplay
	}
	taglineText := "[#87d7ff]kubernest AI Cli[-]"
	if versionDisplay != "" {
		taglineText += "\n[#5f87af]" + versionDisplay + "[-]"
	}
	taglineView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	taglineView.SetText(taglineText)
	taglineView.SetBackgroundColor(tcell.ColorDefault)

	// Logo area — collapsed by clearing its text after the first command
	logoArea := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(logoView, 6, 0, false).
		AddItem(taglineView, 2, 0, false)

	// ── Output area (scrollable results) ──
	outputView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	outputView.SetBorder(true).
		SetTitle(" Output ").
		SetBorderColor(tcell.NewRGBColor(99, 148, 245))
	outputView.SetBackgroundColor(tcell.ColorDefault)

	// ── Input field (fixed at bottom, like opencode) ──
	inputView := tview.NewInputField().
		SetLabel("[#00ddff]▶[-] ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetPlaceholder("Type a kubectl command or :help...")
	inputView.SetPlaceholderStyle(tcell.StyleDefault.
		Foreground(tcell.NewRGBColor(80, 80, 80)))

	// ── Shared state ──
	var (
		logoShown = true
		mu        sync.Mutex
	)

	// Thread-safe helper: append text to the output view
	appendOutput := func(text string) {
		mu.Lock()
		defer mu.Unlock()
		if text == "" {
			return
		}
		fmt.Fprint(outputView, text)
	}
	// ── SetDoneFunc handles Enter (same pattern as ai_panel.go) ──
	inputView.SetDoneFunc(func(key tcell.Key) {
		text := inputView.GetText()
		if text == "" {
			return
		}
		inputView.SetText("")
		c.history.Add(text)

		// Echo: show the prompt + input in the output area
		appendOutput("[#00ddff]▶[-] " + text + "\n")

		// Hide logo after first command (opencode style)
		if logoShown {
			logoShown = false
			app.QueueUpdateDraw(func() {
				logoView.SetText("")
				taglineView.SetText("")
			})
		}

		// Dispatch command
		result := c.tuiDispatch(app, inputView, outputView, text)
		if result != "" {
			appendOutput(result)
			if !strings.HasSuffix(result, "\n") {
				appendOutput("\n")
			}
		}

		app.ForceDraw()
	})

	// ── SetInputCapture handles Tab, Up/Down, Ctrl+L (NOT Enter) ──
	inputView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			c.tuiAutoComplete(inputView, outputView, &mu, appendOutput)
			return nil

		case tcell.KeyUp:
			if entry, ok := c.history.Previous(); ok {
				inputView.SetText(entry)
			}
			return nil

		case tcell.KeyDown:
			if entry, ok := c.history.Next(); ok {
				inputView.SetText(entry)
			} else {
				inputView.SetText("")
			}
			return nil

		case tcell.KeyCtrlL:
			outputView.SetText("")
			return nil
		}
		return event
	})

	// ── Main layout ──
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(logoArea, 8, 0, false).   // Logo (hidden after first command)
		AddItem(outputView, 0, 1, false). // Output (fills remaining space)
		AddItem(inputView, 1, 0, true)    // Input (1 line fixed at bottom, focused)

	app.SetRoot(flex, true)

	// ── App-level key handling ──
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			app.Stop()
			return nil
		}
		return event
	})

	return app.Run()
}

// ──────────────────────────────────────────────
// Command dispatch (TUI mode)
// ──────────────────────────────────────────────

// tuiDispatch routes a command and returns the output string.
// For :quit/:exit it stops the tview application.
func (c *CLI) tuiDispatch(app *tview.Application, inputView *tview.InputField, outputView *tview.TextView, input string) string {
	if strings.HasPrefix(input, ":") {
		return c.tuiHandleBuiltin(app, outputView, input[1:])
	}
	return c.tuiExecuteKubectl(input)
}

// tuiHandleBuiltin processes built-in :commands and returns output as a string.
func (c *CLI) tuiHandleBuiltin(app *tview.Application, outputView *tview.TextView, cmdLine string) string {
	cmdLine = strings.TrimSpace(cmdLine)
	parts := strings.Fields(cmdLine)
	if len(parts) == 0 {
		return HelpText()
	}
	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "help":
		return HelpText()
	case "quit", "exit":
		app.Stop()
		return ""
	case "clear":
		outputView.SetText("")
		return ""
	case "version":
		return fmt.Sprintf("k13d version %s\n  Build time: %s\n  Git commit: %s\n",
			c.version.Version, c.version.BuildTime, c.version.GitCommit)
	case "namespace":
		return c.tuiHandleNamespace(args)
	case "context":
		return c.tuiHandleContext(args)
	case "history":
		return c.tuiPrintHistory()
	case "ai":
		if len(args) == 0 {
			return "Usage: :ai <question>\n"
		}
		return c.tuiHandleAI(args)
	default:
		return fmt.Sprintf("Unknown command: :%s\nType :help for available commands.\n", cmd)
	}
}

// tuiHandleNamespace shows or sets the namespace.
func (c *CLI) tuiHandleNamespace(args []string) string {
	if len(args) == 0 {
		return fmt.Sprintf("Current namespace: %s\n", c.namespace)
	}
	c.namespace = args[0]
	return fmt.Sprintf("Namespace set to: %s\n", c.namespace)
}

// tuiHandleContext shows or switches the Kubernetes context.
func (c *CLI) tuiHandleContext(args []string) string {
	if c.client == nil {
		return "Kubernetes client not available\n"
	}
	if len(args) == 0 {
		cur, err := c.client.GetCurrentContext()
		if err != nil {
			return fmt.Sprintf("Error getting context: %v\n", err)
		}
		return fmt.Sprintf("Current context: %s\n", cur)
	}
	err := c.client.SwitchContext(args[0])
	if err != nil {
		return fmt.Sprintf("Error switching context: %v\n", err)
	}
	return fmt.Sprintf("Context switched to: %s\n", args[0])
}

// tuiPrintHistory returns the command history as a string.
func (c *CLI) tuiPrintHistory() string {
	entries := c.history.Entries()
	if len(entries) == 0 {
		return "No command history.\n"
	}
	var buf strings.Builder
	buf.WriteString("Command history:\n")
	for i, entry := range entries {
		buf.WriteString(fmt.Sprintf("  %3d  %s\n", i+1, entry))
	}
	return buf.String()
}

// tuiHandleAI sends a question to the AI and returns the response.
func (c *CLI) tuiHandleAI(args []string) string {
	if c.aiClient == nil {
		return "AI provider not configured. Set LLM settings in config.yaml.\n"
	}

	question := strings.Join(args, " ")
	systemPrompt := fmt.Sprintf(
		"You are a Kubernetes assistant running in CLI mode. "+
			"The current namespace is '%s'. "+
			"Provide concise, helpful responses for Kubernetes tasks. "+
			"Keep responses to 2-3 paragraphs maximum unless asked for details.",
		c.namespace,
	)

	var respBuilder strings.Builder
	ctx := context.Background()
	fullPrompt := systemPrompt + "\n\nUser question: " + question

	err := c.aiClient.Ask(ctx, fullPrompt, func(chunk string) {
		respBuilder.WriteString(chunk)
	})
	if err != nil {
		log.Debugf("Streaming AI call failed: %v, falling back to non-streaming", err)
		resp, fallbackErr := c.aiClient.AskNonStreaming(ctx, fullPrompt)
		if fallbackErr != nil {
			return fmt.Sprintf("Error getting AI response: %v\n", fallbackErr)
		}
		return resp + "\n"
	}

	return "--- AI Response ---\n" + respBuilder.String() + "\n------------------\n"
}

// tuiExecuteKubectl runs a kubectl command and returns the output.
func (c *CLI) tuiExecuteKubectl(input string) string {
	kubectlInput := input
	hasNamespace := strings.Contains(input, "--namespace") ||
		strings.Contains(input, "-n ")
	if !hasNamespace && c.namespace != "" && c.namespace != "default" {
		kubectlInput = input + " --namespace " + c.namespace
	}

	output, err := runKubectlCommand(kubectlInput)
	if err != nil {
		return fmt.Sprintf("[red]Error:[-] %s\n", err.Error())
	}
	return output
}

// ──────────────────────────────────────────────
// Tab autocomplete (opencode style)
// ──────────────────────────────────────────────

// tuiAutoComplete provides Tab autocomplete for commands.
func (c *CLI) tuiAutoComplete(
	inputView *tview.InputField,
	outputView *tview.TextView,
	mu *sync.Mutex,
	appendOutput func(string),
) {
	text := inputView.GetText()

	// :command completion
	if strings.HasPrefix(text, ":") {
		prefix := strings.TrimPrefix(text, ":")
		cmds := []string{"help", "quit", "exit", "clear", "version", "namespace", "context", "history", "ai"}
		matches := filterPrefixes(cmds, prefix)
		switch len(matches) {
		case 0:
			return
		case 1:
			inputView.SetText(":" + matches[0] + " ")
		default:
			appendOutput(fmt.Sprintf("[#5f87af]Commands: %s[-]\n", strings.Join(matches, ", ")))
		}
		return
	}

	// kubectl command completion
	if !strings.Contains(text, " ") {
		kubectlCmds := []string{
			"get", "describe", "delete", "logs", "exec", "apply",
			"create", "edit", "top", "cp", "auth", "config",
			"cluster-info", "explain", "port-forward", "proxy",
			"rollout", "scale", "autoscale", "certificate",
			"plugin", "version", "api-resources", "api-versions",
		}
		matches := filterPrefixes(kubectlCmds, text)
		switch len(matches) {
		case 0:
			return
		case 1:
			inputView.SetText(matches[0] + " ")
		default:
			appendOutput(fmt.Sprintf("[#5f87af]Commands: %s[-]\n", strings.Join(matches, ", ")))
		}
		return
	}
}

// filterPrefixes returns strings from list that have the given prefix.
func filterPrefixes(list []string, prefix string) []string {
	if prefix == "" {
		return list
	}
	var result []string
	for _, s := range list {
		if strings.HasPrefix(s, prefix) {
			result = append(result, s)
		}
	}
	return result
}
