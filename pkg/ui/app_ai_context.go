package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	return "[gray]Ready[-] Enter send | Shift+Tab history | Esc close | Up/Down history | Alt+H/L resize | Alt+F full | /help"
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
