package ui

import (
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

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
			a.appendAISystemSection("AI Help", "Commands:\n/context  show the resource context that is currently attached\n/clear    reset the current transcript\n/new      start a fresh conversation\n/help     show this help\n\nTips:\n- Open the AI panel and press Enter on a table row to attach or detach it\n- Attached rows stay available to AI even if you move to another view\n- Up/Down recall previous prompts\n- Shift+Tab focuses transcript history\n- Tab returns to the prompt from transcript history\n- j/k or PgUp/PgDn scroll the transcript\n- g / G jump to the top or bottom of the transcript\n- Ctrl+E toggles the AI panel\n- Alt+H / Alt+L resize the AI panel\n- Alt+F toggles AI full size\n- Alt+0 resets the AI panel width")
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
	prompt = a.buildAIConversationPrompt(prompt)

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
	a.addAIConversationMessage("user", question)

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
	a.addAIConversationMessage("assistant", finalResponse)
}

// parseJSON is a helper to parse JSON arguments
func parseJSON(jsonStr string, v interface{}) error {
	return json.Unmarshal([]byte(jsonStr), v)
}
