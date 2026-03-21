package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// showSettings displays settings modal with LLM connection test and save functionality
func (a *App) showSettings() {
	if err := a.reloadConfigFromDisk(); err != nil {
		a.flashMsg(fmt.Sprintf("Failed to reload config: %v", err), true)
	}

	form := tview.NewForm()

	statusText := "[gray]●[white] LLM Status: Unknown"
	statusView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(statusText)
	statusView.SetBackgroundColor(tcell.ColorDefault)

	provider := a.config.LLM.Provider
	model := a.config.LLM.Model
	endpoint := a.config.LLM.Endpoint
	apiKey := ""
	hasAPIKey := a.config.LLM.APIKey != ""
	toolPolicy := effectiveUIToolApprovalPolicy(a.config.Authorization.ToolApproval)
	autoApproveReadOnly := toolPolicy.AutoApproveReadOnly
	requireApprovalForWrite := toolPolicy.RequireApprovalForWrite
	blockDangerous := toolPolicy.BlockDangerous
	requireApprovalForUnknown := toolPolicy.RequireApprovalForUnknown
	approvalTimeoutSeconds := toolPolicy.ApprovalTimeoutSeconds
	var infoView *tview.TextView

	updateInfoView := func() {
		if infoView == nil {
			return
		}
		currentAPIKey := hasAPIKey
		if apiKey != "" {
			currentAPIKey = true
		}
		infoView.SetText(
			buildLLMInfoText(provider, model, endpoint, currentAPIKey) + "\n" +
				buildToolApprovalInfoText(config.ToolApprovalPolicy{
					AutoApproveReadOnly:       autoApproveReadOnly,
					RequireApprovalForWrite:   requireApprovalForWrite,
					RequireApprovalForUnknown: requireApprovalForUnknown,
					BlockDangerous:            blockDangerous,
					ApprovalTimeoutSeconds:    approvalTimeoutSeconds,
				}),
		)
	}

	providers := []string{"openai", "ollama", "upstage", "gemini", "anthropic", "bedrock", "azopenai"}
	providerIndex := 0
	for i, p := range providers {
		if p == provider {
			providerIndex = i
			break
		}
	}

	form.AddDropDown("Provider", providers, providerIndex, func(option string, index int) {
		provider = option
		switch option {
		case "ollama":
			if endpoint == "" || endpoint == "https://api.openai.com/v1" {
				endpoint = "http://localhost:11434"
				if item := form.GetFormItemByLabel("Endpoint"); item != nil {
					if input, ok := item.(*tview.InputField); ok {
						input.SetText(endpoint)
					}
				}
			}
			if model == "" || model == "gpt-4" || model == "gpt-4o" || model == config.DefaultSolarModel {
				model = config.DefaultOllamaModel
				if item := form.GetFormItemByLabel("Model"); item != nil {
					if input, ok := item.(*tview.InputField); ok {
						input.SetText(model)
					}
				}
			}
		case "openai":
			if model == "" || model == config.DefaultOllamaModel || model == "llama3.2" || model == config.DefaultSolarModel {
				model = "gpt-4o"
				if item := form.GetFormItemByLabel("Model"); item != nil {
					if input, ok := item.(*tview.InputField); ok {
						input.SetText(model)
					}
				}
			}
		case "anthropic":
			if model == "" {
				model = "claude-sonnet-4-20250514"
				if item := form.GetFormItemByLabel("Model"); item != nil {
					if input, ok := item.(*tview.InputField); ok {
						input.SetText(model)
					}
				}
			}
		case "upstage":
			if endpoint == "" || endpoint == "https://api.openai.com/v1" {
				endpoint = "https://api.upstage.ai/v1"
				if item := form.GetFormItemByLabel("Endpoint"); item != nil {
					if input, ok := item.(*tview.InputField); ok {
						input.SetText(endpoint)
					}
				}
			}
			if model == "" || model == "gpt-4" || model == "gpt-4o" || model == config.DefaultOllamaModel || model == "llama3.2" {
				model = config.DefaultSolarModel
				if item := form.GetFormItemByLabel("Model"); item != nil {
					if input, ok := item.(*tview.InputField); ok {
						input.SetText(model)
					}
				}
			}
		}
		updateInfoView()
	})
	form.AddInputField("Model", model, 30, nil, func(text string) {
		model = text
		updateInfoView()
	})
	form.AddInputField("Endpoint", endpoint, 50, nil, func(text string) {
		endpoint = text
		updateInfoView()
	})
	form.AddPasswordField("API Key", "", 50, '*', func(text string) {
		apiKey = text
		updateInfoView()
	})
	form.AddCheckbox("Auto-approve Read-only", autoApproveReadOnly, func(checked bool) {
		autoApproveReadOnly = checked
		updateInfoView()
	})
	form.AddCheckbox("Require Write Approval", requireApprovalForWrite, func(checked bool) {
		requireApprovalForWrite = checked
		updateInfoView()
	})
	form.AddCheckbox("Block Dangerous", blockDangerous, func(checked bool) {
		blockDangerous = checked
		updateInfoView()
	})
	form.AddCheckbox("Require Unknown Approval", requireApprovalForUnknown, func(checked bool) {
		requireApprovalForUnknown = checked
		updateInfoView()
	})
	form.AddInputField("Approval Timeout (s)", strconv.Itoa(approvalTimeoutSeconds), 8, nil, func(text string) {
		value, err := strconv.Atoi(strings.TrimSpace(text))
		if err == nil && value > 0 {
			approvalTimeoutSeconds = value
		}
		updateInfoView()
	})

	infoView = tview.NewTextView().
		SetDynamicColors(true).
		SetText(buildLLMInfoText(provider, model, endpoint, hasAPIKey) + "\n" + buildToolApprovalInfoText(toolPolicy))
	infoView.SetBackgroundColor(tcell.ColorDefault)

	form.AddButton("Save", func() {
		statusView.SetText("[yellow]◐[white] Saving configuration...")
		a.QueueUpdateDraw(func() {})

		a.safeGo("saveConfig", func() {
			timeout := approvalTimeoutSeconds
			if timeout <= 0 {
				timeout = config.DefaultToolApprovalPolicy().ApprovalTimeoutSeconds
			}
			if timeout > 600 {
				timeout = 600
			}

			a.config.LLM.Provider = provider
			a.config.LLM.Model = model
			a.config.LLM.Endpoint = endpoint
			if apiKey != "" {
				a.config.LLM.APIKey = apiKey
				hasAPIKey = true
			}
			a.config.Authorization.ToolApproval = config.ToolApprovalPolicy{
				AutoApproveReadOnly:       autoApproveReadOnly,
				RequireApprovalForWrite:   requireApprovalForWrite,
				RequireApprovalForUnknown: requireApprovalForUnknown,
				BlockDangerous:            blockDangerous,
				BlockedPatterns:           append([]string(nil), a.config.Authorization.ToolApproval.BlockedPatterns...),
				ApprovalTimeoutSeconds:    timeout,
			}
			a.config.SyncActiveModelProfileFromLLM()

			if err := a.config.Save(); err != nil {
				a.QueueUpdateDraw(func() {
					statusView.SetText(fmt.Sprintf("[red]✗[white] Failed to save: %s", err))
				})
				return
			}

			newClient, err := ai.NewClient(&a.config.LLM)
			if err != nil {
				a.QueueUpdateDraw(func() {
					statusView.SetText(fmt.Sprintf("[yellow]●[white] Saved, but client init failed: %s", err))
				})
				return
			}
			a.aiMx.Lock()
			a.aiClient = newClient
			a.aiMx.Unlock()

			a.QueueUpdateDraw(func() {
				statusView.SetText("[green]●[white] Configuration saved! Press 'Test Connection' to verify")
				approvalTimeoutSeconds = timeout
				updateInfoView()
				a.updateHeader()
				a.applyAIChrome()
			})
		})
	})

	form.AddButton("Test", func() {
		statusView.SetText("[yellow]◐[white] Testing connection...")
		a.QueueUpdateDraw(func() {})

		a.safeGo("testConnection", func() {
			a.aiMx.RLock()
			testClient := a.aiClient
			a.aiMx.RUnlock()
			var resultText string
			if testClient == nil {
				resultText = "[red]✗[white] LLM Not Configured - Save settings first"
			} else {
				ctx, cancel := context.WithTimeout(a.getAppContext(), 30*time.Second)
				defer cancel()

				status := testClient.TestConnection(ctx)
				if status.Connected {
					resultText = fmt.Sprintf("[green]●[white] Connected! %s/%s (%dms)",
						status.Provider, status.Model, status.ResponseTime)
				} else {
					resultText = fmt.Sprintf("[red]✗[white] Failed: %s", status.Error)
					if status.Message != "" {
						resultText += "\n    [gray]" + status.Message + "[white]"
					}
					if provider == "ollama" && status.Error == "tool calling 모델이 필요함" {
						resultText += "\n    [yellow]" + ollamaModelToolsHint(model) + "[white]"
					}
				}
			}

			a.QueueUpdateDraw(func() {
				statusView.SetText(resultText)
			})
		})
	})

	form.AddButton("Close", func() {
		a.closeModal("settings")
		a.SetFocus(a.table)
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(infoView, 12, 0, false).
		AddItem(statusView, 2, 0, false).
		AddItem(form, 0, 1, true)

	flex.SetBorder(true).SetTitle(" Settings (Esc to close) ")
	flex.SetBackgroundColor(tcell.ColorDefault)

	a.showModal("settings", centered(flex, 88, 38), true)
	a.SetFocus(form)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			a.closeModal("settings")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})

	a.safeGo("initConfig-status", func() {
		a.aiMx.RLock()
		initClient := a.aiClient
		a.aiMx.RUnlock()
		var initialStatus string
		if initClient == nil {
			initialStatus = "[gray]●[white] LLM Not Configured - Enter settings and Save"
		} else if initClient.IsReady() {
			initialStatus = fmt.Sprintf("[yellow]●[white] LLM: %s/%s - Press 'Test' to verify",
				initClient.GetProvider(), initClient.GetModel())
		} else {
			initialStatus = "[gray]●[white] LLM Configuration Incomplete - Enter settings and Save"
		}
		a.QueueUpdateDraw(func() {
			statusView.SetText(initialStatus)
		})
	})
}
