package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/rivo/tview"
)

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
