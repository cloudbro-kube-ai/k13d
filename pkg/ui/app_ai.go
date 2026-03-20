package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type aiPromptContext struct {
	Resource          string
	Namespace         string
	SelectedResource  string
	SelectedName      string
	SelectedNamespace string
	SelectedSummary   string
	DetailedContext   string
}

type aiAttachedSelection struct {
	Resource  string
	Namespace string
	Name      string
	Summary   string
}

func (s aiAttachedSelection) IsZero() bool {
	return strings.TrimSpace(s.Resource) == "" || strings.TrimSpace(s.Name) == ""
}

func (s aiAttachedSelection) Matches(other aiAttachedSelection) bool {
	return strings.EqualFold(strings.TrimSpace(s.Resource), strings.TrimSpace(other.Resource)) &&
		strings.TrimSpace(s.Namespace) == strings.TrimSpace(other.Namespace) &&
		strings.TrimSpace(s.Name) == strings.TrimSpace(other.Name)
}

func (s aiAttachedSelection) TargetLabel() string {
	target := strings.TrimSpace(s.Name)
	if ns := strings.TrimSpace(s.Namespace); ns != "" {
		target = ns + "/" + target
	}
	return target
}

func buildAIPrompt(question string, ctx aiPromptContext) string {
	var prompt strings.Builder
	prompt.WriteString("You are helping a user inside the k13d terminal UI.\n")
	if ctx.Resource != "" {
		prompt.WriteString(fmt.Sprintf("Current resource view: %s.\n", ctx.Resource))
	}
	if ctx.Namespace == "" {
		prompt.WriteString("Namespace scope: all namespaces.\n")
	} else {
		prompt.WriteString(fmt.Sprintf("Namespace scope: %s.\n", ctx.Namespace))
	}
	if ctx.SelectedSummary != "" {
		prompt.WriteString(fmt.Sprintf("Selected row: %s.\n", ctx.SelectedSummary))
	}
	if ctx.SelectedName != "" && ctx.SelectedResource != "" {
		prompt.WriteString(fmt.Sprintf("Selected object: %s/%s.\n", ctx.SelectedResource, ctx.SelectedName))
		if ctx.SelectedNamespace != "" {
			prompt.WriteString(fmt.Sprintf("Selected object namespace: %s.\n", ctx.SelectedNamespace))
		}
	}
	if ctx.DetailedContext != "" {
		prompt.WriteString("\nSelected resource context:\n")
		prompt.WriteString(ctx.DetailedContext)
		prompt.WriteString("\n")
	}
	prompt.WriteString("\nUser question:\n")
	prompt.WriteString(question)
	prompt.WriteString("\n\nProvide a concise, evidence-based answer. If you suggest kubectl commands, explain why.")
	return prompt.String()
}

func trimAIBlock(text string, maxRunes int) string {
	trimmed := strings.TrimSpace(text)
	if maxRunes <= 0 {
		return trimmed
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "\n...[truncated]"
}

func summarizeAIToolResult(result string) string {
	if strings.TrimSpace(result) == "" {
		return "(no output)"
	}
	return trimAIBlock(result, 420)
}

func defaultAIInputStatusText() string {
	return "[gray]Ready[-] Enter send | Shift+Tab history | Esc close | Up/Down history | Alt+H/L resize | /help"
}

func defaultAITranscriptStatusText() string {
	return "[gray]History[-] j/k or PgUp/PgDn scroll | g/G top/bottom | Tab prompt | Esc close"
}

func isAIReadyStatusText(text string) bool {
	return strings.Contains(text, "Enter send") ||
		strings.Contains(text, "Shift+Tab history") ||
		strings.Contains(text, "j/k or PgUp/PgDn scroll") ||
		strings.Contains(text, "Tab prompt")
}

func (a *App) readyAIStatusText() string {
	if a != nil && a.GetFocus() == a.aiPanel {
		return defaultAITranscriptStatusText()
	}
	return defaultAIInputStatusText()
}

func (a *App) updateAIReadyStatusIfIdle() {
	if a.aiStatusBar == nil {
		return
	}
	current := strings.TrimSpace(a.aiStatusBar.GetText(false))
	if current == "" || isAIReadyStatusText(current) {
		a.setAIStatus(a.readyAIStatusText())
	}
}

func (a *App) focusAIInput() {
	if a.aiInput == nil {
		return
	}
	a.SetFocus(a.aiInput)
	a.updateAIReadyStatusIfIdle()
}

func (a *App) focusAITranscript() {
	if a.aiPanel == nil {
		return
	}
	a.SetFocus(a.aiPanel)
	a.updateAIReadyStatusIfIdle()
}

func (a *App) resetAIConversation() {
	if a.aiPanel == nil {
		return
	}
	a.aiMx.Lock()
	a.aiConversationTurns = 0
	a.aiMx.Unlock()
	a.aiPanel.Clear()
	fmt.Fprint(a.aiPanel,
		"[#7aa2f7::b]AI Assistant[-::-]\n"+
			"[gray]Ask about the current resource, or press Enter on a selected row to attach deeper Kubernetes context for AI.[-]\n\n"+
			"[#565f89]Try:[-]\n"+
			"  [#9ece6a]-[-] Why is this workload failing?\n"+
			"  [#9ece6a]-[-] Explain this resource\n"+
			"  [#9ece6a]-[-] Show me the rollout risk\n\n"+
			"[#565f89]Commands:[-] /context  /clear  /help\n"+
			"[#565f89]Keys:[-] Enter attach row  Shift+Tab history  Tab prompt  j/k or PgUp/PgDn scroll\n")
	a.aiPanel.ScrollTo(0, 0)
	a.setAIStatus(a.readyAIStatusText())
}

func (a *App) setAIStatus(text string) {
	if a.aiStatusBar == nil {
		return
	}
	a.aiStatusBar.SetText(text)
}

func (a *App) currentAISelectionCandidate() aiAttachedSelection {
	if a.table == nil {
		return aiAttachedSelection{}
	}

	row, _ := a.table.GetSelection()
	return a.aiSelectionCandidateForRow(row)
}

func (a *App) getAttachedAIContext() aiAttachedSelection {
	a.aiMx.RLock()
	defer a.aiMx.RUnlock()
	return a.attachedAIContext
}

func (a *App) toggleSelectedAIContext() {
	candidate := a.currentAISelectionCandidate()
	if candidate.IsZero() {
		a.flashMsg("Select a row first to attach AI context", true)
		return
	}

	attachedLabel := candidate.TargetLabel()
	if attachedLabel == "" {
		attachedLabel = candidate.Name
	}

	attached := true
	a.aiMx.Lock()
	if a.attachedAIContext.Matches(candidate) {
		a.attachedAIContext = aiAttachedSelection{}
		attached = false
	} else {
		a.attachedAIContext = candidate
	}
	a.aiMx.Unlock()

	a.QueueUpdateDraw(func() {
		a.refreshTableDecorations()
		a.applyAIChrome()
		a.updateAIReadyStatusIfIdle()
	})

	if attached {
		a.flashMsg(fmt.Sprintf("AI context attached: %s %s", candidate.Resource, attachedLabel), false)
		return
	}
	a.flashMsg("AI context detached", false)
}

func (a *App) getAIPromptContext() aiPromptContext {
	a.mx.RLock()
	resource := a.currentResource
	ns := a.currentNamespace
	a.mx.RUnlock()

	snapshot := aiPromptContext{
		Resource:  resource,
		Namespace: ns,
	}

	attached := a.getAttachedAIContext()
	if attached.IsZero() {
		return snapshot
	}

	snapshot.SelectedResource = attached.Resource
	snapshot.SelectedName = attached.Name
	snapshot.SelectedNamespace = attached.Namespace
	snapshot.SelectedSummary = attached.Summary

	return snapshot
}

func (a *App) loadDetailedAIContext(base aiPromptContext) aiPromptContext {
	if a.k8s == nil || base.SelectedName == "" || base.SelectedResource == "" {
		return base
	}

	ctx, cancel := context.WithTimeout(a.getAppContext(), 4*time.Second)
	defer cancel()

	resourceContext, err := a.k8s.GetResourceContext(ctx, base.SelectedNamespace, base.SelectedName, base.SelectedResource)
	if err != nil {
		return base
	}
	base.DetailedContext = trimAIBlock(resourceContext, 12000)
	return base
}

func (a *App) formatAIContextLabel(ctx aiPromptContext) string {
	if ctx.SelectedName == "" {
		if candidate := a.currentAISelectionCandidate(); !candidate.IsZero() {
			return fmt.Sprintf(
				"[#7dcfff]%s %s[-] [gray](selected, Enter attach)[-]",
				tview.Escape(candidate.Resource),
				tview.Escape(candidate.TargetLabel()),
			)
		}
		scope := "all namespaces"
		if ctx.Namespace != "" {
			scope = ctx.Namespace
		}
		return fmt.Sprintf("[#7dcfff]%s[-] [gray](scope: %s)[-]", tview.Escape(ctx.Resource), tview.Escape(scope))
	}

	target := ctx.SelectedName
	if ctx.SelectedNamespace != "" {
		target = ctx.SelectedNamespace + "/" + target
	}
	resource := ctx.SelectedResource
	if resource == "" {
		resource = ctx.Resource
	}
	detail := "attached"
	if ctx.DetailedContext != "" {
		detail = "attached: YAML/events/logs"
	}
	return fmt.Sprintf("[#9ece6a]%s %s[-] [gray](%s)[-]", tview.Escape(resource), tview.Escape(target), detail)
}

func (a *App) applyAIChrome() {
	if a.aiMetaBar == nil {
		return
	}

	a.aiMx.RLock()
	client := a.aiClient
	turns := a.aiConversationTurns
	a.aiMx.RUnlock()

	ctx := a.getAIPromptContext()
	modelLabel := "not configured"
	statusLabel := "[#f7768e]● offline[-]"
	toolsLabel := "[#f7768e]off[-]"
	if client != nil {
		modelParts := make([]string, 0, 2)
		if provider := client.GetProvider(); provider != "" {
			modelParts = append(modelParts, provider)
		}
		if model := client.GetModel(); model != "" {
			modelParts = append(modelParts, model)
		}
		if len(modelParts) > 0 {
			modelLabel = strings.Join(modelParts, "/")
		}
		if client.IsReady() {
			statusLabel = "[#9ece6a]● online[-]"
		}
		if client.SupportsTools() {
			toolsLabel = "[#9ece6a]on[-]"
		}
	}
	meta := fmt.Sprintf(
		"[#bb9af7::b]AI Assistant[-::-] %s [gray]|[-] [yellow]%s[-] [gray]|[-] tools %s [gray]|[-] turns [white]%d[-]\n[gray]Context:[-] %s [gray]|[-] /help /context /clear",
		statusLabel,
		tview.Escape(modelLabel),
		toolsLabel,
		turns,
		a.formatAIContextLabel(ctx),
	)
	a.aiMetaBar.SetText(meta)
	if strings.TrimSpace(a.aiStatusBar.GetText(false)) == "" {
		a.setAIStatus(a.readyAIStatusText())
	}
}

func (a *App) appendAIEscaped(text string) {
	if text == "" {
		return
	}
	fmt.Fprint(a.aiPanel, tview.Escape(text))
	a.aiPanel.ScrollToEnd()
}

func (a *App) appendAIMarkup(text string) {
	if text == "" {
		return
	}
	fmt.Fprint(a.aiPanel, text)
	a.aiPanel.ScrollToEnd()
}

func (a *App) appendAISystemSection(title, body string) {
	if strings.TrimSpace(a.aiPanel.GetText(false)) != "" {
		a.appendAIMarkup("\n")
	}
	a.appendAIMarkup(fmt.Sprintf("[#e0af68::b]%s[-::-]\n", tview.Escape(title)))
	a.appendAIEscaped(strings.TrimSpace(body))
	a.appendAIMarkup("\n")
}

func (a *App) startAITurn(question string, ctx aiPromptContext, mode string) {
	a.aiMx.Lock()
	a.aiConversationTurns++
	turn := a.aiConversationTurns
	a.aiMx.Unlock()

	if turn == 1 {
		a.aiPanel.Clear()
	} else {
		a.appendAIMarkup("\n\n[gray]────────────────────────────────────────[-]\n\n")
	}

	ts := time.Now().Format("15:04:05")
	modeLabel := "[#bb9af7]Chat[-]"
	if mode == "agentic" {
		modeLabel = "[#9ece6a]Agentic[-]"
	}
	a.appendAIMarkup(fmt.Sprintf("[#7aa2f7::b]You[-::-] [gray]%s[-]\n", ts))
	a.appendAIEscaped(question)
	a.appendAIMarkup("\n")
	a.appendAIMarkup(fmt.Sprintf("[gray]Context:[-] %s\n", a.formatAIContextLabel(ctx)))
	a.appendAIMarkup(fmt.Sprintf("\n[#9ece6a::b]Assistant[-::-] %s\n", modeLabel))
	if ctx.DetailedContext != "" {
		a.appendAIMarkup("[gray]Attached resource context (YAML, events, recent logs).[-]\n")
	}
}

func (a *App) appendAIToolExecution(toolName, command, result string, isError bool, toolType, toolServerName string) {
	titleColor := "[#9ece6a]"
	if isError {
		titleColor = "[#f7768e]"
	}
	label := toolName
	if toolType != "" {
		label = toolType + ":" + toolName
	}
	if toolServerName != "" {
		label += "@" + toolServerName
	}
	a.appendAIMarkup(fmt.Sprintf("\n%sTool[-] [white]%s[-]\n", titleColor, tview.Escape(label)))
	if strings.TrimSpace(command) != "" {
		a.appendAIMarkup("[gray]Command:[-] ")
		a.appendAIEscaped(trimAIBlock(command, 240))
		a.appendAIMarkup("\n")
	}
	a.appendAIMarkup("[gray]Result:[-] ")
	a.appendAIEscaped(summarizeAIToolResult(result))
	a.appendAIMarkup("\n")
}

func (a *App) addAIInputHistory(input string) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return
	}

	a.aiMx.Lock()
	defer a.aiMx.Unlock()

	if len(a.aiInputHistory) > 0 && a.aiInputHistory[len(a.aiInputHistory)-1] == trimmed {
		a.aiInputHistoryIdx = -1
		return
	}
	a.aiInputHistory = append(a.aiInputHistory, trimmed)
	if len(a.aiInputHistory) > maxCmdHistory {
		a.aiInputHistory = a.aiInputHistory[1:]
	}
	a.aiInputHistoryIdx = -1
}

func (a *App) recallAIInputHistory(direction int) string {
	a.aiMx.Lock()
	defer a.aiMx.Unlock()

	if len(a.aiInputHistory) == 0 {
		return ""
	}

	switch {
	case direction < 0:
		if a.aiInputHistoryIdx == -1 {
			a.aiInputHistoryIdx = len(a.aiInputHistory) - 1
		} else if a.aiInputHistoryIdx > 0 {
			a.aiInputHistoryIdx--
		}
	case direction > 0:
		if a.aiInputHistoryIdx >= 0 && a.aiInputHistoryIdx < len(a.aiInputHistory)-1 {
			a.aiInputHistoryIdx++
		} else {
			a.aiInputHistoryIdx = -1
			return ""
		}
	}

	if a.aiInputHistoryIdx < 0 || a.aiInputHistoryIdx >= len(a.aiInputHistory) {
		return ""
	}
	return a.aiInputHistory[a.aiInputHistoryIdx]
}

func (a *App) showAIContextPreview() {
	a.startLoading()
	defer a.stopLoading()

	a.QueueUpdateDraw(func() {
		a.setAIStatus("[cyan]Inspecting current selection...[-]")
	})

	ctx := a.loadDetailedAIContext(a.getAIPromptContext())
	candidate := a.currentAISelectionCandidate()
	body := fmt.Sprintf("View: %s\nNamespace: %s\n", ctx.Resource, ctx.Namespace)
	if ctx.Namespace == "" {
		body = fmt.Sprintf("View: %s\nNamespace: all namespaces\n", ctx.Resource)
	}
	if ctx.SelectedName != "" {
		body += fmt.Sprintf("Attached resource: %s\n", ctx.SelectedResource)
		body += fmt.Sprintf("Selected: %s\n", ctx.SelectedName)
		if ctx.SelectedNamespace != "" {
			body += fmt.Sprintf("Selected namespace: %s\n", ctx.SelectedNamespace)
		}
		if ctx.SelectedSummary != "" {
			body += "\nRow summary:\n" + ctx.SelectedSummary + "\n"
		}
		if ctx.DetailedContext != "" {
			body += "\nAttached preview:\n" + trimAIBlock(ctx.DetailedContext, 1400)
		}
	} else if !candidate.IsZero() {
		body += fmt.Sprintf("\nSelected row available: %s %s\n", candidate.Resource, candidate.TargetLabel())
		if candidate.Summary != "" {
			body += "\nRow summary:\n" + candidate.Summary + "\n"
		}
		body += "\nThis row is not attached yet. Press Enter in the table while the AI panel is open to attach it."
	} else {
		body += "\nNo row selected. The AI will answer using the current resource view only."
	}

	a.QueueUpdateDraw(func() {
		a.appendAISystemSection("Context Preview", body)
		a.setAIStatus(a.readyAIStatusText())
		a.applyAIChrome()
	})
}

func (a *App) handleAICommand(input string) bool {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		return false
	}

	switch strings.TrimPrefix(strings.ToLower(fields[0]), "/") {
	case "clear", "new":
		a.QueueUpdateDraw(func() {
			a.resetAIConversation()
			a.applyAIChrome()
		})
	case "help":
		a.QueueUpdateDraw(func() {
			a.appendAISystemSection("AI Help", "Commands:\n/context  show the resource context that is currently attached\n/clear    reset the current transcript\n/new      start a fresh conversation\n/help     show this help\n\nTips:\n- Open the AI panel and press Enter on a table row to attach or detach it\n- Attached rows stay available to AI even if you move to another view\n- Up/Down recall previous prompts\n- Shift+Tab focuses transcript history\n- Tab returns to the prompt from transcript history\n- j/k or PgUp/PgDn scroll the transcript\n- g / G jump to the top or bottom of the transcript\n- Ctrl+E toggles the AI panel\n- Alt+H / Alt+L resize the AI panel\n- Alt+0 resets the AI panel width")
			a.setAIStatus(a.readyAIStatusText())
			a.applyAIChrome()
		})
	case "context":
		a.safeGo("ai-context-preview", a.showAIContextPreview)
	default:
		a.QueueUpdateDraw(func() {
			a.appendAISystemSection("Unknown Command", input)
			a.setAIStatus(a.readyAIStatusText())
		})
	}
	return true
}

// askAI sends a question to the AI and displays the response
func (a *App) askAI(question string) {
	a.startLoading()
	defer a.stopLoading()

	a.QueueUpdateDraw(func() {
		a.setAIStatus("[cyan]Collecting Kubernetes context...[-]")
	})

	promptCtx := a.loadDetailedAIContext(a.getAIPromptContext())
	prompt := buildAIPrompt(question, promptCtx)

	a.aiMx.RLock()
	client := a.aiClient
	a.aiMx.RUnlock()
	if client == nil || !client.IsReady() {
		a.QueueUpdateDraw(func() {
			a.startAITurn(question, promptCtx, "chat")
			a.appendAISystemSection("AI Unavailable", "Configure an LLM in ~/.config/k13d/config.yaml or open Settings (Shift+O) to connect a provider.")
			a.setAIStatus(a.readyAIStatusText())
			a.applyAIChrome()
		})
		return
	}

	mode := "chat"
	if client.SupportsTools() {
		mode = "agentic"
	}
	a.QueueUpdateDraw(func() {
		a.startAITurn(question, promptCtx, mode)
		a.setAIStatus("[cyan]Thinking...[-]")
		a.applyAIChrome()
	})

	ctx := a.getAppContext()
	var (
		fullResponse strings.Builder
		pending      strings.Builder
		pendingMu    sync.Mutex
		err          error
	)

	flushPending := func(force bool) {
		pendingMu.Lock()
		if pending.Len() == 0 {
			pendingMu.Unlock()
			return
		}
		if !force {
			now := time.Now().UnixNano()
			last := atomic.LoadInt64(&a.lastAIDraw)
			if now-last < 120_000_000 {
				pendingMu.Unlock()
				return
			}
			atomic.StoreInt64(&a.lastAIDraw, now)
		}
		delta := pending.String()
		pending.Reset()
		pendingMu.Unlock()

		a.QueueUpdateDraw(func() {
			a.appendAIEscaped(delta)
		})
	}

	streamCallback := func(chunk string) {
		fullResponse.WriteString(chunk)
		pendingMu.Lock()
		pending.WriteString(chunk)
		pendingMu.Unlock()
		flushPending(false)
	}

	if client.SupportsTools() {
		err = client.AskWithToolsAndExecution(ctx, prompt, streamCallback, func(toolName string, args string) bool {
			flushPending(true)

			var cmdArgs struct {
				Command   string `json:"command"`
				Namespace string `json:"namespace,omitempty"`
			}
			if err := parseJSON(args, &cmdArgs); err != nil {
				a.logger.Error("Failed to parse tool args", "error", err, "args", args)
			}

			fullCmd := ""
			switch toolName {
			case "kubectl":
				fullCmd = "kubectl " + cmdArgs.Command
				if cmdArgs.Namespace != "" && !strings.Contains(cmdArgs.Command, "-n ") {
					fullCmd = "kubectl -n " + cmdArgs.Namespace + " " + cmdArgs.Command
				}
			case "bash":
				fullCmd = cmdArgs.Command
			default:
				fullCmd = cmdArgs.Command
				if fullCmd == "" {
					fullCmd = args
				}
			}

			decision := a.evaluateAIToolDecision(toolName, fullCmd)
			if decision.Allowed && !decision.RequiresApproval {
				return true
			}

			if !decision.Allowed {
				a.QueueUpdateDraw(func() {
					a.appendAIMarkup("\n[red::b]Command Blocked[-::-]\n")
					a.appendAIMarkup("[gray]Command:[-] ")
					a.appendAIEscaped(trimAIBlock(fullCmd, 240))
					a.appendAIMarkup("\n")
					a.appendAIEscaped(decision.BlockReason)
					a.appendAIMarkup("\n")
					for _, warning := range decision.Warnings {
						a.appendAIMarkup("[red]-[-] ")
						a.appendAIEscaped(warning)
						a.appendAIMarkup("\n")
					}
					a.setAIStatus("[red]Command blocked by policy[-]")
				})
				return false
			}

			a.setToolCallState(toolName, args, fullCmd)
			a.QueueUpdateDraw(func() {
				a.showToolApprovalModal(toolName, fullCmd, decision)
				a.setAIStatus("[yellow]Awaiting approval[-] Y/Enter approve | N/Esc cancel")
			})

			select {
			case <-a.pendingToolApproval:
			default:
			}

			approvalTimeout := time.After(a.getToolApprovalTimeout())
			select {
			case approved := <-a.pendingToolApproval:
				if approved {
					a.QueueUpdateDraw(func() {
						a.setAIStatus("[cyan]Executing approved tool...[-]")
					})
				} else {
					a.QueueUpdateDraw(func() {
						a.setAIStatus("[yellow]Command cancelled[-]")
					})
				}
				return approved
			case <-approvalTimeout:
				a.clearToolCallState()
				a.QueueUpdateDraw(func() {
					a.closeToolApprovalModal()
					a.setAIStatus("[yellow]Approval timed out[-]")
				})
				return false
			case <-ctx.Done():
				a.clearToolCallState()
				a.QueueUpdateDraw(func() {
					a.closeToolApprovalModal()
				})
				return false
			}
		}, func(toolName string, command string, result string, isError bool, toolType string, toolServerName string) {
			a.QueueUpdateDraw(func() {
				a.appendAIToolExecution(toolName, command, result, isError, toolType, toolServerName)
				if isError {
					a.setAIStatus("[yellow]Tool finished with warnings[-]")
				} else {
					a.setAIStatus("[cyan]Tool completed[-]")
				}
			})
		})
	} else {
		err = client.Ask(ctx, prompt, streamCallback)
	}

	flushPending(true)

	if err != nil {
		a.QueueUpdateDraw(func() {
			a.appendAIMarkup("\n[red::b]Error[-::-]\n")
			a.appendAIEscaped(err.Error())
			a.appendAIMarkup("\n")
			a.setAIStatus(a.readyAIStatusText())
		})
		return
	}

	finalResponse := fullResponse.String()
	a.QueueUpdateDraw(func() {
		if finalResponse != "" && !strings.HasSuffix(finalResponse, "\n") {
			a.appendAIMarkup("\n")
		}
		a.setAIStatus(a.readyAIStatusText())
		a.applyAIChrome()
	})

	if !client.SupportsTools() {
		a.analyzeAndShowDecisions(finalResponse)
	}
}

// analyzeAndShowDecisions extracts commands from AI response and shows decision UI
func (a *App) analyzeAndShowDecisions(response string) {
	// Extract kubectl commands from response
	commands := ai.ExtractKubectlCommands(response)
	if len(commands) == 0 {
		return
	}

	// Analyze commands for safety
	a.aiMx.Lock()
	a.pendingDecisions = nil

	var hasDecisions bool
	for _, cmd := range commands {
		decision := a.evaluateAIToolDecision("kubectl", cmd)
		if !decision.Allowed {
			a.QueueUpdateDraw(func() {
				a.appendAIMarkup("\n[red::b]Command Blocked[-::-]\n")
				a.appendAIEscaped(cmd)
				a.appendAIMarkup("\n")
				a.appendAIEscaped(decision.BlockReason)
				a.appendAIMarkup("\n")
			})
			continue
		}
		if decision.RequiresApproval || (decision.Classification != nil && decision.Classification.IsDangerous) {
			hasDecisions = true
			a.pendingDecisions = append(a.pendingDecisions, PendingDecision{
				Command:     cmd,
				Description: getCommandDescription(cmd),
				IsDangerous: decision.Classification != nil && decision.Classification.IsDangerous,
				Warnings:    decision.Warnings,
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
		a.appendAIMarkup("\n[yellow::b]Decision Required[-::-]\n")

		for i, decision := range decisions {
			if decision.IsDangerous {
				a.appendAIMarkup(fmt.Sprintf("[red]⚠ [%d] DANGEROUS[-] ", i+1))
			} else {
				a.appendAIMarkup(fmt.Sprintf("[yellow]? [%d] Confirm[-] ", i+1))
			}
			a.appendAIEscaped(decision.Command)
			a.appendAIMarkup("\n")

			for _, warning := range decision.Warnings {
				a.appendAIMarkup("   [red]-[-] ")
				a.appendAIEscaped(warning)
				a.appendAIMarkup("\n")
			}
			a.appendAIMarkup("\n")
		}

		a.appendAIMarkup("[gray]Press 1-9 to execute, A to execute all, Esc to cancel[-]\n")
		a.setAIStatus("[yellow]Pending command approvals[-] 1-9 execute | A all | Esc cancel")
	})
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

	runtimeDecision := a.evaluateAIToolDecision("kubectl", decision.Command)
	if !runtimeDecision.Allowed {
		a.QueueUpdateDraw(func() {
			a.appendAIMarkup("\n[red::b]Command Blocked[-::-]\n")
			a.appendAIEscaped(decision.Command)
			a.appendAIMarkup("\n")
			a.appendAIEscaped(runtimeDecision.BlockReason)
			a.appendAIMarkup("\n")
			a.setAIStatus("[red]Command blocked by policy[-]")
		})
		return
	}

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

		a.appendAIMarkup("\n[yellow::b]Execution Result[-::-]\n")
		a.appendAIEscaped(decision.Command)
		a.appendAIMarkup("\n")
		a.appendAIEscaped(result)
		a.appendAIMarkup("\n")
		a.setAIStatus(a.readyAIStatusText())
	})

	// Refresh if it was a modifying command
	a.safeGo("refresh-after-execute", a.refresh)
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
				a.closeModal("confirm-all")
				if buttonLabel == "Execute All" {
					a.safeGo("doExecuteAll", a.doExecuteAll)
				}
			})
		modal.SetBackgroundColor(tcell.ColorDarkRed)
		a.showModal("confirm-all", modal, true)
	} else {
		a.safeGo("doExecuteAll", a.doExecuteAll)
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
		runtimeDecision := a.evaluateAIToolDecision("kubectl", decision.Command)
		if !runtimeDecision.Allowed {
			results.WriteString(fmt.Sprintf("\n[cyan]%s[white]\n", decision.Command))
			results.WriteString(fmt.Sprintf("[red]Blocked:[white] %s\n", runtimeDecision.BlockReason))
			continue
		}

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
		a.appendAIMarkup(results.String())
		a.setAIStatus(a.readyAIStatusText())
	})

	a.flashMsg(fmt.Sprintf("Executed %d commands", len(decisions)), false)
	a.safeGo("refresh-after-batch", a.refresh)
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
		a.closeToolApprovalModal()
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

// parseJSON is a helper to parse JSON arguments
func parseJSON(jsonStr string, v interface{}) error {
	return json.Unmarshal([]byte(jsonStr), v)
}
