package ui

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
	"github.com/cloudbro-kube-ai/k13d/pkg/ai/safety"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// askAI sends a question to the AI and displays the response
func (a *App) askAI(question string) {
	a.startLoading()
	defer a.stopLoading()

	// Preserve existing conversation history
	existingText := a.aiPanel.GetText(false)
	historyPrefix := ""
	if strings.TrimSpace(existingText) != "" {
		historyPrefix = existingText + "\n\n[gray]────────────────────────────────[white]\n\n"
	}

	// Show loading state
	a.QueueUpdateDraw(func() {
		a.aiPanel.SetText(historyPrefix + fmt.Sprintf("[yellow]Question:[white] %s\n\n[gray]Thinking...", question))
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
	ctx := a.getAppContext()
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
	a.aiMx.RLock()
	client := a.aiClient
	a.aiMx.RUnlock()
	if client == nil || !client.IsReady() {
		a.QueueUpdateDraw(func() {
			a.aiPanel.SetText(historyPrefix + fmt.Sprintf("[yellow]Q:[white] %s\n\n[red]AI is not available.[white]\n\nConfigure LLM in config file:\n[gray]~/.kube-ai-dashboard/config.yaml", question))
		})
		return
	}

	// Check if AI supports tool calling (agentic mode)
	var fullResponse strings.Builder
	var err error

	if client.SupportsTools() {
		// Use agentic mode with tool calling
		a.QueueUpdateDraw(func() {
			a.aiPanel.SetText(historyPrefix + fmt.Sprintf("[yellow]Q:[white] %s\n\n[cyan]🤖 Agentic Mode[white] - AI can execute kubectl commands\n\n[gray]Thinking...", question))
		})

		err = client.AskWithTools(ctx, prompt, func(chunk string) {
			fullResponse.WriteString(chunk)
			// Throttle streaming draws to reduce ghosting (50ms minimum interval)
			now := time.Now().UnixNano()
			last := atomic.LoadInt64(&a.lastAIDraw)
			if now-last < 50_000_000 {
				return // Skip this draw, next chunk will catch up
			}
			atomic.StoreInt64(&a.lastAIDraw, now)
			response := fullResponse.String()
			a.QueueUpdateDraw(func() {
				a.aiPanel.SetText(historyPrefix + fmt.Sprintf("[yellow]Q:[white] %s\n\n[cyan]🤖 Agentic Mode[white]\n\n[green]A:[white] %s", question, response))
			})
		}, func(toolName string, args string) bool {
			// Tool approval callback - kubectl-ai style Decision Required
			a.logger.Info("Tool callback invoked", "tool", toolName, "args", args)

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
			classification := safety.Classify(fullCmd)

			// Read-only commands: auto-approve
			if classification.IsReadOnly {
				return true
			}

			// Store current tool info for approval (deadlock-safe)
			a.setToolCallState(toolName, args, fullCmd)

			// Show Decision Required UI
			a.QueueUpdateDraw(func() {
				var sb strings.Builder
				sb.WriteString(historyPrefix)
				sb.WriteString(fmt.Sprintf("[yellow]Q:[white] %s\n\n", question))
				sb.WriteString(fullResponse.String())
				sb.WriteString("\n\n[yellow::b]━━━ DECISION REQUIRED ━━━[white::-]\n\n")

				if classification.IsDangerous {
					sb.WriteString("[red]⚠ DANGEROUS COMMAND[white]\n")
				} else if classification.Category == "write" {
					sb.WriteString("[yellow]? WRITE OPERATION[white]\n")
				} else {
					sb.WriteString("[gray]? COMMAND APPROVAL[white]\n")
				}

				sb.WriteString(fmt.Sprintf("\n[cyan]%s[white]\n\n", fullCmd))

				for _, w := range classification.Warnings {
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

			// Wait for approval with 5-minute timeout
			approvalTimeout := time.After(5 * time.Minute)
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
			case <-approvalTimeout:
				a.clearToolCallState()
				a.QueueUpdateDraw(func() {
					currentText := a.aiPanel.GetText(false)
					a.aiPanel.SetText(currentText + "\n\n[yellow]⏰ Approval timed out (5 min)[white]")
				})
				return false
			case <-ctx.Done():
				return false
			}
		})
	} else {
		// Fallback to regular streaming
		err = client.Ask(ctx, prompt, func(chunk string) {
			fullResponse.WriteString(chunk)
			// Throttle streaming draws to reduce ghosting (50ms minimum interval)
			now := time.Now().UnixNano()
			last := atomic.LoadInt64(&a.lastAIDraw)
			if now-last < 50_000_000 {
				return // Skip this draw, next chunk will catch up
			}
			atomic.StoreInt64(&a.lastAIDraw, now)
			response := fullResponse.String()
			a.QueueUpdateDraw(func() {
				a.aiPanel.SetText(historyPrefix + fmt.Sprintf("[yellow]Q:[white] %s\n\n[green]A:[white] %s", question, response))
			})
		})
	}

	// Ensure final response is always drawn (in case last chunk was throttled)
	if err == nil {
		finalResponse := fullResponse.String()
		if finalResponse != "" {
			a.QueueUpdateDraw(func() {
				prefix := "[cyan]🤖 Agentic Mode[white]\n\n"
				if !client.SupportsTools() {
					prefix = ""
				}
				a.aiPanel.SetText(historyPrefix + fmt.Sprintf("[yellow]Q:[white] %s\n\n%s[green]A:[white] %s", question, prefix, finalResponse))
			})
		}
	}

	if err != nil {
		a.QueueUpdateDraw(func() {
			a.aiPanel.SetText(historyPrefix + fmt.Sprintf("[yellow]Q:[white] %s\n\n[red]Error:[white] %v", question, err))
		})
		return
	}

	// After response complete, analyze for commands that need approval (fallback mode)
	if !client.SupportsTools() {
		finalResponse := fullResponse.String()
		a.analyzeAndShowDecisions(question, finalResponse)
	}
}

// analyzeAndShowDecisions extracts commands from AI response and shows decision UI
func (a *App) analyzeAndShowDecisions(question, response string) {
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
		classification := safety.Classify(cmd)
		if classification.RequiresApproval || classification.IsDangerous {
			hasDecisions = true
			a.pendingDecisions = append(a.pendingDecisions, PendingDecision{
				Command:     cmd,
				Description: getCommandDescription(cmd),
				IsDangerous: classification.IsDangerous,
				Warnings:    classification.Warnings,
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
