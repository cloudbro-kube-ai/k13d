package ui

import (
	"fmt"
	"os/exec"
	"strings"
	"sync/atomic"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
	"github.com/cloudbro-kube-ai/k13d/pkg/ai/safety"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (a *App) analyzeAndShowDecisions(response string) {
	commands := ai.ExtractKubectlCommands(response)
	if len(commands) == 0 {
		return
	}

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

	decisions := make([]PendingDecision, len(a.pendingDecisions))
	copy(decisions, a.pendingDecisions)
	a.aiMx.Unlock()

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

func (a *App) executeDecision(idx int) {
	a.aiMx.Lock()
	if idx < 0 || idx >= len(a.pendingDecisions) {
		a.aiMx.Unlock()
		return
	}

	decision := a.pendingDecisions[idx]
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

	output, err := a.runApprovedKubectlCommand(decision.Command)

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

	a.safeGo("refresh-after-execute", a.refresh)
}

func (a *App) executeAllDecisions() {
	a.aiMx.RLock()
	if len(a.pendingDecisions) == 0 {
		a.aiMx.RUnlock()
		return
	}

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
		return
	}

	a.safeGo("doExecuteAll", a.doExecuteAll)
}

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

		output, err := a.runApprovedKubectlCommand(decision.Command)

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

func parseApprovedKubectlCommand(command string) ([]string, error) {
	parsed := safety.ParseCommand(strings.TrimSpace(command))
	if parsed == nil {
		return nil, fmt.Errorf("invalid command")
	}
	if parsed.ParseError != nil {
		return nil, fmt.Errorf("invalid kubectl command: %w", parsed.ParseError)
	}
	if parsed.Program != "kubectl" {
		return nil, fmt.Errorf("only kubectl commands can be executed from AI decisions")
	}
	if parsed.IsPiped || parsed.IsChained || parsed.HasRedirect {
		return nil, fmt.Errorf("shell features are not allowed in AI decision execution")
	}
	if len(parsed.Args) == 0 {
		return nil, fmt.Errorf("kubectl command is missing arguments")
	}
	return append([]string(nil), parsed.Args...), nil
}

func (a *App) runApprovedKubectlCommand(command string) ([]byte, error) {
	args, err := parseApprovedKubectlCommand(command)
	if err != nil {
		return nil, err
	}

	// #nosec G204 -- args are parsed from an already-approved kubectl command and shell features are rejected above.
	cmd := exec.CommandContext(a.getAppContext(), "kubectl", args...)
	return cmd.CombinedOutput()
}

func (a *App) approveToolCall(approved bool) {
	select {
	case a.pendingToolApproval <- approved:
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
	}
}

func (a *App) setToolCallState(name, args, command string) {
	a.aiMx.Lock()
	a.currentToolCallInfo.Name = name
	a.currentToolCallInfo.Args = args
	a.currentToolCallInfo.Command = command
	a.aiMx.Unlock()
	atomic.StoreInt32(&a.hasToolCall, 1)
}

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

func (a *App) clearPendingDecisions() {
	a.aiMx.Lock()
	hadDecisions := len(a.pendingDecisions) > 0
	a.pendingDecisions = nil
	a.aiMx.Unlock()

	if hadDecisions {
		a.flashMsg("Cancelled pending commands", false)
	}
}

func getCommandDescription(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) < 2 {
		return cmd
	}

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
